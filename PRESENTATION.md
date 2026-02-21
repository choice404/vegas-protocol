# V.E.G.A.S. Protocol

**Virtual Electronic General Assistant System**

> A distributed, touch-enabled terminal field assistant inspired by Fallout: New Vegas — built for Rebel Hacks 2025 at UNLV.

- **Developer:** Austin Choi
- **Event:** Rebel Hacks 2025, UNLV — February 20-21, 2026
- **Theme:** Las Vegas
- **Version:** 0.3.0

---

## The Problem

Whether you're at DEF CON, a hackathon, or managing headless servers in the field, you don't want to carry a full laptop with keyboard and mouse into every situation. Cybersecurity specialists, pentesters, and field engineers need system monitoring, secure communication, AI assistance, and project management, but the tools for these are scattered across dozens of GUI applications that require a desktop environment. When you're working from a Raspberry Pi, a single-board computer in a server rack, or a minimal terminal at a conference where you'd rather not have your primary laptop exposed, you need a single tool that does everything from one screen and works offline.

## The Solution

V.E.G.A.S. Protocol is a **distributed personal data assistant** that runs entirely in the terminal. A single binary gives you system monitoring, AI chat, project management, a file browser with git integration, music playback control, peer-to-peer networking with encrypted chat, and multiplayer Texas Hold'em poker — all wrapped in a Fallout: New Vegas Pip-Boy aesthetic.

It's built for people who live in the terminal and need a compact, self-contained toolkit they can deploy anywhere:

- **Cybersecurity specialists** — Run it on a Raspberry Pi with a touchscreen at DEF CON. Run nmap scans, dig DNS records, curl endpoints, and hash data from the built-in toolkit. Monitor system health, coordinate with your team over encrypted P2P chat, manage tasks, and query a local AI — all without exposing a laptop or needing internet access.
- **Terminal-first developers** — One binary replaces a window manager's worth of tools. System stats, git operations, project browsing, Spotify control, AI chat, and task management without leaving the terminal.
- **Headless system administrators** — SSH into any box, run the binary, and get a full dashboard with system monitoring, file browsing, and project management. No X11 forwarding, no VNC, no web UI.
- **Hackathon competitors** — Offline-first by design. The AI runs locally via Ollama, P2P networking needs no infrastructure, and the whole thing cross-compiles to ARM64 with a single command.

The architecture is split into two nodes:

- **Client** — A Go TUI binary that runs on anything with a terminal, including a Raspberry Pi 5 with a touchscreen (raw `/dev/input` hardware reader, no gpm or desktop environment needed)
- **Server** — An HTTP relay that connects the client to a local Ollama LLM instance for AI chat

The client works standalone for most features. The server is only needed for AI chat. P2P networking works device-to-device with zero infrastructure.

---

## Architecture

```
┌─────────────────────────────┐          ┌─────────────────────────────┐
│  CLIENT (Raspberry Pi 5)    │          │  SERVER (Laptop/Desktop)    │
│  Touch Screen TUI           │          │  Compute Node               │
│                             │          │                             │
│  ┌─ Bubble Tea Loop ──────┐ │          │  ┌─ Chi HTTP Server ──────┐ │
│  │  Input: Touch/Keyboard │ │          │  │  POST /api/chat        │ │
│  │  STATS: gopsutil       │ │   HTTP   │  │    → Ollama Relay      │ │
│  │  ITEMS: CLI Toolkit    │ ├──────────┤  │  POST /auth/*          │ │
│  │  DATA: AI Terminal     │ │          │  │    → Supabase Proxy    │ │
│  │  QUESTS: Task Manager  │ │          │  │  GET  /health          │ │
│  │  PROJ: File Browser    │ │          │  └────────────────────────┘ │
│  │  RADIO: Spotify Player │ │          │             │               │
│  │  MAP: ASCII Wasteland  │ │          │  ┌──────────▼─────────────┐ │
│  │  LINK: P2P + Poker     │ │          │  │  Ollama                │ │
│  │  SET: Configuration    │ │          │  │  Llama 3.1 8B (GPU)    │ │
│  └────────────────────────┘ │          │  └────────────────────────┘ │
│                             │          │                             │
│  Config: ~/.config/         │          │  Optional:                  │
│          vegas-protocol/    │          │                             │
│  ┌────────────────────────┐ │          │  ┌────────────────────────┐ │
│  │ settings.json          │ │          │  │ PostgreSQL (Supabase)  │ │
│  │ quests.json            │ │          │  │ JWT Auth               │ │
│  │ spotify_token.json     │ │          │  └────────────────────────┘ │
│  └────────────────────────┘ │          └─────────────────────────────┘
└──────────┬──────────────────┘
           │  TLS/TCP (P2P)
┌──────────▼──────────────────┐
│  OTHER CLIENTS              │
│  P2P Chat + Poker           │
│  Star Topology (Host Relay) │
└─────────────────────────────┘
```

---

## Features

### 10 Tabs, One Interface

Every tab is fully navigable by keyboard **and** mouse/touch. Tabs are accessible via number keys (1-9, 0), Tab/Shift+Tab cycling, or clicking the tab bar.

---

### 1. STATS — System Monitoring

Real-time hardware gauges powered by gopsutil:

- **CPU** usage with percentage and visual bar
- **Memory** usage (used / total GB) with visual bar
- **Temperature** from system thermal sensors (supports k10temp, coretemp, cpu_thermal)
- **Disk** usage with percentage and visual bar
- **Hostname** and **uptime** display

Color-coded thresholds: green (normal), amber (>75% warning), red (>90% critical). Auto-refreshes every 2 seconds. Manual `[REFRESH]` button available.

---

### 2. ITEMS — Cybersecurity Toolkit

The Pip-Boy inventory reimagined as a real, functional cybersecurity and sysadmin toolkit. Eight CLI tools with Fallout codenames, executable directly from the TUI with a form-based parameter interface and scrollable output.

**Tools Inventory:**

| Codename          | Real Tool    | Category | Description                                                        |
| ----------------- | ------------ | -------- | ------------------------------------------------------------------ |
| Signal Tracker    | `dig`        | RECON    | DNS reconnaissance — resolve A, AAAA, MX, NS, TXT, ANY records     |
| Wasteland Courier | `curl`       | NETWORK  | HTTP requests — GET/POST/HEAD with custom headers                  |
| Radar Scanner     | `nmap`       | RECON    | Network scanning — TCP connect, SYN, ping sweep, version detection |
| Intel Lookup      | `whois`      | RECON    | Domain/IP OSINT — registration, ownership, nameservers             |
| Echo Locator      | `ping`       | NETWORK  | ICMP connectivity test with configurable count                     |
| Route Mapper      | `traceroute` | NETWORK  | Network path tracing — hop-by-hop route mapping                    |
| Socket Scanner    | `ss`         | NETWORK  | Local socket inspection — listening, TCP, UDP, or all connections  |
| Cipher Kit        | `openssl`    | CRYPTO   | Hash generation — MD5, SHA-1, SHA-256, SHA-512                     |

**4-State UI Flow:**

1. **List** — Browse tools with descriptions and category tags, `[USE]` to select
2. **Form** — Settings-style parameter entry (j/k navigate, Enter to edit text or cycle options like scan type), `[RUN]` to execute
3. **Running** — Animated braille spinner with elapsed time counter, `[CANCEL]` to abort. Commands execute via `exec.CommandContext` with a 30-second timeout.
4. **Output** — Full command output in a scrollable viewport, `[RUN AGAIN]` to re-execute with same parameters

```
 SIGNAL TRACKER (dig) — RECON
 DNS reconnaissance — resolve domain records
 ─────────────────────────────

   TARGET      example.com
 > RECORD TYPE A ↵

 [ RUN ]  [ BACK ]
```

Each tool's argument builder validates required fields and constructs the correct CLI invocation. Option fields (record type, scan type, HTTP method, hash algorithm, socket filter) cycle through presets with Enter — no typing needed. The openssl tool pipes input through `printf | openssl dgst` to handle stdin-based hashing.

---

### 3. DATA — AI Chat Terminal

A conversational interface with "V.E.G.A.S.", an AI assistant that speaks in-character as a Fallout terminal. Uses Ollama running Llama 3.1 8B locally — no cloud API keys needed.

- Non-blocking async requests via `tea.Cmd`
- Themed error messages: "CONNECTION LOST — RAD INTERFERENCE"
- JSON error parsing from server for actionable feedback
- 500-character input with `[SEND]` and `[CLEAR]` buttons

**AI Quest Creation:** When you ask V.E.G.A.S. to plan a project, it returns structured JSON wrapped in a custom marker. The TUI auto-parses this and adds it directly to your Quest Log — bridging AI conversation with actionable project management.

```
User: "Plan my hackathon project"
V.E.G.A.S.: "Understood, Courier. I've drafted a questline for you."
→ Quest "HACKATHON PROJECT" auto-added to QUESTS tab with tasks
```

---

### 4. QUESTS — Project Management

A two-level task management system themed as RPG questlines:

- **QuestLines** are projects (e.g., "REBEL HACKS 2026")
- **Tasks** are individual items within a questline, each with a priority and completion toggle

Create quests manually with `[NEW QUEST]` or let the AI create them via chat. Tasks are toggled with Enter/Space. Everything persists to `~/.config/vegas-protocol/quests.json`.

Ships with a default "REBEL HACKS 2026" questline containing 8 starter tasks.

---

### 5. PROJ — Project Browser + Git Integration

A file browser that doubles as a lightweight IDE launcher:

- Browse configured project directories
- Navigate into subdirectories, open files in your configured editor via `tea.Exec`
- Case-insensitive search filtering with `/`
- `[ADD DIR]` to register new project directories

**Built-in Git operations** — without leaving the terminal:

- View branch name, ahead/behind status, staged/unstaged/untracked files
- Stage all changes (`git add -A`)
- Commit with message
- Push, pull, and fetch (with `GIT_TERMINAL_PROMPT=0` to prevent auth hangs on headless systems)

---

### 6. RADIO — Spotify Player + ASCII Album Art

Full Spotify playback control with OAuth2 authentication:

- **Controls:** Play/Pause, Next, Previous, Shuffle toggle, Repeat cycle (off → track → context), Volume up/down
- **Display:** Current track, artist, album, progress bar with timestamps
- **16-bar animated equalizer** that pulses in time with playback (updates every 150ms)
- **ASCII album art** — downloads cover art, resizes to 20x10, and renders using Unicode block characters (█▒░ ) with brightness-mapped green CRT shading

Auth flow works on both desktop (auto-opens browser) and headless Pi (displays URL for manual auth). Tokens persist to disk with 0600 permissions and auto-refresh.

---

### 7. MAP — ASCII Mojave Wasteland

A scrollable ASCII art map of the Mojave Wasteland. Navigate with arrow keys, vim keys (hjkl), mouse wheel, or click-drag to pan. `[RESET]` returns to the origin.

---

### 8. LINK — P2P Networking, Chat, and Texas Hold'em

The newest feature — peer-to-peer connectivity over the local network with zero external dependencies.

**P2P Networking:**

- **Transport:** TLS-encrypted TCP (self-signed ECDSA P-256 certs generated at runtime)
- **Wire format:** JSON Lines (newline-delimited JSON) — human-readable, easy to debug
- **Topology:** Star — one host runs a TLS server, joiners connect. Host relays messages between all peers.
- **Authentication:** HMAC-SHA256 challenge-response using a shared passphrase
  1. Host sends random salt + challenge
  2. Joiner derives key via `SHA-256(passphrase + salt)`
  3. Joiner responds with `HMAC-SHA256(challenge, derived_key)`
  4. Host verifies — accepts or rejects with reason
- **BubbleTea integration:** Hub delivers messages via Go channel → `tea.Cmd` polls channel → returns `tea.Msg`
- All crypto uses Go stdlib only (`crypto/tls`, `crypto/ecdsa`, `crypto/hmac`, `crypto/sha256`, `crypto/x509`)

**UI Flow:**

1. **Lobby** — Enter display name + passphrase, choose `[HOST]` or `[JOIN]`
2. **Hosting** — Shows your IP:port, waiting for peers to connect
3. **Joining** — Enter host address + passphrase, `[CONNECT]`
4. **Connected** — See peer list, recent chat preview, `[CHAT]` `[POKER]` `[DISCONNECT]`
5. **Chat** — Real-time messaging with viewport history and timestamps
6. **Poker** — Full Texas Hold'em game table

**Texas Hold'em Poker (CAPS Casino):**

- 2-6 players, host is authoritative dealer
- Virtual "CAPS" as chips (1000 starting stack, 10/20 blinds)
- Full game phases: Pre-Flop → Flop → Turn → River → Showdown
- Actions: Fold, Check, Call, Raise (with amount input), All-In
- **Hand evaluation engine:** Tests all C(7,5) = 21 five-card combinations from 7 cards
  - Recognizes all 10 hand ranks: High Card through Royal Flush
  - Handles the A-2-3-4-5 "wheel" straight
  - Kicker-based tiebreaking with 5-rank comparison arrays
- Pot splitting for ties
- Card deck shuffled with `crypto/rand` (Fisher-Yates)
- **ASCII card rendering:** 5-line box art for each card, face-up and face-down
- Host sends personalized state to each player (opponents' hole cards hidden until showdown)

```
  COMMUNITY CARDS
  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐
  │A    │ │K    │ │10   │ │░░░░░│ │░░░░░│
  │  ♠  │ │  ♥  │ │  ♦  │ │░░░░░│ │░░░░░│
  │    A│ │    K│ │   10│ │░░░░░│ │░░░░░│
  └─────┘ └─────┘ └─────┘ └─────┘ └─────┘

  YOUR HAND
  ┌─────┐ ┌─────┐
  │A    │ │J    │
  │  ♠  │ │  ♠  │
  │    A│ │    J│
  └─────┘ └─────┘
```

---

### 9. SET — Settings Editor

Cursor-navigated settings editor (j/k to move, Enter to edit):

| Setting       | Default                  | Description                       |
| ------------- | ------------------------ | --------------------------------- |
| Editor        | `$EDITOR` / nano         | External editor for file opening  |
| Server URL    | `http://localhost:8080`  | V.E.G.A.S. server address         |
| Ollama URL    | `http://localhost:11434` | Local LLM endpoint                |
| Ollama Model  | `llama3.1:8b`            | Model for AI chat                 |
| Theme         | green                    | Color scheme                      |
| Check Updates | ON                       | Check GitHub for updates at boot  |
| Auto Update   | OFF                      | Install updates without prompting |
| Display Name  | (empty)                  | P2P display name                  |
| P2P Port      | 9999                     | Port for hosting P2P sessions     |
| Project Dirs  | (none)                   | Directories for project browser   |

All settings persist to `~/.config/vegas-protocol/settings.json`. `[SAVE]` writes to disk, `[RESET DEFAULTS]` restores factory settings.

---

### Boot Sequence

The app starts with a cinematic Fallout-style boot:

```
 ██╗   ██╗ ███████╗ ██████╗  █████╗ ███████╗
 ██║   ██║ ██╔════╝██╔════╝ ██╔══██╗██╔════╝
 ██║   ██║ █████╗  ██║  ███╗███████║███████╗
 ╚██╗ ██╔╝ ██╔══╝  ██║   ██║██╔══██║╚════██║
  ╚████╔╝  ███████╗╚██████╔╝██║  ██║███████║
   ╚═══╝   ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝
  VIRTUAL ELECTRONIC GENERAL ASSISTANT SYSTEM

  > BIOS CHECK ........................ OK
  > MEMORY ALLOCATION ................ OK
  > SENSOR ARRAY CALIBRATION ......... OK
  > NETWORK INTERFACE ................ OK
  > AI CORE LINK ..................... STANDBY
  > PERSONALITY MATRIX ............... LOADED
  > HOLOTAPE DRIVER .................. OK
  > RADIATION SENSORS ................ NOMINAL

  [████████████████████████████████████████] 100%

  SYSTEM READY. WELCOME, COURIER.

  [ ENTER SYSTEM ]
```

Eight animated system checks at 250ms intervals with a progress bar. Clickable `[ ENTER SYSTEM ]` button for touch devices, or press any key.

---

### Auto-Update System

- Checks GitHub Releases API at boot for new versions
- Semantic version comparison (handles module-aware tags like `vegas-tui/v0.3.0`)
- User prompt with `[ YES ]` / `[ NO ]` buttons (or auto-install if configured)
- Installs via `go install`, then hot-restarts using `syscall.Exec()` to replace the running process with the updated binary — seamless, no manual restart needed

---

## Tech Stack

| Layer         | Technology              | Purpose                                                      |
| ------------- | ----------------------- | ------------------------------------------------------------ |
| Language      | **Go 1.25**             | Single binary, cross-compiles to ARM64                       |
| TUI Engine    | **Bubble Tea v1.3**     | Elm-architecture terminal UI framework                       |
| Styling       | **Lip Gloss v1.1**      | CSS-like terminal styling and layout                         |
| Mouse/Touch   | **bubblezone v1.0**     | Click/tap regions in terminal UIs                            |
| UI Components | **bubbles v1.0**        | Viewport, text input widgets                                 |
| System Stats  | **gopsutil v3**         | CPU, memory, temperature, disk metrics                       |
| HTTP Server   | **Chi v5**              | Lightweight HTTP router for server component                 |
| Database      | **pgx v5**              | PostgreSQL driver (Supabase, optional)                       |
| Auth          | **golang-jwt v5**       | JWT validation for protected endpoints                       |
| Music         | **zmb3/spotify v2**     | Spotify Web API client                                       |
| OAuth         | **golang.org/x/oauth2** | OAuth2 token management                                      |
| Config        | **godotenv**            | .env file loading                                            |
| P2P Crypto    | **Go stdlib**           | `crypto/tls`, `crypto/ecdsa`, `crypto/hmac`, `crypto/sha256` |
| P2P Network   | **Go stdlib**           | `net`, `bufio`, `encoding/json`                              |
| Card Shuffle  | **Go stdlib**           | `crypto/rand` (cryptographically secure)                     |
| Tool Exec     | **Go stdlib**           | `os/exec`, `context` (30s timeout, cancelable)               |
| Touch Input   | **Go stdlib**           | `encoding/binary`, `os` (raw Linux input_event decoding)     |

**Zero external dependencies for the P2P, poker, cybersecurity toolkit, and touch input systems** — all networking, cryptography, game logic, command execution, and hardware input uses only the Go standard library.

---

## Technical Decisions

### Why Go?

Go compiles to a single static binary with zero runtime dependencies. Cross-compilation to Raspberry Pi is a single command: `GOOS=linux GOARCH=arm64 go build`. No Docker, no Node.js, no Python virtual environments — just `scp` the binary and run it.

### Why a TUI instead of a web app?

The target deployment is a Raspberry Pi 5 with a 7" touchscreen running in a bare TTY — no desktop environment, no browser. A TUI works everywhere: SSH sessions, bare consoles, desktop terminal emulators, and touch-capable framebuffer consoles. It also runs offline by default.

### Why Bubble Tea?

Bubble Tea implements the Elm Architecture (Model-Update-View) in Go. Each tab is an independent model with its own state, update logic, and view function. The parent App model routes messages to the active tab. This makes adding new tabs trivial — the LINK tab was added without modifying any existing tab's code.

### Why a star topology for P2P?

A mesh topology (every peer connects to every other peer) requires N\*(N-1)/2 connections and NAT traversal between arbitrary pairs. A star topology (one host, everyone connects to host) requires only N-1 connections, the host's IP is the only address anyone needs, and the host can relay messages trivially. For a LAN hackathon setting with 2-6 players, this is the right trade-off.

### Why HMAC challenge-response instead of just checking the passphrase?

Sending the passphrase over the wire (even over TLS) means the host sees and stores it. With HMAC challenge-response, the passphrase never leaves the client. The host sends a random challenge, the client proves it knows the passphrase by computing `HMAC-SHA256(challenge, SHA256(passphrase + salt))`, and the host verifies. This means even a compromised host never learns the passphrase.

### Why crypto/rand for card shuffling?

`math/rand` is predictable if you know the seed. For a multiplayer poker game, even a local one, using `crypto/rand` (Fisher-Yates with cryptographically random indices) ensures the deck is truly unpredictable. It's a single line difference and eliminates an entire class of fairness concerns.

### Why self-signed TLS certs?

The P2P connections need encryption (players' hole cards are transmitted), but we can't use CA-signed certs for ephemeral LAN connections. Self-signed ECDSA P-256 certs are generated in-memory at runtime — no files to manage, no cert authorities to configure. Authentication is handled separately by the HMAC handshake, so `InsecureSkipVerify` on the client side is safe: we don't trust the cert's identity, but we do get encrypted transport.

### Why IPv4-forced database connections?

Supabase's DNS returns IPv6 (AAAA) records first, but their connection pooler only has IPv4 addresses. On networks without IPv6 routing, the default Go dialer tries IPv6 first, hangs for 30+ seconds, then falls back to IPv4. The custom pgx dialer forces `"tcp4"` to skip the IPv6 attempt entirely — instant connections.

### Why read /dev/input directly instead of using gpm?

The standard approach for touchscreen input on a Linux TTY is `gpm` (General Purpose Mouse), which translates input device events into terminal mouse escape sequences. But `gpm` has issues with certain touchscreen controllers — the Raspberry Pi's official 7" display touchscreen wasn't being recognized through `gpm`'s device detection layer. Rather than debug `gpm` configuration, we bypassed the entire terminal mouse stack.

The touch listener opens `/dev/input/event1` directly and reads raw `input_event` structs from the kernel (24 bytes each on arm64: 16-byte timeval + 2-byte type + 2-byte code + 4-byte value). It tracks `ABS_X`/`ABS_Y` coordinates (and `ABS_MT_POSITION_X`/`ABS_MT_POSITION_Y` for multitouch controllers), then on `BTN_TOUCH` release (value=0, finger lift), scales the raw coordinates (0-1920, 0-1080) to terminal cell coordinates using the current terminal dimensions. The result is injected into Bubble Tea as a synthetic `tea.MouseMsg` via `p.Send()`, which means all existing `bubblezone` click handling works unchanged — every button, tab, and list item is tap-accessible with zero modifications to any tab's code.

Firing on finger **lift** rather than touch-down follows mobile UI conventions: it gives the user time to adjust finger position for precise targeting, avoiding accidental clicks from initial contact.

On non-Pi machines, `os.Open("/dev/input/event1")` simply fails and the goroutine exits silently — zero overhead, no error messages, no build tags needed.

### Why real CLI tools instead of decorative inventory items?

The original ITEMS tab was a thematic Fallout inventory (Stimpak, Nuka-Cola, etc.) — fun but non-functional. For a tool aimed at cybersecurity specialists and sysadmins, the tab real estate is better used as an integrated toolkit. The Fallout codenames (Signal Tracker for `dig`, Radar Scanner for `nmap`) maintain the aesthetic while making the tab genuinely useful.

Each tool is defined as a struct with a `BuildArgs` function that validates input and constructs the correct CLI invocation. This pattern makes adding new tools trivial — define the fields, write the arg builder, and the 4-state UI (list → form → running → output) handles everything else automatically. Commands run via `exec.CommandContext` with a 30-second timeout and a stored `context.CancelFunc` so users can abort long-running scans.

### Why JSON Lines for the P2P wire protocol?

Every message is a single JSON object terminated by a newline. This is trivially parseable with `bufio.Scanner`, human-readable for debugging (just `nc` into the port and watch), and extensible — adding a new message type means adding a new struct and a case statement. Binary protocols would be faster but add complexity for zero practical gain at 6 players exchanging poker actions.

---

## Project Structure

```
vegas-tui/
├── cmd/
│   ├── vegas/main.go              # TUI entry point
│   └── vegas-server/main.go       # HTTP server entry point
├── internal/
│   ├── app.go                     # Root model — boot, tab routing, mouse zones
│   ├── boot.go                    # Boot animation (ASCII logo, progress bar)
│   ├── updater.go                 # GitHub release checker, auto-updater
│   ├── stats.go                   # STATS tab — CPU/RAM/Temp/Disk gauges
│   ├── items.go                   # ITEMS tab — Cybersecurity toolkit (8 CLI tools)
│   ├── touch.go                   # Raw touchscreen input (Linux /dev/input)
│   ├── data.go                    # DATA tab — AI chat with quest parsing
│   ├── quests.go                  # QUESTS tab — Questlines + tasks
│   ├── projects.go                # PROJ tab — File browser + editor launch
│   ├── git.go                     # Git operations (status, stage, commit, push)
│   ├── radio.go                   # RADIO tab — Spotify controls + equalizer
│   ├── spotify.go                 # Spotify OAuth2 auth flow
│   ├── albumart.go                # ASCII album art renderer
│   ├── mapview.go                 # MAP tab — Scrollable ASCII map
│   ├── link.go                    # LINK tab — P2P chat + poker UI
│   ├── settingsview.go            # SET tab — Settings editor
│   ├── p2p/
│   │   ├── protocol.go            # Wire protocol types, HMAC crypto helpers
│   │   └── hub.go                 # TLS server/client, connection management
│   ├── games/
│   │   ├── cards.go               # Card types, deck, ASCII rendering, hand eval
│   │   └── holdem.go              # Texas Hold'em state machine + game engine
│   ├── theme/theme.go             # Color palette, lipgloss styles, ASCII art
│   ├── settings/settings.go       # Persistent JSON config + quest storage
│   ├── client/client.go           # HTTP client for server API
│   ├── config/config.go           # Server env var config
│   ├── db/db.go                   # PostgreSQL pool (IPv4-forced dialer)
│   └── server/
│       ├── server.go              # Chi router setup
│       ├── handlers/
│       │   ├── auth.go            # Supabase auth proxy
│       │   ├── chat.go            # Ollama relay with JSON errors
│       │   └── health.go          # Health check endpoint
│       └── middleware/auth.go     # JWT validation middleware
├── schema.sql                     # PostgreSQL schema (profiles + RLS)
├── scripts/schema-push.sh         # DB deployment (IPv4-aware DNS resolution)
├── go.mod                         # 11 direct dependencies
└── CLAUDE.md                      # LLM agent development context
```

---

## Installation & Deployment

### One-Line Install

```bash
# Install the TUI client
go install github.com/choice404/vegas-protocol/vegas-tui/cmd/vegas@latest

# Install the server (only needed for AI chat)
go install github.com/choice404/vegas-protocol/vegas-tui/cmd/vegas-server@latest
```

That's it. Single binary, no runtime dependencies, no config files required. It creates its config directory on first run.

### Build from Source

```bash
# Run locally (development)
go run ./cmd/vegas

# Build for current platform
go build -o vegas-protocol ./cmd/vegas

# Cross-compile for Raspberry Pi 5 (ARM64)
GOOS=linux GOARCH=arm64 go build -o vegas-protocol ./cmd/vegas

# Transfer to Pi and run
scp vegas-protocol pi@raspberrypi:~/
ssh pi@raspberrypi ./vegas-protocol

# Server (only needed for AI chat)
go run ./cmd/vegas-server
```

### Raspberry Pi Touch Setup

Touch input works by reading raw Linux input events directly from the kernel — no `gpm`, `fbterm`, or desktop environment required:

```bash
# Add your user to the input group (one-time)
sudo usermod -aG input $USER

# Or run with sudo
sudo ./vegas-protocol
```

The TUI automatically opens `/dev/input/event1` (the default Pi touchscreen device) at startup. If the device isn't found (non-Pi environments), it silently falls back to mouse-only input with zero overhead.

Override the device path with `VEGAS_TOUCH_DEVICE`:

```bash
# Use a different input device
VEGAS_TOUCH_DEVICE=/dev/input/event2 ./vegas-protocol

# Explicitly disable touch input
VEGAS_TOUCH_DEVICE=none ./vegas-protocol
```

---

## Judging Criteria Alignment

### Application — Real-World Problem Solving

V.E.G.A.S. Protocol solves a real problem for multiple audiences. Cybersecurity specialists attending conferences like DEF CON need a minimal, self-contained toolkit they can run on a Raspberry Pi without exposing a laptop — V.E.G.A.S. gives them system monitoring, a full suite of recon and network tools (dig, nmap, curl, whois, ping, traceroute, ss, openssl), encrypted P2P communication, task management, and local AI assistance in a single binary with no internet dependency. Terminal-first developers get a unified dashboard that replaces a dozen scattered tools. Headless sysadmins get a full interactive environment over SSH with no GUI dependencies.

The integrated cybersecurity toolkit is immediately practical: select a tool, fill in the parameters, and see output — all without leaving the TUI. The P2P layer is equally practical: encrypted chat and coordination between team members on a local network with zero infrastructure — no servers, no accounts, no cloud services. Just share a passphrase and connect. This is exactly what you want at a security conference or a hackathon with unreliable WiFi.

### Technicality — Implementation Quality

- **32 Go source files** across a clean, modular architecture
- **Elm Architecture (Model-Update-View)** with message-based state management
- **TLS-encrypted P2P networking** with HMAC challenge-response authentication — all from Go stdlib
- **Poker hand evaluation algorithm** that brute-forces all C(7,5)=21 combinations with kicker-based tiebreaking
- **Integrated cybersecurity toolkit** — 8 CLI tools with form-based parameter input, async execution, cancelable commands, and scrollable output
- **Raw touchscreen input** — reads Linux kernel `input_event` structs directly from `/dev/input`, scales to terminal coordinates, and injects synthetic mouse events into Bubble Tea's event loop
- **OAuth2 flow** with persistent token storage and automatic refresh for Spotify
- **IPv4-forced database dialer** to work around Supabase's IPv6-only DNS
- **Hot-restart auto-updater** using `syscall.Exec()` for seamless binary replacement
- **ASCII art renderer** that converts album cover images to Unicode block characters with brightness mapping
- **Cryptographically secure** card shuffling using `crypto/rand` Fisher-Yates
- **Cross-compilation** to ARM64 for Raspberry Pi deployment

### Creativity — Originality and Innovation

The Fallout: New Vegas Pip-Boy concept transforms a developer productivity tool into an immersive retro-futuristic experience. The AI assistant speaks in-character. System checks become "RADIATION SENSORS: NOMINAL." Project management becomes "Quest Logs." The poker game uses "CAPS" as chips. Real CLI tools become wasteland artifacts — `nmap` is a "Radar Scanner," `dig` is a "Signal Tracker," `curl` is a "Wasteland Courier." The entire UX is cohesive — not a theme slapped on top, but a ground-up design where every label, color, and interaction reinforces the aesthetic.

The cybersecurity toolkit is the bridge between novelty and utility: it takes what would be a themed demo project and makes it a tool you'd actually bring to DEF CON. The P2P poker implementation is equally novel: a full Texas Hold'em engine with ASCII card art, hand evaluation, and encrypted multiplayer — built with zero external networking dependencies, running in a terminal.

### Functionality — Stability and Completeness

- All 10 tabs are fully implemented and interactive
- Mouse/touch support works alongside keyboard shortcuts on every screen — touchscreen input reads raw kernel events for hardware-level reliability on bare TTYs
- All 8 cybersecurity tools execute with proper validation, timeout handling, and cancelable async execution
- Settings, quests, and Spotify tokens persist across sessions
- P2P connections handle authentication failures, disconnects, and peer departure gracefully
- The poker game handles all edge cases: all-in, split pots, wheel straights, fold-to-win
- The auto-updater checks, prompts, installs, and hot-restarts without user intervention
- Builds clean with `go build` and passes `go vet` with zero warnings

### Theme Alignment — Las Vegas

**Yes.** The project is a love letter to Las Vegas through the lens of Fallout: New Vegas:

- Named "V.E.G.A.S. Protocol"
- Full Pip-Boy terminal aesthetic (green-on-black, CRT styling, boot sequence)
- Cybersecurity tools with Fallout codenames (Signal Tracker, Radar Scanner, Wasteland Courier, Cipher Kit)
- The AI assistant is named V.E.G.A.S. and calls you "Courier"
- ASCII map of the Mojave Wasteland
- **CAPS Casino** — Texas Hold'em poker, the quintessential Las Vegas activity
- Quest system named after the RPG mechanic
- Radio tab echoing the in-game Mojave Music Radio

The Las Vegas theme isn't decorative — it's structural. The entire application is designed as if it were a real Pip-Boy operating system running in the Mojave Wasteland. The toolkit embodies this: real cybersecurity tools disguised as wasteland inventory items, functional enough for a DEF CON floor but themed enough to feel like looting an abandoned bunker.
