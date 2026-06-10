# agent-carousel

A tmux image carousel for coding agents — a big preview plus a filmstrip of
thumbnails, rendered in a tmux split. Shows every image a coding-agent session
touches (reads, writes, screenshots) so you can browse them without leaving the
terminal.

Full-fidelity preview on kitty-graphics terminals (kitty/ghostty); `chafa`
block-art fallback elsewhere.

## Architecture

The viewer is **agent-agnostic**. It renders a per-pane manifest and has no
knowledge of which agent produced it — the [manifest JSONL](docs/MANIFEST.md) is
the stable interface. Each agent gets a small **capture adapter** that appends
to the manifest; the viewer never changes.

- **Viewer** (Go binary) — reads `${CLAUDE_STATUS_DIR:-/tmp/claude-status}/images/<pane>.jsonl`,
  renders the carousel via the kitty graphics protocol (or chafa fallback).
- **Adapters** (`adapters/`) — per-agent capture. Today: `claude-code/`
  (a Claude Code PostToolUse hook + plugin + skill).

Extracted from [lazytmux](https://github.com/noamsto/lazytmux), which consumes
this repo as a flake input.

## Status

Pre-extraction. See [docs/2026-06-10-design.md](docs/2026-06-10-design.md) for
the design: standalone-repo extraction + kitty zoom/pan, with D2 diagrams and
non-kitty zoom as future work.
