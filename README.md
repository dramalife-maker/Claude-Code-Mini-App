# Claude Code Mini App

> A self-hosted **multi-agent console** for Claude Code, Cursor Agent, Codex, Kiro and Antigravity. Run and supervise all your coding agents in one place — from a desktop window, any browser, or a Telegram Mini App on your phone. **One Go binary**, no separate frontend build.

[![Version](https://img.shields.io/badge/version-0.12.0-blue)](#) [![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE) [![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)](#)

[繁體中文](README.zh-TW.md)

![Agent Console: sessions for Claude, Codex, Kiro and Cursor grouped by project, with a streaming chat on the right](docs/images/console.png)

<sub>Demo data on a throwaway instance. The UI is shown in Traditional Chinese.</sub>

Your agents keep running on the machine where your code lives; this app is the console in front of them. Each conversation is a **session** bound to a `work_dir`, an agent, and a permission mode, so you can juggle several agents across several projects, see which ones are working or waiting for approval, and step in from wherever you are.

| Way in | For | Auth |
|---|---|---|
| 🖥️ **Desktop window** (Windows) | Daily use at your PC — tray icon, Ctrl+P switcher | same web password as the browser |
| 🌐 **Browser** | Any device on your LAN, or remote through a reverse proxy / tunnel | web password (restricted to `web.allowed_cidrs`) |
| 📱 **Telegram Mini App** | Away from your desk — phone push on errors, approvals on the go | Telegram `initData` + allowlist |

All three talk to the same server and see the same sessions in real time.

## 🚀 Quick Start

**First time? Follow [Getting started from scratch](docs/getting-started.md)** — installation, configuration, sign-in, and your first successful agent reply, with separate Windows and Linux commands. Downloaded builds do not require Go or a frontend build.

1. Download the Windows x64 zip or Linux x64 tar.gz from [Releases](https://github.com/jerry12122/Claude-Code-Mini-App/releases/latest). Extract the whole folder, including `internal/static/`.
2. Install and sign in to **one** agent on that computer. [CLI links and checks](docs/getting-started.md#2-install-and-sign-in-to-one-agent) include the required command names.
3. Copy `config.example.yaml` to `config.yaml` and follow the [configuration steps](docs/getting-started.md#3-create-your-configuration). Password sign-in currently requires a Bot Token even without Telegram remote access; replace the example password.
4. Run **one** executable: Windows desktop `claude-miniapp-desktop.exe`, Windows server `claude-miniapp.exe`, or Linux server `./claude-miniapp`. For server builds, open <http://127.0.0.1:8080> on the host and enter `web.password`.
5. [Create a session and send a test message](docs/getting-started.md#5-create-your-first-session). The working directory is a path on the server computer.

**Telegram remote access:** complete local setup first, then follow [HTTPS tunnel, allowlist, and BotFather menu setup](docs/getting-started.md#6-optional-open-from-telegram). A Bot Token alone does not configure a Mini App button.

> [!WARNING]
> `no_auth: true` disables owner authentication but does **not** bind the server to localhost. Use it only in an isolated local development environment, never with a public tunnel.

[Troubleshooting](docs/getting-started.md#troubleshooting) · [Build from source (Go 1.25+)](docs/getting-started.md#building-from-source) · [Desktop window and shortcuts](docs/features.md#desktop-window-windows)

## 🔄 Desktop auto-update (Windows)

The Windows desktop app can check GitHub Releases and update **itself**. Only `claude-miniapp-desktop.exe` is replaced.

- **Upgrade once by hand first.** Versions before the updater cannot update themselves. Download the new `claude-miniapp-v<X.Y.Z>-windows-amd64.zip` and replace `claude-miniapp-desktop.exe` in your existing folder, keeping `config.yaml` and your data. After that, updates happen inside the app.
- **Check:** tray → Check for updates, or Settings → Desktop update. The app also checks at startup and every 6 hours, and only shows the result. It never downloads or restarts on its own.
- **Download and restart are separate clicks.** Download starts only when you click it. Restart and update also needs a click, and the app refuses it while work is busy.
- **Busy refusal:** a restart is refused (`update_busy`) while any session has a running task, a shell command, a pending approval, or **any queued message, even if the queue is paused**. Finish the work or clear the queue, then try again. The downloaded update is kept.
- **Writable folder required:** the folder containing the exe must be writable by your user. The updater does not ask for administrator rights.
- **Kept as-is:** `config.yaml`, `claude-miniapp.db`, `workspace/` (attachments), `logs/`, and `internal/static/` are not changed. Only `claude-miniapp-desktop.exe` is replaced.
- **Not auto-updated:** the Windows server (`claude-miniapp.exe`) and Linux builds. Update those by hand.
- **Manual restore:** before upgrading, keep a backup copy of the current exe. To go back, close the app and copy the backup over the exe:

```powershell
Copy-Item .\claude-miniapp-desktop.exe .\claude-miniapp-desktop.previous.exe
# To roll back: close the app, then
Copy-Item .\claude-miniapp-desktop.previous.exe .\claude-miniapp-desktop.exe -Force
```

  Only the exe files are touched. Data is not downgraded automatically.
- **Integrity:** downloads are checked against the SHA256 in `checksums.txt`. That proves the file matches the release. It does **not** prove who published it, because there is no signature check yet.
- **Network:** checking and downloading need access to GitHub. An update does not make the app work offline.

## ✨ Features

- **Multi-agent** — Claude Code, Cursor Agent, Codex, Kiro ACP and Antigravity, chosen per session. Tool approvals show the full command or file content before you allow them
- **Live chat** — WebSocket streaming with Markdown, syntax-highlighted code and one-click copy; multiple tabs stay in sync
- **Sessions** — each bound to a `work_dir` and permission mode, with real-time unread and needs-approval badges, a Ctrl/Cmd+P quick switcher, and an account-quota badge in the header
- **Message queue** — messages sent while a task runs are queued, survive restarts, and pause on failure or interrupt
- **Attachments** — paste or drag & drop images and files; the agent gets them by path, stored outside your `work_dir`
- **Temporary sharing** — link + 6-digit PIN + nickname, default 1 hour; guests join as read-only `viewer` or full-access `editor`
- **MCP server** — let other agents operate sessions, read history and query activity; `ask_session` has a hop limit against agent-to-agent loops
- **Shell & logs** — optional shell in `work_dir` (off by default) and a live server-log viewer in Settings → Logs

Every feature in detail: [docs/features.md](docs/features.md).

## 🤔 Why this?

| | SSH + tmux | Generic Telegram bot | **This app** |
|---|---|---|---|
| Multiple agents, one view | One terminal each | One bot, one tool | Claude / Cursor / Codex / Kiro ACP / Antigravity side by side |
| Session state | Manual | Usually none | Persisted, with unread and approval badges |
| Collaboration | Share a terminal | None | Temporary PIN link, viewer / editor |
| Agent-to-agent | You wire it | None | Built-in MCP server (`ask_session`) |

## 🏗️ Architecture

```
Desktop window · Browser · Telegram Mini App · Share guests
        ↕ WebSocket / REST          MCP clients ↔ POST /mcp
┌─────────────────────────────────┐
│   Go binary (Fiber + SQLite)    │
│   Sessions · queue · sharing    │
│   spawn CLI per message (no PTY)│
│   QuotaService (cached fetch)   │
└────────────────┬────────────────┘
                 ↓
  claude · cursor-agent · codex · kiro-cli (ACP) · agy
```

Each user message spawns a short-lived subprocess. Details: [`docs/spec/plan.md`](docs/spec/plan.md), [`docs/spec/headless.md`](docs/spec/headless.md).

## 🔒 Security

> [!WARNING]
> Shell, shares and the MCP token all grant control of the host. Enable them only on networks you trust, and never use `no_auth` in production.

- Keep real secrets out of git.
- **`shell.enabled`** — shell access on the host for authenticated users. Allowlist rules: [`docs/spec/shell-allowlist-schema.md`](docs/spec/shell-allowlist-schema.md).
- **Shares** — anyone holding the link and PIN can join until it expires or you end it; an `editor` has the same power as you (sending messages, approving tool use), so prefer `viewer`.
- **`mcp_token`** — full session control (including shell) for any client holding it; treat it like `bot_token` and expose `/mcp` only on trusted networks.
- **Uploads** — any authenticated user can write files of any type and size into `./workspace/uploads/<session_id>/` (not your `work_dir`), independent of `shell.enabled`.

## 📚 Documentation

| Topic | Path |
|---|---|
| Install, sign in, and get your first reply | [`docs/getting-started.md`](docs/getting-started.md) |
| All features in detail | [`docs/features.md`](docs/features.md) |
| Spec & API / WebSocket | [`docs/spec/plan.md`](docs/spec/plan.md) |
| Config reference | [`config.example.yaml`](config.example.yaml) |
| Claude / Cursor / Codex / Kiro / Antigravity CLI | [`docs/spec/`](docs/spec/) |

## Testing

This project is tested with BrowserStack.

## License

MIT
