# Review GUI

The Review GUI is a [Tauri v2](https://tauri.app) desktop application that provides a graphical interface for the review daemon.

## Features

- **Auto-discovery** — finds review daemons running on your local machine automatically
- **Remote SSH discovery** — probes remote machines via SSH tunneling (configure hosts in Settings)
- **Session browser** — browse all review sessions on any discovered daemon
- **Diff viewer** — view changed files with additions/deletions
- **Comment workflow** — add, view, and resolve review comments
- **Approve / Request changes** — approve and push, or request changes with a message

## Architecture

```
GUI (React/TypeScript + Tailwind)
  ├── invoke("discover_daemons") → Rust backend (reqwest + SSH)
  ├── fetch(daemon API)        → Go daemon (HTTP JSON API)
  └── @tauri-apps/plugin-store → settings persistence
```

The Rust backend handles:
- Probing `localhost:7080/health` for local daemons
- Setting up SSH tunnels (`ssh -L`) to forward remote daemon ports
- Probing through tunnels for remote daemons

All session management (commits, diffs, comments, approve) goes through the daemon's HTTP JSON API directly via `fetch()` from the frontend.

## Development

### Prerequisites

- [Bun](https://bun.sh) — JavaScript runtime & package manager
- [Rust](https://rustup.rs) — Tauri backend
- Tauri system dependencies:
  - **macOS:** Xcode Command Line Tools
  - **Linux:** `libgtk-3-dev libwebkit2gtk-4.1-dev libappindicator3-dev librsvg2-dev libssl-dev`

### Setup

```bash
cd gui
bun install
```

### Dev Mode

```bash
cd gui
bun run dev    # starts dev server + Tauri window with hot reload
```

### Build

```bash
cd gui
bun run build  # production build → gui/src-tauri/target/release/bundle/
```

### Type Checking

```bash
cd gui
npx tsc --noEmit
```

## Settings

SSH hosts are configured in the **Settings** tab and persisted via Tauri's store plugin. Each host has:
- **Label** — friendly name for the host
- **Hostname** — SSH hostname or IP
- **SSH Port** — SSH port (default: 22)
- **Daemon Port** — review daemon port on the remote (default: 7080)

## CI

GUI builds run on every PR and push to `main`:
- **macOS aarch64** — full Tauri build, uploads DMG as artifact
- **Linux** — `cargo check` to verify Rust compilation
