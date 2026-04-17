# Claude Code Mini App

[繁體中文](README.zh-TW.md)

A Telegram Mini App that lets you remotely drive AI coding CLIs on your server from your phone. Shipped as a **single Go binary** that serves a REST API, WebSocket, and a single-file React SPA (no separate frontend build).

## Features

- **Multiple backends** — When creating a session, choose **Claude Code**, **Cursor Agent**, or **Gemini CLI** (Codex is reserved in the UI; no runner yet)
- **Remote sessions** — Send prompts via the Telegram Mini App or a browser on your LAN; each backend runs the corresponding CLI on the server
- **Session management** — Create, rename, and delete conversations; each session has `work_dir`, permission mode, and `agent_type`
- **Real-time streaming** — Bidirectional WebSocket with streamed Markdown; multi-tab / multi-connection broadcast sync
- **Permissions** — For Claude, `stream-json` `permission_denials` enters an approval flow; allow once or change `permission_mode` (Cursor / Gemini also support mode switching per their CLIs)
- **Telegram auth** — Mini App `initData` HMAC verification plus an allowlisted `tg_id`
- **Web login** — Password login from allowed intranet IPs (HttpOnly cookie), with optional binding to an allowlisted user for **Telegram task-complete / approval notifications**

## Architecture

```
Telegram Mini App / browser
        ↕ WebSocket (/sessions/:id/ws)
┌─────────────────────────────────────────┐
│           Single Go binary               │
│  Fiber │ SQLite (WAL) sessions/messages │
│  One subprocess per user message         │
│  agent.Runner (claude / cursor / gemini) │
│  Non-interactive + stream events → UI    │
└─────────────────────────────────────────┘
```

- Each message **spawns a short-lived subprocess**; **no PTY**.
- Claude uses `-p`, `--output-format stream-json`, `--resume <agent_session_id>`, etc. (see `docs/headless.md`, `docs/claude-code-cli.md`).
- Cursor and Gemini use dedicated runners and event mapping (see `docs/cursor-agent-cli.md`, `docs/gemini-cli.md`).

## Prerequisites

- Go 1.25+
- The CLI(s) you plan to use installed and authenticated on the server (e.g. `claude`, `cursor agent`, `gemini`)
- A Telegram bot token from [@BotFather](https://t.me/BotFather)

## Setup

### 1. Clone and build

```bash
git clone https://github.com/jerry12122/Claude-Code-Mini-App
cd claude-miniapp
go build -o claude-miniapp ./cmd/server
```

### 2. Configure

```bash
cp config.example.yaml config.yaml
```

**`config.yaml` essentials:**

```yaml
bot_token: "YOUR_BOT_TOKEN_HERE"

whitelist_tg_ids:
  - 123456789  # Allowlisted Telegram user IDs

web:
  # Web login password (JSON body to POST /auth/login — not query params)
  password: "change-me"
  # Only these CIDRs may use web password login (real IP: CF-Connecting-IP > X-Forwarded-For > direct)
  allowed_cidrs:
    - "127.0.0.0/8"
    - "10.0.0.0/8"
    - "172.16.0.0/12"
    - "192.168.0.0/16"
  session_ttl: "24h"
  # Default Telegram user for notifications on web login (must be allowlisted); set when multiple users exist
  # default_notify_tg_id: 123456789

no_auth: false  # true = skip all auth (local dev only)

server:
  port: 8080

db:
  path: "./claude-miniapp.db"
```

> **Security:** Do not commit `config.yaml` with real secrets. Never use `no_auth` in production.

### 3. Run

```bash
./claude-miniapp
```

Listens on `:8080` by default. Static UI is served from `./internal/static`.

## Configuration reference

| Field | Description | Default |
|---|---|---|
| `bot_token` | Telegram Bot API token | Required unless `no_auth` |
| `whitelist_tg_ids` | Allowlisted Telegram user IDs | `[]` |
| `web.password` | Web login password | `""` (web login disabled if empty) |
| `web.allowed_cidrs` | Source IPs allowed for web password login | Private RFC1918 ranges |
| `web.session_ttl` | Session cookie lifetime | `24h` |
| `web.default_notify_tg_id` | Default notify target for web login | `0` (unset) |
| `no_auth` | Disable all authentication | `false` |
| `server.port` | HTTP port | `8080` |
| `db.path` | SQLite file path | `./claude-miniapp.db` |

## REST API

| Method | Path | Description |
|---|---|---|
| `GET` | `/sessions` | List sessions |
| `POST` | `/sessions` | Create session (JSON: `name`, `description`, `work_dir`, `permission_mode`, `agent_type`, …) |
| `PATCH` | `/sessions/:id` | Rename (`{"name":"..."}`) |
| `DELETE` | `/sessions/:id` | Delete session |
| `GET` | `/sessions/:id/messages` | Message history |
| `POST` | `/auth/login` | Web login (only from `allowed_cidrs`) |
| `POST` | `/auth/logout` | Log out and clear cookie |
| `WS` | `/sessions/:id/ws` | WebSocket chat |

Except static files and auth routes, endpoints require Telegram `initData` (header `X-Telegram-Init-Data` or query) or a valid web session cookie.

## WebSocket protocol (summary)

**Client → Server:**

```json
{ "type": "input", "data": "user prompt" }
{ "type": "allow_once", "tools": ["Write"] }
{ "type": "set_mode", "mode": "acceptEdits" }
{ "type": "interrupt" }
{ "type": "reset_context" }
```

**Server → Client:**

```json
{ "type": "sync", "value": "IDLE", "messages": [...] }
{ "type": "status", "value": "STREAMING" }
{ "type": "delta", "content": "..." }
{ "type": "user_message", "content": "..." }
{ "type": "permission_request", "tools": [...] }
{ "type": "reset" }
{ "type": "error", "content": "..." }
```

On connect, the client receives `sync` (UI state + history). Background work and approval state are reflected in the session `status` field (`idle`, `running`, `awaiting_confirm`).

## Permission modes (Claude / Cursor / Gemini)

| Mode | Description |
|---|---|
| `default` | Default; writes/execution follow each CLI’s policy |
| `acceptEdits` | Looser file edits (Claude: `--permission-mode acceptEdits`) |
| `bypassPermissions` | Skips permission checks (high risk; Cursor passes extra force behavior in this mode) |

For Claude denials, the UI can allow once or switch mode permanently; non-Claude agents ignore `allow_once`.

## Telegram notifications

If the request is tied to a `tg_id` (in-app Telegram or web login bound to an allowlisted user), the bot can send short messages when a task finishes or approval is needed.

## Tech stack

| Layer | Technology |
|---|---|
| Backend | Go, [Fiber](https://gofiber.io/) |
| Database | SQLite (WAL) via `modernc.org/sqlite` |
| WebSocket | `gofiber/contrib/websocket` |
| Config | [Viper](https://github.com/spf13/viper) (`config.yaml`) |
| Frontend | Single-file React SPA (`internal/static`) |
| Auth | Telegram `initData` HMAC-SHA256; web session cookie + IP CIDR |

## Further docs

- `docs/plan.md` — Specification and roadmap  
- `docs/headless.md` — Claude `-p` and stream-json  
- `docs/claude-code-cli.md`, `docs/cursor-agent-cli.md`, `docs/gemini-cli.md` — CLI references  

## License

MIT
