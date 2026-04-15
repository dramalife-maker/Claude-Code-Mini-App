# Claude Code Mini App

A Telegram Mini App that lets you remotely control `claude code` CLI from your mobile device. Built as a single Go binary serving REST API, WebSocket, and a single-file React SPA.

## Features

- **Remote Claude Code access** — Send prompts to `claude code` on your server from anywhere via Telegram
- **Session management** — Create, resume, and manage multiple Claude sessions with persistent history
- **Real-time streaming** — Live markdown rendering with streaming output via WebSocket
- **Permission control** — Interactive permission approval flow when Claude wants to write files or run commands
- **Telegram authentication** — Whitelist-based access via Telegram Mini App `initData` signature verification
- **Web fallback** — Password-based access from regular browsers (non-Telegram)

## Architecture

```
Telegram Mini App (WebView)
        ↕ WebSocket (:8080/sessions/:id/ws)
┌─────────────────────────────────────┐
│         Go Single Binary            │
│  ┌─────────┐  ┌──────────────────┐  │
│  │  Fiber  │  │   SQLite (WAL)   │  │
│  │ Router  │  │  sessions/msgs   │  │
│  └────┬────┘  └──────────────────┘  │
│       │ spawn per message           │
│  claude -p --resume <id>            │
│         --output-format stream-json │
│         --permission-mode <mode>    │
└─────────────────────────────────────┘
```

Each message spawns a short-lived `claude -p` subprocess using `--output-format stream-json` and `--resume <session_id>` for conversation continuity. No PTY required.

## Prerequisites

- Go 1.25+
- [`claude` CLI](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated on the server
- A Telegram Bot token (create one via [@BotFather](https://t.me/BotFather))

## Setup

### 1. Clone and build

```bash
git clone https://github.com/your-username/claude-miniapp
cd claude-miniapp
go build -o claude-miniapp ./cmd/server
```

### 2. Configure

Copy the example config and fill in your values:

```bash
cp config.example.yaml config.yaml
```

**`config.yaml`:**

```yaml
bot_token: "YOUR_BOT_TOKEN_HERE"

whitelist_tg_ids:
  - 123456789  # Your Telegram user ID

# Password for browser access (non-Telegram)
web_password: "your-secure-password"

# true = skip all auth (dev only, never use in production)
no_auth: false

server:
  port: 8080

db:
  path: "./claude-miniapp.db"
```

> **Security:** Never commit `config.yaml` with real credentials. Add it to `.gitignore`.

### 3. Run

```bash
./claude-miniapp
```

The server starts on port `8080`. Open it from your Telegram Mini App or browser.

## Configuration Reference

| Field | Description | Default |
|---|---|---|
| `bot_token` | Telegram Bot API token | required |
| `whitelist_tg_ids` | Allowed Telegram user IDs | `[]` |
| `web_password` | Password for browser access | `""` (disabled) |
| `no_auth` | Disable all authentication | `false` |
| `server.port` | HTTP listen port | `8080` |
| `db.path` | SQLite database path | `./claude-miniapp.db` |

## API Reference

### REST Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/sessions` | List all sessions |
| `POST` | `/sessions` | Create a new session |
| `DELETE` | `/sessions/:id` | Delete a session |
| `GET` | `/sessions/:id/messages` | Get message history |
| `WS` | `/sessions/:id/ws` | WebSocket connection |

### WebSocket Messages

**Client → Server:**

```json
{ "type": "input",      "data": "ls -al" }
{ "type": "allow_once", "tools": ["Write"] }
{ "type": "set_mode",   "mode": "acceptEdits" }
{ "type": "interrupt" }
```

**Server → Client:**

```json
{ "type": "status",             "value": "STREAMING" }
{ "type": "delta",              "content": "### Hello" }
{ "type": "permission_request", "tools": [{ "name": "Write", "input": { "file_path": "..." } }] }
{ "type": "result",             "session_id": "...", "cost_usd": 0.01 }
```

## Permission Modes

Each session has a `permission_mode` that controls how Claude handles file writes and command execution:

| Mode | Description |
|---|---|
| `default` | Triggers approval flow for writes/execution |
| `acceptEdits` | Auto-approves file reads/writes, Bash still requires approval |
| `bypassPermissions` | Skips all permission checks (dangerous) |

When Claude is denied a tool, the frontend shows an approval dialog. The user can allow once or switch the session to a more permissive mode.

## Session States

| State | Description |
|---|---|
| `IDLE` | Ready for input |
| `THINKING` | Prompt sent, waiting for first output |
| `STREAMING` | Receiving streamed response |
| `AWAITING_CONFIRM` | Waiting for user to approve a tool use |

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go, [Fiber](https://gofiber.io/) |
| Database | SQLite (WAL mode) via `modernc.org/sqlite` |
| WebSocket | `gofiber/contrib/websocket` |
| Frontend | Single-file React SPA (no build step) |
| Auth | Telegram `initData` HMAC-SHA256 verification |

## Security Notes

- Telegram `initData` is verified using HMAC-SHA256 with your `BOT_TOKEN`
- Only Telegram user IDs in `whitelist_tg_ids` are granted access
- `bypassPermissions` mode disables all Claude tool approval — use only in trusted environments
- The `no_auth: true` flag is for local development only and must never be used in production

## License

MIT
