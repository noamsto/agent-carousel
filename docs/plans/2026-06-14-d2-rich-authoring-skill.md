# D2 Rich Authoring Skill (Phase 1) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite `skills/diagrams/SKILL.md` into a single rich, beautiful, *render-tested* D2 authoring reference so the agent produces legible, well-styled diagrams by default.

**Architecture:** Keep the skill a single file loaded on demand. Add a bats harness that extracts every ```d2 fenced block from SKILL.md and renders it through the real Phase 0 pipeline (`d2 → d2-fix-fonts.sh → resvg` with the bundled font dir), asserting each compiles, resolves all fonts, and is single-board. Then rewrite SKILL.md so every example passes that harness — the doc can never ship syntax that doesn't render.

**Tech Stack:** Markdown (skill), `d2` 0.7.x, `resvg`, bats, Nix devShell (provides d2/resvg + `AGENT_CAROUSEL_D2_FONT{,_DIR}`).

**Spec:** `docs/2026-06-14-d2-diagrams-rich-and-legible-design.md` (Phase 1 section).

**Base:** branch `feat/d2-rich-authoring-skill` off `main` @ `267cbff` (Phase 0 merged: font-fix + bundled Source Sans 3 are present).

---

## Scope decisions (resolved before planning)

- **Single-board only.** `steps`/`scenarios`/`layers` produce multiple boards (a directory of SVGs) and cannot render to the carousel's single PNG today. They are **out of scope** — deferred to a coherence follow-up after the step-group-navigation lane lands multi-board rendering in `diagrams.sh`. The harness in Task 1 actively rejects multi-board examples so none sneak in.
- **Role palette is mode-agnostic.** The `.d2` author cannot know whether the hook renders the light (`105`) or dark (`200`) theme, so explicit `fill`s would clash on one of them. House-style role `classes` therefore distinguish roles by **`stroke` color + shape**, leaving fill to the theme. This resolves the spec's "role palette vs dark mode" trade-off.
- **Coherence, not collision:** the `docs/d2-diagram-navigation` lane owns viewer navigation/zoom; this task only touches `skills/diagrams/SKILL.md` + a new test. No shared files.

## File Structure

| File | Responsibility |
|---|---|
| `tests/skill-examples-render.bats` | **new** — extract ```d2 blocks from SKILL.md, render each via the Phase 0 pipeline, assert compile + zero `No match` + single-board. |
| `tests/lib/extract-d2-blocks.sh` | **new** — tiny helper: print each ```d2 fenced block from a markdown file as a NUL-separated stream (testable, reused by the bats file). |
| `adapters/claude-code/plugin/skills/diagrams/SKILL.md` | **rewrite** — the rich authoring reference. |

---

## Task 1: Render-test harness for SKILL.md examples

**Files:**
- Create: `tests/lib/extract-d2-blocks.sh`
- Create: `tests/skill-examples-render.bats`

- [ ] **Step 1: Write the extractor helper**

Create `tests/lib/extract-d2-blocks.sh`:

```bash
#!/usr/bin/env bash
# Print each ```d2 fenced code block from a markdown file, separated by a NUL
# byte. Used by the skill-examples render test to render every example.
set -euo pipefail

md="$1"
awk '
  /^```d2[[:space:]]*$/ { inblock=1; next }
  /^```[[:space:]]*$/   { if (inblock) { printf "%s", "\0"; inblock=0 }; next }
  inblock               { print }
'  "$md"
```

- [ ] **Step 2: Write the failing test**

Create `tests/skill-examples-render.bats`:

```bash
#!/usr/bin/env bats
# Every ```d2 example in SKILL.md must compile, render text (0 "No match"),
# and be single-board. Skips when d2/resvg are unavailable.

setup() {
	ROOT="$(dirname "$BATS_TEST_DIRNAME")"
	SKILL="$ROOT/adapters/claude-code/plugin/skills/diagrams/SKILL.md"
	FIX="$ROOT/adapters/claude-code/plugin/scripts/d2-fix-fonts.sh"
	EXTRACT="$ROOT/tests/lib/extract-d2-blocks.sh"
}

@test "every d2 example in SKILL.md compiles, renders text, and is single-board" {
	command -v d2 >/dev/null || skip "d2 not installed"
	command -v resvg >/dev/null || skip "resvg not installed"

	mapfile -d '' -t blocks < <(bash "$EXTRACT" "$SKILL")
	[ "${#blocks[@]}" -ge 1 ] || { echo "no d2 examples found in SKILL.md"; return 1; }

	local i=0
	for block in "${blocks[@]}"; do
		i=$((i + 1))
		local d2f="$BATS_TEST_TMPDIR/ex$i.d2" out="$BATS_TEST_TMPDIR/ex$i.svg"
		printf '%s' "$block" >"$d2f"

		run d2 "$d2f" "$out"
		[ "$status" -eq 0 ] || { echo "example $i failed to compile: $output"; return 1; }
		# Multi-board (steps/scenarios/layers) makes d2 emit a directory, not a file.
		[ -f "$out" ] || { echo "example $i produced multiple boards (not single-board): $d2f"; return 1; }

		bash "$FIX" "$out"
		args=()
		[[ -n ${AGENT_CAROUSEL_D2_FONT_DIR:-} ]] && args=(--skip-system-fonts --use-fonts-dir "$AGENT_CAROUSEL_D2_FONT_DIR")
		run bash -c 'resvg "$@" 2>&1' _ "${args[@]}" "$out" "$BATS_TEST_TMPDIR/ex$i.png"
		[ "$status" -eq 0 ] || { echo "example $i resvg failed: $output"; return 1; }
		[[ $output != *"No match for font-family"* ]] || { echo "example $i has unresolved fonts: $output"; return 1; }
	done
}
```

- [ ] **Step 3: Run the test against the CURRENT SKILL.md, verify it passes**

Run: `nix develop -c bats tests/skill-examples-render.bats`
Expected: PASS — the current SKILL.md's existing ```d2 examples all compile/render. (If it can't find blocks, the extractor regex is wrong — fix before proceeding.) This validates the harness on known-good input before the rewrite.

- [ ] **Step 4: Shellcheck the helper**

Run: `shellcheck tests/lib/extract-d2-blocks.sh`
Expected: no warnings.

- [ ] **Step 5: Commit**

```bash
git add tests/lib/extract-d2-blocks.sh tests/skill-examples-render.bats
git commit -m "test(diagrams): render-test every d2 example in the skill"
```

---

## Task 2: Rewrite SKILL.md as the rich authoring reference

**Files:**
- Modify (rewrite): `adapters/claude-code/plugin/skills/diagrams/SKILL.md`
- Gated by: `tests/skill-examples-render.bats` (Task 1)

This is a writing-craft task — keep one coherent voice, lean prose, and make every example beautiful *and* renderable. Target ~250–350 lines.

**Critical authoring constraint (the harness enforces it):** every ```d2 fenced block must be a **complete, self-contained, compilable single-board diagram** — the Task 1 harness renders all of them. This is a feature, not a burden: it makes examples copy-paste-ready. For a one-liner that needs context (e.g. an FK edge), include the minimal surrounding declarations so it compiles. For non-runnable syntax notes (e.g. "connections: `->` directed"), use inline code (backticks) or a non-`d2` fence, NOT a ```d2 block. A bare `vars`+`classes` header compiles (empty diagram is valid), so the house-style header block is fine as ```d2.

- [ ] **Step 1: Write the new SKILL.md**

Replace the file with these sections (keep frontmatter `name: diagrams`; sharpen `description`). Use REAL, verified syntax (all constructs below were compile-checked against d2 0.7.x):

**Required sections, in order:**

1. **Frontmatter** — `name: diagrams`; description covering when to draw + that it produces beautiful, legible D2.
2. **When to draw / when not** — keep the existing judgment (architecture, data flow, state machines, pipelines, ERDs; not trivial/linear; one diagram per concept; prose stays primary), compressed.
3. **Where to write** — keep (scratch dir from SessionStart guidance; never write `.d2` into the project), compressed.
4. **House style (beautiful-by-default)** — the copy-paste header every diagram starts with, and the role classes. The hook sets the theme by light/dark mode, so the file sets only sketch + spacing:
   ```d2
   vars: {
     d2-config: {
       sketch: true
       pad: 16
     }
   }
   # Role classes distinguish by stroke + shape (NOT fill) so they read on both
   # the light and dark theme the carousel may render.
   classes: {
     svc:   { style: { stroke: "#1565C0"; stroke-width: 2 } }
     store: { shape: cylinder; style: { stroke: "#2E7D32"; stroke-width: 2 } }
     ext:   { style: { stroke: "#E65100"; stroke-width: 2; stroke-dash: 3 } }
   }
   ```
5. **Flow direction** — default `direction: right` (horizontal) so the diagram fills the carousel's landscape preview; use `direction: down` only for inherently-tall structures (sequence diagrams, deep trees); prefer grouping containers over a single long thin chain. (Coordinates with — does not duplicate — the viewer navigation lane.)
6. **Core syntax** — shapes, connections (`->` directed, `<->` bidirectional, `--` undirected), labels, nested containers with `parent.child`. Inline map keys separate with `;` or newlines (NEVER commas — `theme-id: 0, sketch: true` is a parse error).
7. **Rich constructs** (each with a short, verified example):
   - **Grouping / containers** — `name: { ... }`.
   - **ERD via `sql_table`** —
     ```d2
     users: {
       shape: sql_table
       id: int { constraint: primary_key }
       email: varchar { constraint: unique }
     }
     posts: {
       shape: sql_table
       id: int { constraint: primary_key }
       user_id: int { constraint: foreign_key }
     }
     posts.user_id -> users.id
     ```
     Note `constraint`: `primary_key` (PK), `foreign_key` (FK), `unique` (UNQ). Add `layout-engine: elk` in `d2-config` for row-precise FK edges.
   - **Sequence diagram** —
     ```d2
     shape: sequence_diagram
     alice -> bob: request
     bob -> alice: response
     ```
   - **Classes** — `classes: { ... }` + `thing: { class: svc }` (defined in the house-style header).
   - **Icons** — `node: { icon: https://icons.terrastruct.com/...; shape: image }` (or `icon:` on a normal shape). Note: icon URLs are fetched at render; omit when offline.
   - **Styling vocab** — `style`: `stroke`, `stroke-width`, `stroke-dash`, `border-radius`, `shadow`, `font-color`, `fill` (use sparingly — see house style); connection styling (`style.stroke-dash`, label on edge).
   - **`near` for legends/captions** — `note: |md ... |\nnote.near: bottom-center`.
8. **Worked examples (3–4, each beautiful, copy-paste-ready, single-board, render-tested):**
   - Architecture with grouping + role classes (a Backend container with svc/store roles).
   - ERD (2–3 `sql_table`s with FK edges).
   - Sequence diagram OR state machine.
   - A pipeline (horizontal flow with labeled edges).
   Each must use the house-style header and pass the Task 1 harness.
9. **Aesthetic do / don't** — ≤~12 nodes; distinguish roles by stroke + shape, not a rainbow of fills; label every edge; one concept per diagram; `direction: right` unless inherently tall; let layout breathe (don't over-nest).
10. **Requirements** — `d2` + `resvg` on PATH; the hook renders browser-free.

**Do NOT include** `steps`/`scenarios`/`layers` examples — they are multi-board and out of scope (the harness will reject them).

- [ ] **Step 2: Run the render harness — every example must pass**

Run: `nix develop -c bats tests/skill-examples-render.bats`
Expected: PASS. If any example fails to compile / render / is multi-board, fix that example (or the prose around it) until green. Iterate here — this is the core quality gate.

- [ ] **Step 3: Run the full suite + sanity-check the doc**

Run: `nix develop -c bats tests/`
Expected: all pass.
Then: `wc -l adapters/claude-code/plugin/skills/diagrams/SKILL.md` (sanity: ~250–350 lines, not ballooned), and re-read the file once for voice/lean-prose/no-placeholder.

- [ ] **Step 4: Commit**

```bash
git add adapters/claude-code/plugin/skills/diagrams/SKILL.md
git commit -m "feat(diagrams): rich, beautiful, render-tested D2 authoring skill"
```

---

## Final verification

- [ ] `nix develop -c bats tests/` — all pass (harness + render + pre-existing).
- [ ] Manually skim SKILL.md: every section present, examples beautiful, prose lean, no `steps`/`scenarios`.

## Out of scope (tracked elsewhere)
- `steps`/`scenarios`/`layers` authoring guidance — coherence follow-up after the navigation lane lands multi-board rendering.
- Crisp vector zoom / step-group navigation — owned by the `docs/d2-diagram-navigation` + `feat/zoom-pan-crop-rect` lanes.
