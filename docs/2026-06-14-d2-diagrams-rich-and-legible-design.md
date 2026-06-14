# agent-carousel: rich, legible D2 diagrams

## Summary

The D2-diagrams feature (see `2026-06-11-d2-diagrams-design.md`) renders `.d2`
files into the carousel. Dogfooding it surfaced one shipping bug and several
quality gaps that, together, mean the diagrams are neither *worth looking at*
nor *worth drawing* yet:

1. **No text renders.** `d2` embeds fonts as `@font-face`; `resvg` does not
   support `@font-face`, so every label is dropped — diagrams ship as boxes
   with no words.
2. **The authoring skill is shallow.** `skills/diagrams/SKILL.md` teaches only
   boxes/arrows/containers — no ERDs, sequence diagrams, classes, theming,
   icons, flow-direction, or aesthetic guidance. Output is plain.
3. **The viewer under-serves diagrams.** Small diagrams render tiny (no
   upscale), there is no zoom/pan, and duplicate manifest entries show twice.

Goal: **diagrams worth drawing AND worth looking at.** Delivered in three
phases, each with its own implementation plan.

## Goals

- Text (incl. bold/italic) renders reliably, browser-free.
- The diagram theme follows the user's light/dark mode at render time.
- The authoring skill produces rich, beautiful, legible diagrams by default.
- The viewer treats diagrams as first-class (legible at rest, crisp when
  zoomed).

## Non-Goals

- Reintroducing the headless-browser dependency the original design avoided
  (`d2`'s native PNG path). The pipeline stays `d2 → svg → resvg → png`.
- Premium TALA layout engine (not bundled).
- Per-viewer adaptive output (a PNG is a fixed render; mode is chosen at render
  time, not at display time).

## Phase 0 — Fix the render pipeline (live bug, highest priority)

Without text, nothing downstream matters. Scope is `diagrams.sh` + a test.

### Text rendering

`resvg` skips `@font-face` and then drops text whose `font-family` it cannot
resolve (`d2-<n>-font-bold`, etc.). Fix: rewrite the synthetic family names in
d2's SVG to an installed family before resvg runs.

```sh
sed -E 's/d2-[0-9]+-font-[a-z]+/DejaVu Sans/g' "$svg" > "$svg.fixed"
```

Verified live: PNG output grows ~70% (the glyphs). `DejaVu Sans` is present via
fontconfig (1400+ faces available); the family is configurable via
`AGENT_CAROUSEL_D2_FONT` for hosts that prefer another sans.

### Bold / italic fidelity

d2 distinguishes weight by font *file*, not `font-weight`. After the remap,
bold/italic labels fall back to regular weight. Preserve emphasis by injecting
`font-weight:bold` / `font-style:italic` into the corresponding CSS rules (or
mapping the bold/italic family names to a bold/italic installed face). Decision
deferred to the plan; both are small.

### Theme by light/dark mode

A PNG is a fixed render, so the theme is chosen when the hook renders:

| Mode | Default theme-id |
|---|---|
| dark | `200` Dark Mauve |
| light | `105` Buttered Toast |

Overridable via `AGENT_CAROUSEL_D2_THEME` / `AGENT_CAROUSEL_D2_DARK_THEME`.
Mode is read from a configurable signal (env first; no fragile terminal
auto-detection — define the error out of existence). `--sketch` is on by
default, overridable.

### Testing

`tests/diagrams.bats` gains a regression test that renders a labelled diagram
and asserts the PNG actually contains text (output size above a textless
baseline, or a pixel/coverage heuristic) — so a silent regression to boxes
can never ship again.

## Phase 1 — Rich, beautiful authoring skill

Rewrite `skills/diagrams/SKILL.md` as a single rich file (~250–350 lines /
~2.5–3k tokens — normal for a domain-reference skill; loaded only when the
agent is actually drawing). Structure:

1. Frontmatter — sharpen the trigger description.
2. When to draw / when not — keep the existing judgment, compressed.
3. Where to write — keep (scratch dir, never in-repo), compressed.
4. **House style (beautiful-by-default)** — copy-paste header:
   `vars: { d2-config: { sketch: true; pad: 16 } }` (theme is set by the
   hook, not the file), plus a `classes:` block with semantic roles
   (`svc` cool-blue, `store` green cylinder, `ext` amber/dashed). Evidence-
   backed: the palette appears in the official style docs and a peer d2 skill.
5. Core syntax — shapes, `->`/`<->`/`--`, containers, labels.
6. **Flow direction** — default `direction: right` (horizontal) to fill the
   carousel's landscape preview; use vertical (`down`) only for inherently-tall
   structures (sequence diagrams, deep trees); prefer grouping over long thin
   chains. Ties directly to the viewer's aspect-fit.
7. **Rich constructs** — groupings/nested containers, `sql_table` ERDs
   (`constraint: primary_key|foreign_key|unique`, `table.col -> table.col`,
   `layout-engine: elk` for row-precise edges), `shape: sequence_diagram`,
   `classes`/`class`, `icon:`, `near` legends, styling vocab (fill, stroke,
   stroke-dash, border-radius, shadow, font-color).
8. **3–4 worked examples** — architecture+grouping+icons, ERD, sequence/state,
   pipeline. **Every example render-tested** in bats so the doc can never ship
   syntax that doesn't compile (we hit the `,` vs `;` map-separator bug live).
9. Aesthetic do/don't — ≤~12 nodes, 2–3 intentional colors, one shape per role,
   label every edge, let layout breathe.
10. Requirements — `d2` + `resvg` on PATH.

## Phase 2 — Carousel display upgrades

Builds on `docs/plans/2026-06-10-zoom-pan.md`. Diagrams get first-class care via
the `source:"d2"` tag the manifest already carries.

- **Zoom + panning** — per the existing zoom-pan plan. For `source:"d2"`
  entries, zoom re-renders crisp from the vector source/SVG rather than
  bitmap-scaling (diagrams are the densest images and benefit most).
- **Maximize / fill preview** — a key to hide chrome and use the whole pane.
- **Upscale small diagrams** — let a small diagram fill the preview box
  (currently `scale >= 1` returns the source unchanged → tiny diagrams).
- **Viewer-side dedup** — `loadManifest` shows every decodable entry; dedup by
  path so a twice-referenced image appears once.
- **Nav wraparound** — optional; `l` at last → first.

## Known trade-off: role palette vs dark mode

The semantic `classes` use explicit fills (light blue/green/amber). Explicit
fills override theme colors, so the same classes on a dark theme would put
light boxes on a dark background. Phase 1 must resolve this — options: (a) ship
mode-paired palettes (light + dark fill sets), (b) pick role colors that read
on both backgrounds, or (c) let dark mode drop explicit fills and rely on the
theme. Picked in the Phase 1 plan; flagged here so it isn't discovered late.

## Error handling (define errors out of existence)

- Renderers absent → hook no-ops silently (unchanged).
- Font family unavailable → configurable `AGENT_CAROUSEL_D2_FONT`; default
  `DejaVu Sans` is near-universal.
- Unknown/garbled mode signal → fall back to light defaults; never crash.

## Files

| File | Change |
|---|---|
| `adapters/claude-code/plugin/scripts/diagrams.sh` | Phase 0: font remap, bold/italic, theme-by-mode + env overrides, sketch default |
| `tests/diagrams.bats` | Phase 0: text-present regression test; Phase 1: render-test each skill example |
| `adapters/claude-code/plugin/skills/diagrams/SKILL.md` | Phase 1: full rewrite (rich + beautiful + flow direction) |
| `gallery.go`, `gallery_render.go`, `gallery_cache.go` | Phase 2: zoom/pan, maximize, upscale, dedup (aligned with zoom-pan plan) |

## Phasing & ordering

Phase 0 first (unblocks everything; it's a live bug). Phase 1 next (the
authoring win). Phase 2 last (display polish, coordinates with the in-flight
zoom-pan work). Each phase → its own implementation plan via writing-plans.
