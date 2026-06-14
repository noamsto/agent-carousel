<div align="center">

# 🎠 agent-carousel

**A terminal image carousel for coding agents** — browse every screenshot,
render, and image your agent touches, without leaving the terminal.

[![CI](https://github.com/noamsto/agent-carousel/actions/workflows/ci.yml/badge.svg)](https://github.com/noamsto/agent-carousel/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

<!-- TODO(demo): hero screenshot + demo.gif — deferred until the rich-diagrams
     work lands so the demo captures diagrams at their best. Drop assets in
     docs/assets/ and replace this comment. -->

A big **preview** of the selected image plus a **filmstrip** of thumbnails,
rendered in a tmux split or kitty window (dual-mode). One half **captures** every
image a coding-agent session touches (reads, writes, screenshots); the other
**renders** them and auto-refreshes as new ones arrive — so you can glance at
what your agent is doing without leaving the terminal.

## Features

- 🖼️ **Preview + filmstrip** — a large view of the selected image above a
  scrollable strip of thumbnails.
- 🪄 **Auto-capture** — a PostToolUse hook records every image the session reads,
  writes, or screenshots into a per-pane manifest. Nothing to do by hand.
- 🔭 **Dual-mode rendering** — a tmux split or a kitty window, auto-detected from
  the host. Opens beside your agent, not wherever you happened to navigate.
- ⚡ **Live** — opens on the newest image and follows new captures as they stream
  in, until you take over with the keyboard; polls for changes every ~1.5s.
- ✨ **Crisp** — kitty graphics protocol on kitty/ghostty, with a
  [`chafa`](https://hpjansson.org/chafa/) block-art fallback everywhere else.
- 🧹 **Robust** — skips deleted or corrupt entries instead of rendering blank
  cells; logs why, for when you wonder where an image went.
- 📊 **Diagrams (optional)** — the agent can draw [D2](https://d2lang.com)
  diagrams that render straight into the carousel.

## Install

Two PATH entrypoints — the `agent-carousel` viewer and the `tmux-claude-images`
toggle that opens it. Install **both** (the toggle launches the viewer into a
fresh pane, which resolves `agent-carousel` from that pane's PATH).

**Prebuilt binaries** — no toolchain, Linux/macOS · amd64/arm64. Downloads both
entrypoints:

```bash
os=$(uname -s | tr '[:upper:]' '[:lower:]'); arch=$(uname -m)
case $arch in x86_64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; esac
mkdir -p ~/.local/bin   # ensure this is on your PATH
curl -fsSL "https://github.com/noamsto/agent-carousel/releases/latest/download/agent-carousel_${os}_${arch}.tar.gz" \
  | tar -xz -C ~/.local/bin agent-carousel tmux-claude-images
```

(Or download an archive from the [releases page](https://github.com/noamsto/agent-carousel/releases) and extract both onto your PATH.)

**Nix:**

```bash
nix profile install github:noamsto/agent-carousel          # viewer
nix profile install github:noamsto/agent-carousel#toggle   # toggle
```

**Go** — viewer only; pair it with the toggle from the release archive or
`scripts/tmux-claude-images.sh`:

```bash
go install github.com/noamsto/agent-carousel@latest
```

Then install the **capture** half — the Claude Code plugin (run inside Claude
Code, not the shell):

```
/plugin marketplace add noamsto/agent-carousel
/plugin install agent-carousel@agent-carousel
```

> 📖 **Step-by-step, agent-friendly guide:** [`docs/INSTALL.md`](docs/INSTALL.md)
> — host check, both entrypoints, plugin, optional deps, and a smoke test, each
> with a verification command.

<details>
<summary>Via lazytmux (Nix / Home Manager)</summary>

[lazytmux](https://github.com/noamsto/lazytmux) consumes this repo as a flake
input — it puts the viewer and `tmux-claude-images` on PATH and binds
`prefix + I`. If you run lazytmux you already have the carousel; just add the
plugin above for capture.

</details>

## Usage

Ask your agent to *show the images from this conversation* (or invoke the
`image-gallery` skill), or open it yourself:

```bash
tmux-claude-images   # toggle: run again to close. In tmux, also prefix + I (lazytmux)
```

It does nothing until images have been captured — the manifest fills as the
session reads/writes/screenshots images.

### Keybindings

| Key | Action |
|---|---|
| `←` `→` / `h` `l` / `↑` `↓` / `k` `j` | Move selection |
| `n` / `p` | Page the filmstrip |
| `g` / `G` (or `Home` / `End`) | First / last image |
| `1`–`9` | Jump to the Nth image |
| `Enter` / `o` | Open in the default app |
| `O` | Open the containing folder |
| `r` | Reload the manifest |
| `q` / `Ctrl-C` | Quit |

## Architecture

The viewer is **agent-agnostic**. It renders a per-pane manifest and has no
knowledge of which agent produced it — the [manifest JSONL](docs/MANIFEST.md) is
the stable interface. Each agent gets a small **capture adapter** that appends to
the manifest; the viewer never changes.

- **Viewer** (Go binary) — reads
  `${AGENT_CAROUSEL_DIR:-${CLAUDE_STATUS_DIR:-/tmp/claude-status}}/images/<pane>.jsonl`
  and renders via the kitty graphics protocol (or chafa fallback).
- **Adapters** (`adapters/`) — per-agent capture. Today: `claude-code/`
  (a PostToolUse hook + plugin + skill).

Extracted from [lazytmux](https://github.com/noamsto/lazytmux), which consumes
this repo as a flake input.

## Status

Live standalone repo — the viewer binary, Claude Code capture adapter, and plugin
skill all build and are consumed by
[lazytmux](https://github.com/noamsto/lazytmux) as a flake input. Zoom/pan is not
yet implemented; see [docs/2026-06-10-design.md](docs/2026-06-10-design.md) for
the design notes.
