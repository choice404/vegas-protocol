# V.E.G.A.S. Protocol

**Virtual Electronic General Assistant System**

A distributed, touch-enabled TUI field assistant inspired by Fallout: New Vegas. Built with Go and the Charmbracelet stack. Designed to run on a Raspberry Pi as a dedicated "Pip-Boy" terminal, connecting to a server running Ollama for local AI inference.

---

## Table of Contents

- [Overview](#overview)
- [Screenshots](#screenshots)
- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
  - [From Source](#from-source)
  - [Using go install](#using-go-install)
  - [Cross-Compile for Raspberry Pi](#cross-compile-for-raspberry-pi)
- [Quick Start](#quick-start)
- [Server Setup](#server-setup)
  - [Chat-Only Mode](#chat-only-mode)
  - [Full Mode with Supabase](#full-mode-with-supabase)
- [Usage](#usage)
  - [Navigation](#navigation)
  - [Tabs](#tabs)
- [Configuration](#configuration)
- [Project Structure](#project-structure)
- [TODO](#todo)
- [License](#license)

---

## Overview

V.E.G.A.S. Protocol is a terminal-based personal data assistant with a Fallout: New Vegas aesthetic. It provides:

- Real-time system monitoring (CPU, RAM, temperature, disk)
- An AI chat terminal powered by local Ollama models
- A project management system (questlines and tasks) with LLM integration
- A project file browser with editor integration
- A visual radio player with animated equalizer
- A scrollable ASCII map with drag-to-scroll
- Full mouse/touch support alongside keyboard controls

The system is designed as a client-server architecture. The TUI client runs on any terminal (or a Raspberry Pi with a touchscreen), while the server handles AI inference via Ollama and optional database operations via Supabase.

---

## Screenshots

_Coming soon._

---

## Features

**System Monitoring** -- Real-time gauges for CPU, RAM, temperature, and disk usage with color-coded thresholds.

**AI Chat Terminal** -- Chat with V.E.G.A.S., a local AI assistant running on Ollama (Llama 3.1 8B). The AI can create project plans that automatically populate your quest log.

**Quest System** -- Two-level project management with questlines (projects) containing individual tasks. Supports priority levels, completion tracking, and persistent storage.

**Project Browser** -- Browse project directories, search with filtering, navigate file trees, and open files directly in your preferred editor.

**Visual Radio** -- Five themed radio stations with an animated equalizer, frequency dial, and playback controls.

**ASCII Map** -- A scrollable Mojave Wasteland map with drag-to-scroll and mouse wheel support.

**Settings** -- Configure your editor, server URL, Ollama model, theme, and project directories. All settings persist to disk.

**Mouse and Touch** -- Every tab has clickable buttons. Lists support click selection and mouse wheel scrolling. The map supports click-drag panning.

---

## Requirements

- **Go** 1.21 or later
- **Ollama** (for AI chat features)
- A terminal emulator with mouse support (most modern terminals)

Optional:

- **PostgreSQL** (Supabase) for database features
- A **Raspberry Pi** with touchscreen for the full Pip-Boy experience

---

## Installation

### From Source

```bash
git clone https://github.com/choice404/vegas-protocol.git
cd vegas-protocol/vegas-tui

# Build the TUI client
go build -o vegas-protocol ./cmd/tui

# Build the server
go build -o vegas-server ./cmd/server
```

### Using go install

```bash
# Install the TUI client
go install github.com/choice404/vegas-protocol/vegas-tui/cmd/tui@latest

# Install the server
go install github.com/choice404/vegas-protocol/vegas-tui/cmd/server@latest
```

### Cross-Compile for Raspberry Pi

```bash
cd vegas-tui
GOOS=linux GOARCH=arm64 go build -o vegas-protocol ./cmd/tui
GOOS=linux GOARCH=arm64 go build -o vegas-server ./cmd/server
```

Transfer the binaries to your Pi via `scp` or USB.

---

## Quick Start

The TUI works standalone for most features (system stats, inventory, quests, projects, radio, map, settings). AI chat requires the server and Ollama.

```bash
# 1. Pull an Ollama model
ollama pull llama3.1:8b

# 2. Start the server (in one terminal)
./vegas-server

# 3. Start the TUI (in another terminal)
./vegas-protocol
```

Press any key after the boot sequence to enter the main interface.

---

## Server Setup

### Chat-Only Mode

No configuration needed. The server runs with Ollama relay only:

```bash
./vegas-server
```

This starts on port 8080 and connects to Ollama at `localhost:11434`.

### Full Mode with Supabase

```bash
cp .env.example .env
```

Edit `.env` with your Supabase credentials:

```
DATABASE_URL=postgresql://postgres.USER:PASSWORD@HOST:PORT/postgres
SUPABASE_URL=https://YOUR_REF.supabase.co
SUPABASE_ANON_KEY=your-anon-key
SUPABASE_JWT_SECRET=your-jwt-secret
SERVER_PORT=8080
OLLAMA_URL=http://localhost:11434
```

**Important:** If your database password contains special characters, they must be URL-encoded (`%` becomes `%25`, `!` becomes `%21`, etc.).

**Important:** Use the Supabase **connection pooler** URL (from Dashboard > Project Settings > Database) rather than the direct connection URL. The pooler has IPv4 support; direct connections may be IPv6-only.

Push the database schema:

```bash
./scripts/schema-push.sh
```

Then start the server:

```bash
./vegas-server
```

---

## Usage

### Navigation

| Input        | Action                          |
| ------------ | ------------------------------- |
| `1` - `8`    | Jump to tab                     |
| `Tab`        | Next tab                        |
| `Shift+Tab`  | Previous tab                    |
| `q`          | Quit                            |
| `Ctrl+C`     | Force quit                      |
| `9`          | Exit the program                |
| Mouse click  | Click tabs, buttons, list items |
| Mouse wheel  | Scroll lists and viewports      |
| Click + drag | Pan the map view                |

### Tabs

| #   | Tab    | Key | Description                                                                                                   |
| --- | ------ | --- | ------------------------------------------------------------------------------------------------------------- |
| 1   | STATS  | `1` | Real-time system gauges. Auto-refreshes every 2 seconds.                                                      |
| 2   | ITEMS  | `2` | Fallout-themed inventory. `j`/`k` to navigate, `Enter` to inspect.                                            |
| 3   | DATA   | `3` | AI chat terminal. Type a query and press `Enter` to send.                                                     |
| 4   | QUESTS | `4` | Project management. `A` for new quest, `a` for new task, `d` to delete, `Enter`/`Space` to toggle completion. |
| 5   | PROJ   | `5` | Project browser. `/` to search, `e` to edit file, `P` to add directory, `Esc` to go back.                     |
| 6   | RADIO  | `6` | Radio player. `Space` to play/pause, `n`/`p` for next/prev.                                                   |
| 7   | MAP    | `7` | ASCII map. Arrow keys or `hjkl` to scroll, click-drag to pan.                                                 |
| 8   | SET    | `8` | Settings editor. `Enter` to edit, `s` to save to disk.                                                        |
| 9   | EXIT   | `9` | Exit the program                                                                                              |

---

## Configuration

Settings are stored at `~/.config/vegas-protocol/settings.json` and can be edited from the SET tab.

| Setting      | Default                  | Description                     |
| ------------ | ------------------------ | ------------------------------- |
| Editor       | `$EDITOR` or `nano`      | Text editor for file opening    |
| Server URL   | `http://localhost:8080`  | V.E.G.A.S. server address       |
| Ollama URL   | `http://localhost:11434` | Ollama API address              |
| Ollama Model | `llama3.1:8b`            | Model for AI chat               |
| Theme        | `green`                  | Color theme (`green`, `amber`)  |
| Project Dirs | (none)                   | Directories containing projects |

Quest data is stored at `~/.config/vegas-protocol/quests.json`.

---

## Project Structure

```
vegas-tui/
├── cmd/
│   ├── server/main.go        # HTTP server entry point
│   └── tui/main.go           # TUI entry point
├── internal/
│   ├── app.go                # Root model, tab routing, input handling
│   ├── boot.go               # Boot sequence animation
│   ├── stats.go              # System monitoring tab
│   ├── items.go              # Inventory tab
│   ├── data.go               # AI chat tab
│   ├── quests.go             # Quest management tab
│   ├── projects.go           # Project browser tab
│   ├── radio.go              # Radio player tab
│   ├── mapview.go            # Map viewer tab
│   ├── settingsview.go       # Settings editor tab
│   ├── theme/                # Colors, styles, ASCII art
│   ├── settings/             # Persistent config and quest storage
│   ├── config/               # Server environment config
│   ├── db/                   # Database connection (PostgreSQL)
│   ├── client/               # HTTP client
│   └── server/               # HTTP server, handlers, middleware
├── scripts/
│   └── schema-push.sh        # Database schema deployment
├── .env.example              # Environment variable template
├── schema.sql                # PostgreSQL schema
└── go.mod
```

---

## TODO

The following features are planned for future development:

### Git Integration for Projects

- Display git status (branch, dirty state, ahead/behind) for each project in the PROJ tab
- Show recent commits and changed files
- Support basic git operations (stage, commit, pull, push) from within the TUI
- Display diff views for modified files

### Automatic Update Checking

- Use git tags and GitHub releases to check for new versions of V.E.G.A.S.
- Notify the user in the STATS or boot screen when an update is available
- Support self-update by pulling the latest release binary or rebuilding from source

### Spotify Integration for Radio

- Connect to the Spotify API for real-time playback in the RADIO tab
- Display currently playing track, artist, and album art (ASCII-rendered)
- Support playback controls (play, pause, skip, previous, shuffle, repeat)
- Browse playlists and search for tracks within the TUI
- Fall back to the existing visual-only radio when Spotify is not connected

### TV Tab (Anime Streaming via ani-cli)

- Add a new TV tab that integrates with `ani-cli` for searching and playing anime
- Browse trending, recently updated, and search results
- Select episodes and launch playback through `ani-cli` using `tea.ExecProcess`
- Track watch history and maintain a watchlist as part of the quest/config system
- Display episode metadata and synopsis within the TUI

---

## License

MIT
