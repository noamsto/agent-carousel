---
name: diagrams
description: Use when drawing a diagram for the user with D2 — architecture, data flow, state machines, pipelines, entity relationships. The diagram renders into the agent-carousel viewer. Covers the d2 syntax cheatsheet and where to write the file.
---

# Diagrams (D2 → carousel)

When a picture beats prose, write a [D2](https://d2lang.com) diagram as a `.d2`
file. A `PostToolUse` hook renders it browser-free (`d2 → svg → resvg → png`)
into the per-pane image manifest, and the carousel shows it like any other
image — auto-opening once per session.

## Where to write it

Write the `.d2` file to the scratch dir named in the SessionStart guidance —
an absolute path under the carousel state dir
(`<state-dir>/images/diagrams/src/<name>.d2`), **outside any repo**. Never write
`.d2` files into the working project; they are throwaway diagram sources, not
project artifacts.

## When to draw (and when not to)

- **Do:** architecture, data flow, state machines, pipelines, entity
  relationships — anything with structure that's clearer seen than read.
- **Don't:** trivial or linear one-step things; restating prose. One diagram per
  concept. Prose stays primary — the diagram supplements the explanation.

## D2 cheatsheet

```d2
# shapes + connections
api: API Server
db: Postgres
api -> db: query

# nested containers
service: {
  worker
  queue
  worker -> queue
}

# labeled, directional, bidirectional
a -> b: starts
a <-> b: syncs
a -- b: peers

# shape styling
cache: {shape: cylinder}
user: {shape: person}
```

- Connections: `->` directed, `<->` bidirectional, `--` undirected.
- Containers nest with `name: { ... }`; reference children as `parent.child`.
- Keep diagrams small and focused — a handful of nodes, one idea.

## Requirements

Rendering needs `d2` and `resvg` on PATH. If either is missing the hook no-ops
silently (no diagram appears, no error) — install both to enable diagrams.
