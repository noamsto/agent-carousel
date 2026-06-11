# agent-carousel: D2 diagrams in the carousel

**Date:** 2026-06-11
**Status:** Design approved, pending spec review
**Origin:** Brainstormed out of the D2 non-goal in `docs/2026-06-10-design.md` (§Non-Goals 1).

## Summary

Let the agent draw diagrams that appear in the carousel. When a diagram would
clarify an explanation, the agent emits a ```` ```d2 ```` fenced block in its
reply; a `Stop` hook renders it **browser-free** (`d2 → svg → resvg → png`) into
the per-pane manifest, and the carousel shows it like any other image.

Two behavioral decisions shape the feature:

- **Agent-proactive** — the agent decides when a diagram helps and emits one
  unprompted (not gated behind a user request or a slash command). This requires
  the "when to diagram" guidance to be *ambient* (in context every turn), so it
  is injected by a `SessionStart` hook — but only when a carousel host is
  present (see §Host-gating).
- **Auto-open once per session** — the first diagram of a session opens the
  carousel so the user notices it exists; after that, diagrams render to the
  manifest silently (the carousel refreshes if open, and is never force-opened
  again).

The manifest JSONL stays the stable interface — diagrams are appended as
ordinary `type:"image"` entries, so the viewer needs **zero display changes**.

## Goals

- The agent proactively emits `d2` blocks where a picture beats prose
  (architecture, data flow, state machines, pipelines, entity relationships).
- A `Stop` hook renders those blocks to PNG browser-free and appends them to the
  manifest; the carousel displays them with no viewer-display changes.
- The carousel auto-opens **once per session** on the first diagram, then leaves
  open/closed state under user control.
- The proactive nudge loads only where it can pay off (tmux or kitty-graphics
  host present).

## Non-Goals (explicit, to prevent scope creep)

1. **Codebase scanning / infra-from-code generation.** Tools like
   `heathdutton/claude-d2-diagrams` scan Terraform/K8s/Docker to *generate*
   architecture docs as committed files. Out of scope — this feature only
   renders a diagram the agent is *already drawing* in its reply.
2. **Mermaid for GitHub docs.** GitHub renders Mermaid inline but not D2; D2
   renders browser-free for the terminal but Mermaid's renderer needs Chromium.
   These are different surfaces with a separate authoring guideline ("emit D2 for
   the carousel, Mermaid for GitHub docs") — not part of this skill, and there is
   **no** Mermaid→D2 transpilation step (no reliable converter exists; the agent
   emits the right language per surface).
3. **Animated diagrams.** The carousel rasterizes `svg → png` and displays a
   static frame (the kitty *protocol* supports animation, but the viewer
   transmits PNG via `f=100` and decodes only the first frame). D2 "animated
   traffic flow" is SVG/CSS animation and dies on rasterization.
4. **D2 syntax auto-fix.** On a parse error the hook skips and logs (see §Error
   handling). We borrow the *validate* instinct, not the auto-fix machinery.

## Architecture

### Three components

**(a) Ambient guidance — `SessionStart` hook.**
Injects the "when to diagram" guidance into context every session (via the
hook's stdout / `additionalContext`), gated to hosts where the carousel can
render (see §Host-gating). The full guidance
(~10 lines) lives here so the agent's judgment about *when* to diagram is
reliable, not just a pointer:

- Emit a ```` ```d2 ```` block when a diagram clarifies: architecture, data
  flow, state machines, pipelines, entity relationships.
- Do **not** diagram trivial/linear/one-step things; one diagram per concept;
  prose stays primary — a diagram supplements, never replaces, the explanation.

The detailed d2 dialect/cheatsheet stays in an on-demand skill body (loaded only
when actually drawing) to keep the ambient footprint small.

**(b) Render hook — `Stop` hook (`adapters/claude-code/plugin/scripts/diagrams.sh`).**
Fires when the agent's turn ends:

1. Read `transcript_path` from the Stop payload; extract **only the most recent
   assistant turn** (not the whole cumulative transcript).
2. **Fast-bail:** `grep` for a ```` ```d2 ```` fence first; exit immediately if
   none — the hook runs on *every* turn, so the no-diagram path must be
   microseconds with no JSON parsing.
3. Derive the manifest key with the **exact same logic as `images.sh`**:
   `${TMUX_PANE:-${CLAUDE_CODE_SESSION_ID}}`, strip a leading `%`, and apply the
   same `^[A-Za-z0-9_@:.-]+$` traversal guard — otherwise diagrams land in a
   different manifest than the captured images and the carousel splits in two.
4. For each `d2` block: content-hash the source → `<hash>.png`. **Render** only
   if that file is absent (an identical re-emitted diagram is a no-op; an edited
   diagram is a new hash). Render browser-free: `d2 - <hash>.svg` then
   `resvg <hash>.svg <hash>.png` — never `d2 ... .png` (that shells to Chromium).
5. **Append** `{type:"image", path, source:"d2", ts, mtime}` to the per-pane
   manifest — guarded by a `path`-dedup check (mirroring `images.sh`'s
   `(path,mtime)` guard), *independent* of the render step, so a diagram missing
   from the manifest is re-added even when its PNG is already cached. The viewer
   decodes only `path` + `source` (`gallery_render.go:14`), so `type`/`ts`/`mtime`
   are inert decoration kept for manifest-format parity; `source:"d2"` records
   provenance.
6. If ≥1 new diagram rendered **and** this session has not auto-opened yet,
   call the viewer in `--ensure-open` mode and drop the once-per-session marker
   (see §Auto-open).

**Open risk — transcript parsing is a new pattern.** Nothing in agent-carousel
or lazytmux reads `transcript_path` today; capture is `PostToolUse`
(`tool_input`/`tool_response`). The implementation plan must first verify the
Stop payload's `transcript_path`, the JSONL record shape, and that the final
assistant turn is already flushed when `Stop` fires. **Lower-risk alternative if
that proves brittle:** have the skill instruct the agent to `Write` the diagram
to a `*.d2` file and render it from a `PostToolUse` hook — reusing the proven
hook surface and `cwd`-relative path logic, at the cost of a visible `Write` tool
call instead of a plain chat code block. Decide in the plan; default to the
`Stop` approach if transcript parsing checks out.

**(c) Viewer change — `--ensure-open` on `scripts/tmux-claude-images.sh`.**
Today the script *toggles* (an existing viewer is killed). Add an
`--ensure-open` flag that opens-if-closed and **no-ops if already open** (skips
the kill branch in `launch_tmux`/`launch_kitty`). Manual toggle behavior is
unchanged; only the hook uses `--ensure-open`.

### Storage: diagrams are primary artifacts, not cache

Rendered diagram PNG files live in **`${AGENT_CAROUSEL_DIR:-…}/images/diagrams/`** —
durable, alongside the manifest. They are **not** placed in the transcode cache
(`agent-carousel-imgcache`): that cache holds *derived* artifacts regenerable
from a source image, so it is safely evictable. A diagram PNG is the *only* copy
of its rendered output (the d2 source lives only in the transcript), so cache
eviction would leave dangling manifest paths and gaps in the carousel.

### Data flow

```
agent turn ends
  → Stop hook (diagrams.sh)
    → read last assistant turn → grep ```d2 (bail if none)
    → for each block: hash → d2→svg→png → images/diagrams/<hash>.png
    → append {type:"image", source:"d2", …} to manifest
    → first diagram this session? → tmux-claude-images --ensure-open + touch marker
  → carousel refreshes → diagram appears
```

## Decisions

### Host-gating (#3 from review)

The `SessionStart` injection is guarded by
`[[ -n $TMUX || -n $KITTY_LISTEN_ON ]]`. Without tmux or a kitty-graphics host
there is nowhere to display a diagram, so the nudge would be pure context waste
and chat clutter. Gating to a present host means the proactive bias appears only
where it can pay off — and a developer who never uses tmux/kitty never sees it.
No opt-out flag for now (YAGNI); add an env kill-switch later if the nudge proves
noisy *even with a host*.

### Auto-open once per session (#4 from review)

The first diagram of a session opens the carousel (so the user discovers it);
subsequent diagrams render to the manifest only (refresh-if-open, never
force-open). This avoids the nag where the hook reopens a carousel the user
deliberately closed. Mechanism: a per-key marker file
`${AGENT_CAROUSEL_DIR:-…}/images/<key>.opened`. First render with no marker →
`--ensure-open` + touch marker; later renders skip the open. No viewer changes
and no need to distinguish *how* the carousel was closed — which the stricter
"dismiss sentinel" alternative would have required (the Go viewer writing a
marker on `q`-quit). New diagrams still land in the manifest and appear the
moment the user reopens. (Minor: `.opened` markers are not garbage-collected;
they share the cleanup story of the per-key manifests themselves, and a reused
tmux pane id at most suppresses one auto-open — acceptable.)

## Error handling (define errors out of existence)

- **`d2` or `resvg` not on PATH** → hook no-ops silently, matching the viewer's
  "not installed → do nothing" convention.
- **Malformed d2** (`d2` exits non-zero) → skip that block, append the error to
  `images/diagrams/render-errors.log` (one line: timestamp + hash + stderr),
  **no manifest entry**. Never inject a broken image.
- **No tmux/kitty host** → `--ensure-open` no-ops, same as the toggle today.

## Known trade-offs (accepted)

- **Raw `d2` source is visible in chat.** The hook needs the block in the
  transcript, so the fenced source appears in the reply while the *picture* is in
  the carousel pane. Inherent to proactive + Stop-hook; mitigated by keeping
  diagrams small and prose-first.
- **Render latency.** The diagram appears a beat after the text, once the Stop
  hook runs and the carousel refreshes.
- **Two new dependencies.** `d2` and `resvg` join `jq`/`tmux`/`kitty`. The
  viewer decodes raster only (no SVG path in `gallery_cache.go`), so `resvg` is
  mandatory for the browser-free pipeline. Both must be packaged in the flake;
  absent → graceful no-op.

## Testing

`bats` tests mirroring `tests/adapter.bats`, with fixture transcripts:

- valid d2 block → asserts manifest entry + cached PNG under `images/diagrams/`;
- duplicate block (same source) → asserts no second entry;
- malformed block → asserts skip + sidecar log, no manifest entry;
- `d2` absent on PATH → asserts clean no-op;
- fast-bail: transcript with no `d2` fence → asserts no JSON parsing / no entry;
- `--ensure-open` against an already-open pane → asserts it does **not** kill it;
- once-per-session: second diagram with marker present → asserts no second open.

## Files

| Path | Change |
|------|--------|
| `adapters/claude-code/plugin/scripts/diagrams.sh` | **new** — `Stop` hook: scan last turn, render `d2→svg→png`, append manifest, once-per-session ensure-open. |
| `adapters/claude-code/plugin/scripts/diagram-guidance.sh` | **new** — `SessionStart` hook: host-gated ambient nudge. |
| `adapters/claude-code/plugin/hooks/hooks.json` | add `Stop` + `SessionStart` hook registrations. |
| `adapters/claude-code/plugin/skills/diagrams/SKILL.md` | **new** — on-demand d2 dialect/cheatsheet (sibling of `image-gallery`). |
| `scripts/tmux-claude-images.sh` | add `--ensure-open` (open-if-closed; no kill). |
| `flake.nix` | package `d2` + `resvg` into the plugin/devShell runtime. |
| `tests/diagrams.bats` | **new** — render/dedup/error/once-per-session tests. |
