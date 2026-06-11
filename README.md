# agent-carousel

A terminal image carousel for coding agents — a big preview plus a filmstrip of
thumbnails, rendered in a tmux split or kitty window (dual-mode). Shows every
image a coding-agent session touches (reads, writes, screenshots) so you can
browse them without leaving the terminal.

Full-fidelity preview on kitty-graphics terminals (kitty/ghostty); `chafa`
block-art fallback elsewhere.

## Architecture

The viewer is **agent-agnostic**. It renders a per-pane manifest and has no
knowledge of which agent produced it — the [manifest JSONL](docs/MANIFEST.md) is
the stable interface. Each agent gets a small **capture adapter** that appends
to the manifest; the viewer never changes.

- **Viewer** (Go binary) — reads `${AGENT_CAROUSEL_DIR:-${CLAUDE_STATUS_DIR:-/tmp/claude-status}}/images/<pane>.jsonl`,
  renders the carousel via the kitty graphics protocol (or chafa fallback).
- **Adapters** (`adapters/`) — per-agent capture. Today: `claude-code/`
  (a Claude Code PostToolUse hook + plugin + skill).

Extracted from [lazytmux](https://github.com/noamsto/lazytmux), which consumes
this repo as a flake input.

## Status

Live standalone repo — the viewer binary, Claude Code capture adapter, and
plugin skill all build and are consumed by
[lazytmux](https://github.com/noamsto/lazytmux) as a flake input.

See [docs/2026-06-10-design.md](docs/2026-06-10-design.md) for the design
notes. Zoom/pan (Plan 2) is not yet implemented.
