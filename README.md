# V.E.G.A.S. Protocol

**Virtual Electronic General Assistant System**

A distributed, touch-enabled TUI field assistant inspired by Fallout: New Vegas. Built with Go and the Charmbracelet stack. Designed to run on a Raspberry Pi as a dedicated "Pip-Boy" terminal, connecting to a server running Ollama for local AI inference.

![vegas.gif](./assets/vegas.gif)

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

**Spotify Radio** -- Real-time Spotify playback with ASCII-rendered album art, animated equalizer, and full controls (play/pause, next/prev, shuffle, repeat, volume). Displays the auth URL in the TUI for headless Pi deployment. Falls back gracefully when Spotify is not connected.

**Git Integration** -- View git status, branch info, stage files, commit, push, pull, and fetch directly from the PROJ tab.

**ASCII Map** -- A scrollable Mojave Wasteland map with drag-to-scroll and mouse wheel support.

**Settings** -- Configure your editor, server URL, Ollama model, theme, and project directories. All settings persist to disk.

**Mouse and Touch** -- Every tab has clickable buttons. Lists support click selection and mouse wheel scrolling. The map supports click-drag panning.

---

## Requirements

- **Go** 1.21 or later
- **Ollama** (for AI chat features)
- A terminal emulator with mouse support (most modern terminals)

Optional:

- **PostgreSQL** (Supabase) for database features (To be implemented)
- A **Raspberry Pi** with touchscreen for the full Pip-Boy experience

---

## Installation

### From Source

```bash
git clone https://github.com/choice404/vegas-protocol.git
cd vegas-protocol/vegas-tui

# Build the TUI client
go build -o vegas-protocol ./cmd/vegas # can also name `vegas`` to match go install

# Build the server
go build -o vegas-server ./cmd/vegas-server
```

### Using go install

```bash
# Install the TUI client
go install github.com/choice404/vegas-protocol/vegas-tui/cmd/vegas@latest

# Install the server
go install github.com/choice404/vegas-protocol/vegas-tui/cmd/vegas-server@latest
```

### Cross-Compile for Raspberry Pi

```bash
cd vegas-tui
GOOS=linux GOARCH=arm64 go build -o vegas-protocol ./cmd/vegas
GOOS=linux GOARCH=arm64 go build -o vegas-server ./cmd/vegas-server
```

You can also use `go isntall`.

---

## Quick Start

The TUI works standalone for most features (system stats, inventory, quests, projects, radio, map, settings). AI chat requires the server and Ollama.

```bash
# 1. Pull an Ollama model
ollama pull llama3.1:8b

# 2. Start the server (in one terminal)
./vegas-server

# 3. Start the TUI (in another terminal)
./vegas
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
| 6   | RADIO  | `6` | Spotify radio. `Space` play/pause, `n`/`p` next/prev, `s` shuffle, `r` repeat, `+`/`-` volume.                |
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

### Spotify

The RADIO tab connects to Spotify for real-time playback. To set it up:

1. Create a Spotify app at https://developer.spotify.com/dashboard
2. Set the redirect URI to `http://127.0.0.1:8888/callback`
3. Export your credentials:

```bash
export SPOTIFY_ID="your-client-id"
export SPOTIFY_SECRET="your-client-secret"
```

4. Launch the TUI, go to the RADIO tab, and press `Enter` to authenticate
5. A browser window will open (or the auth URL is displayed in the TUI for headless setups)
6. After authorizing, the token is saved to `~/.config/vegas-protocol/spotify_token.json` and reused across sessions

**Pi deployment:** Copy `spotify_token.json` from a machine with a browser to the Pi's `~/.config/vegas-protocol/` directory, and set the same `SPOTIFY_ID`/`SPOTIFY_SECRET` env vars. The TUI will connect immediately without needing a browser.

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
│   ├── radio.go              # Spotify radio player tab
│   ├── spotify.go            # Spotify auth, API commands, state polling
│   ├── albumart.go           # Album art fetch, decode, ASCII render
│   ├── git.go                # Git integration for project browser
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

### TV Tab (Anime Streaming via ani-cli)

- Add a new TV tab that integrates with `ani-cli` for searching and playing anime
- Browse trending, recently updated, and search results
- Select episodes and launch playback through `ani-cli` using `tea.ExecProcess`
- Track watch history and maintain a watchlist as part of the quest/config system
- Display episode metadata and synopsis within the TUI
- Would output to external video player (default is mpv on linux)
