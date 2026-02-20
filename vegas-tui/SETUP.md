# V.E.G.A.S. Protocol - Setup Guide

## Quick Start (Chat-Only Mode)

The simplest way to run V.E.G.A.S. — **no database required**:

```bash
# 1. Install Ollama (https://ollama.ai)
curl -fsSL https://ollama.ai/install.sh | sh

# 2. Pull a model
ollama pull llama3

# 3. Start the server (chat relay only)
cd vegas-tui
go run ./cmd/server

# 4. In another terminal, start the TUI
cd vegas-tui
go run ./cmd/tui
```

The server auto-detects that no database is configured and runs in **chat-only mode** (just the `/api/chat` Ollama relay).

---

## Full Setup (With Supabase Database)

### 1. Create a Supabase Project

1. Go to [supabase.com](https://supabase.com) and create a new project.
2. Wait for the project to finish provisioning.

### 2. Get Your Credentials

From your Supabase project dashboard:

- **Project Settings > API**:
  - `SUPABASE_URL` — Your project URL (e.g., `https://abcdefgh.supabase.co`)
  - `SUPABASE_ANON_KEY` — The `anon` / `public` key
- **Project Settings > API > JWT Settings**:
  - `SUPABASE_JWT_SECRET` — The JWT secret
- **Project Settings > Database > Connection string > URI**:
  - `DATABASE_URL` — The PostgreSQL connection string
  - Use **Session mode** (port 5432) for direct connections

### 3. Configure Environment

```bash
cd vegas-tui

# Copy the example env file
cp .env.example .env

# Edit .env with your Supabase credentials
```

Your `.env` should look like:

```env
DATABASE_URL=postgresql://postgres:YOUR_PASSWORD@db.YOUR_PROJECT_REF.supabase.co:5432/postgres
SUPABASE_URL=https://YOUR_PROJECT_REF.supabase.co
SUPABASE_ANON_KEY=eyJhbGciOiJIUzI1NiIs...
SUPABASE_JWT_SECRET=your-jwt-secret-here
SERVER_PORT=8080
OLLAMA_URL=http://localhost:11434
```

### 4. Push the Database Schema

```bash
# Make sure DATABASE_URL is set in .env or exported
cd vegas-tui
bash scripts/schema-push.sh
```

This creates the `profiles` table and row-level security policies.

### 5. Run the Full Stack

```bash
# Terminal 1: Start Ollama
ollama serve

# Terminal 2: Start the server
cd vegas-tui
go run ./cmd/server

# Terminal 3: Start the TUI
cd vegas-tui
go run ./cmd/tui
```

---

## Building for Raspberry Pi

```bash
cd vegas-tui

# Cross-compile for ARM64 (Raspberry Pi 5)
GOOS=linux GOARCH=arm64 go build -o vegas-protocol ./cmd/tui

# Copy to Pi
scp vegas-protocol pi@<PI_IP>:~/

# On the Pi: install GPM for touch support
sudo apt-get install gpm
sudo systemctl enable gpm
sudo systemctl start gpm

# Run it
./vegas-protocol
```

---

## Environment Variables Reference

| Variable              | Required | Default                  | Description                        |
| --------------------- | -------- | ------------------------ | ---------------------------------- |
| `DATABASE_URL`        | No*      | —                        | PostgreSQL connection string       |
| `SUPABASE_URL`        | No*      | —                        | Supabase project URL               |
| `SUPABASE_ANON_KEY`   | No*      | —                        | Supabase anonymous API key         |
| `SUPABASE_JWT_SECRET` | No*      | —                        | JWT signing secret                 |
| `SERVER_PORT`         | No       | `8080`                   | HTTP server port                   |
| `OLLAMA_URL`          | No       | `http://localhost:11434` | Ollama API endpoint                |

*Required only for auth/database features. Server runs in chat-only mode without them.

---

## Troubleshooting

**Server says "config: DATABASE_URL is required"**
- You're running in full mode. Either set all Supabase env vars or they'll be auto-skipped in chat-only mode (after the server fix).

**TUI shows "CONNECTION LOST - RAD INTERFERENCE"**
- Make sure the server is running on the expected port
- Make sure Ollama is running (`ollama serve`)
- Check that the model is pulled (`ollama list`)

**No mouse/touch support in terminal**
- On Raspberry Pi: Install and start `gpm` (see Pi Setup above)
- On desktop: Most modern terminals support mouse events natively
- On tmux: Add `set -g mouse on` to your tmux config
