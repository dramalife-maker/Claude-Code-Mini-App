# Getting started from scratch

[← README](../README.md) · [繁體中文](getting-started.zh-TW.md)

Follow this guide to start the console on your computer, create a session, and receive your first agent reply. Get local use working first; connecting Telegram is optional.

## 1. Download the console

Open [Releases](https://github.com/jerry12122/Claude-Code-Mini-App/releases/latest) and download from Assets:

| Computer | Archive | How to start |
|---|---|---|
| Windows x64 | `claude-miniapp-vVERSION-windows-amd64.zip` | Desktop: `claude-miniapp-desktop.exe`; server only: `claude-miniapp.exe`. Choose one |
| Linux x64 | `claude-miniapp-vVERSION-linux-amd64.tar.gz` | Run `./claude-miniapp` from the extracted directory |
| Other platforms / architectures | No matching prebuilt archive | See [building from source](#building-from-source); your agent CLI must also support that environment |

Do not choose GitHub's automatic “Source code” archives unless you intend to build. Extract the **whole folder** to a writable location. Keep `internal/static/`, `config.example.yaml`, and the executables together. The downloaded console does not require Go or Node.js; your chosen CLI may have its own requirements.

The Windows desktop app needs [WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/). The executables are unsigned. If SmartScreen blocks them, verify that they came from this project's Release, then choose **More info → Run anyway**.

## 2. Install and sign in to one agent

**Install only the agent you want to use.** Install and authenticate on the computer running the console, under the same OS user account. Your phone is just a remote interface. Your agent account needs access and sufficient quota for its service.

| Agent in the UI | Official installation / sign-in guide | Command the console needs |
|---|---|---|
| Claude | [Claude Code quickstart](https://code.claude.com/docs/en/quickstart) | `claude` |
| Codex | [Codex CLI](https://developers.openai.com/codex/cli/) | `codex` |
| Cursor | [Cursor CLI installation](https://cursor.com/docs/cli/installation) | `cursor-agent` |
| Kiro ACP | [Kiro CLI](https://kiro.dev/docs/cli/) | `kiro-cli` |
| Antigravity | [Antigravity CLI](https://antigravity.google/docs/overview?tab=cli) | `agy` |

For example, install Claude using its official instructions:

Windows PowerShell:

```powershell
irm https://claude.ai/install.ps1 | iex
```

Linux / macOS:

```bash
curl -fsSL https://claude.ai/install.sh | bash
```

Open a new terminal after installation, run `claude --version`, then run `claude`. Complete the sign-in prompts and ask “Reply only OK. Do not use tools.” Exit the CLI after receiving a reply. For other agents, follow their official guide and complete one conversation in the terminal first; for example, running `codex` for the first time offers sign-in options.

Check the command location with `Get-Command claude` on Windows or `command -v claude` on Linux / macOS. Substitute your chosen command. If it is missing, fix PATH following the installer instructions, then restart the console.

**Cursor compatibility:** Current official instructions use `agent`, while this project invokes `cursor-agent`. Verify that `Get-Command cursor-agent` / `command -v cursor-agent` resolves an executable; having only `agent` or a PowerShell alias is insufficient. If your installation does not provide the compatible command, use another agent first.

A Windows console cannot directly invoke a CLI installed only inside WSL. If your CLI lives in WSL, run the Linux server in that same WSL environment and use its filesystem paths.

## 3. Create your configuration

Choose one route:

| Route | Use case | Telegram Bot Token needed? |
|---|---|---|
| A: Password sign-in (recommended) | Desktop, local browser, and optionally Telegram later | **Currently yes**, even for browser-only use |
| B: No-auth trial | Local development with network access isolated | No |

### A: Password sign-in

1. Open [@BotFather](https://t.me/BotFather) in Telegram, send `/newbot`, follow the prompts, and save the token. See the [official bot creation instructions](https://core.telegram.org/bots/tutorial#obtain-your-bot-token). Local use does not need a public URL yet.
2. In the extracted folder, copy `config.example.yaml` to `config.yaml` (not `config.yaml.txt`).
3. Open `config.yaml` in a text editor and **replace its entire contents** with the following. Replace both placeholder strings; do not keep an example password:

```yaml
bot_token: "PASTE_YOUR_BOTFATHER_TOKEN"
whitelist_tg_ids: []
no_auth: false
web:
  password: "REPLACE_WITH_YOUR_OWN_LONG_PASSWORD"
  allowed_cidrs:
    - "127.0.0.1/32"
    - "::1/128"
  trusted_proxies:
    - "127.0.0.1/32"
    - "::1/128"
  session_ttl: "168h"
server:
  port: 8080
db:
  path: "./claude-miniapp.db"
```

This initially permits password sign-in only from the local computer. Telegram access and notifications are configured later. **The desktop window uses this same password**, not your Telegram or agent password. Other options and defaults are in the [configuration example](../config.example.yaml).

### B: No-auth trial

Use this only when a firewall or network isolation already prevents other people from connecting. `no_auth` bypasses owner authentication; **the server still listens on all network interfaces, not just localhost**. Do not enable a tunnel, router port forwarding, or external firewall access.

Create `config.yaml` with:

```yaml
bot_token: ""
whitelist_tg_ids: []
no_auth: true
server:
  port: 8080
db:
  path: "./claude-miniapp.db"
```

If you cannot confirm network isolation, choose A. Before connecting Telegram later, switch to A and restart.

## 4. Start and sign in

**Windows desktop:** Double-click `claude-miniapp-desktop.exe`. Its window opens automatically. For A, enter your `web.password`; B opens directly.

**Windows server:** Open PowerShell in the extracted folder and run:

```powershell
.\claude-miniapp.exe
```

**Linux server:** Open a terminal in the extracted folder and run:

```bash
./claude-miniapp
```

Keep the server terminal open. On **that same computer**, visit <http://127.0.0.1:8080> and sign in according to A / B. Do not run desktop and server builds on the same port simultaneously; the desktop app already includes the server and accepts browser connections.

Closing the desktop window hides it to the tray. To exit or apply configuration changes, choose **結束 (Quit)** in the tray menu or press Ctrl+Q, then reopen. Stop the server with Ctrl+C. Changes to `config.yaml` require a restart.

## 5. Create your first session

The UI currently uses Traditional Chinese; the labels below match it.

1. Click **＋** in the sidebar to open **新建 Session** (New Session).
2. Choose the agent you installed, authenticated, and tested in step 2.
3. Set the working directory to an **existing absolute project directory on the server computer**, such as `C:/Users/you/Projects/demo` or `/home/you/projects/demo`. It is not a phone path or a GitHub URL. If you do not have a project yet, create an empty directory first.
4. Leave the name optional, permissions at their default, and Model / Effort empty to use the CLI defaults.
5. Enter “Reply only OK. Do not use tools or modify files.” Click **建立並送出** (Create and send). This is a real model request and uses your agent account's quota.
6. **Success means:** an agent reply appears, the running state ends, and you can send another message.

If an agent requests approval, inspect the operation before allowing it. You do not need to enable unrestricted permissions to complete this check. If no reply arrives, see [troubleshooting](#troubleshooting).

Models load in the background. If the selector only shows **預設** (Default), let startup fetching finish, or refresh models under **設定 → 一般** (Settings → General), then reopen the conversation. Fetch failures for agents you have not installed do not prevent using your installed agent.

## 6. Optional: Open from Telegram

Complete step 5 first. Use **route A with `no_auth: false`**. A tunnel on the same computer will connect a public HTTPS address to your local server.

### Add your Telegram user ID

Find your numeric User ID. The configuration example suggests [@userinfobot](https://t.me/userinfobot), a third-party bot: use it only to look up your ID, never send it tokens or passwords. Replace the empty allowlist in your configuration:

```yaml
whitelist_tg_ids:
  - 123456789 # Replace with your numeric User ID, not @username or the bot's ID
```

Restart the console. Open a **private chat with your own new bot** and press Start or send `/start` to allow notifications. This project does not implement a conversational bot command handler, so no welcome reply is expected. With one allowlisted user, browser sign-in automatically binds notifications to that user; sign out and back in if already logged in. By default, errors trigger notifications, successful replies do not.

### Get an HTTPS address

For a trial, [Cloudflare Quick Tunnel](https://developers.cloudflare.com/tunnel/get-started/quick-tunnels/) does not require your own domain. Open the [official cloudflared download page](https://developers.cloudflare.com/tunnel/downloads/). On Windows, choose Windows → Executable → 64-bit and save `cloudflared-windows-amd64.exe` in a folder. Keep the console running, open a separate PowerShell in that folder, and run:

```powershell
.\cloudflared-windows-amd64.exe tunnel --url http://127.0.0.1:8080
```

On Linux / macOS, install `cloudflared` following the same download page. Keep the console running and run this in a separate terminal:

```bash
cloudflared tunnel --url http://127.0.0.1:8080
```

Copy the printed `https://random-name.trycloudflare.com` URL. Keep that terminal open. Quick Tunnels are for testing; the URL changes when recreated. For ongoing use, configure a [tunnel with a stable hostname](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/get-started/create-remote-tunnel/) pointing to the same local service.

### Configure the Telegram button

1. Send `/setmenubutton` to @BotFather and select the bot created in step 3.
2. Follow the prompts to enter the HTTPS URL and button text, such as “Open console.” See the [official menu button instructions](https://core.telegram.org/bots/webapps#launching-mini-apps-from-the-menu-button).
3. Return to your bot's private chat and tap its menu button to **open inside Telegram**.
4. You should see the same sessions as on your computer. Send another test message and confirm the reply.

Pasting the public URL into an ordinary browser does not supply Telegram identity. The configuration above rejects non-local password sign-in; this is expected. Do not change `allowed_cidrs` to allow the whole Internet to work around it. Keep the shown `trusted_proxies` for a tunnel on the same computer; a proxy on another host needs its exact source CIDR. Repeat `/setmenubutton` if the tunnel URL changes.

## Troubleshooting

| Symptom | First check |
|---|---|
| Missing `config.yaml` / static page 404 | Start from the extracted folder; check that the file is not `.yaml.txt` and `internal/static/index.html` exists. Do not move just the executable |
| Missing `bot_token` | Route A needs a real bot token; password-only operation without a token is not yet supported |
| Wrong password / still asks to sign in | Use `web.password`. Fully quit and restart after editing configuration; desktop also requires sign-in |
| `僅允許內網存取` (LAN access only) / 403 | First try `127.0.0.1` on the host. For LAN access, add only your trusted subnet to `web.allowed_cidrs`. For Telegram, open through the bot menu and check your allowlisted ID |
| Port 8080 in use | Stop the previous server or tray app, or change `server.port` and update the browser / tunnel port too |
| `executable file not found` | Use `Get-Command` / `command -v` for the command in step 2 under the same user and environment. Fix PATH and restart. Cursor requires `cursor-agent` |
| Not authenticated / quota exhausted | Get the CLI to complete a conversation independently under the same user account. The console does not sign in or purchase quota for you |
| Working directory not found | Use an existing, accessible absolute path on the server; do not mix Windows and WSL paths |
| Empty model selector | Refresh in Settings → General and reopen the conversation. Ignore failures for uninstalled agents; inspect logs for installed ones |
| Blank / unstyled page | React, Babel, Tailwind, and other frontend assets require CDN access. Check connectivity or blocking rules, then reload. This is not a fully offline bundle |
| Telegram cannot open / 502 | Keep the host awake and both server and tunnel running; confirm the bot menu uses the current HTTPS URL |
| Telegram authentication fails | The token must belong to the bot whose menu you use; allowlist your own User ID. Close the Mini App and reopen through the bot menu |
| No Telegram notifications | Press Start in your own bot's chat; check notification ID, allowlist, and `notify` settings. Successful replies do not notify by default |

Open **設定 → 日誌** (Settings → Logs). If the UI cannot load, inspect `logs/server.log` in the program directory or launch the server build in a terminal. Include platform, agent, reproduction steps, and relevant errors when reporting a problem; redact tokens, passwords, and private content.

## Building from source

Install [Git](https://git-scm.com/downloads) and [Go 1.25 or newer](https://go.dev/dl/). Check `git --version` and `go version` first.

```bash
git clone https://github.com/jerry12122/Claude-Code-Mini-App
cd Claude-Code-Mini-App
```

Windows PowerShell:

```powershell
Copy-Item config.example.yaml config.yaml
# Edit config.yaml using step 3, then choose one build
go build -o claude-miniapp.exe ./cmd/server
.\claude-miniapp.exe
```

For the Windows desktop build instead:

```powershell
go build -ldflags "-H windowsgui" -o claude-miniapp-desktop.exe ./cmd/desktop
.\claude-miniapp-desktop.exe
```

Linux / macOS server:

```bash
cp config.example.yaml config.yaml
# Edit config.yaml using step 3
go build -o claude-miniapp ./cmd/server
./claude-miniapp
```

Continue with steps 4 and 5. The first build downloads Go dependencies; no separate frontend build is needed. This project does not provide a macOS desktop window.

## Restarting and updating

Start from the same program directory to retain your sessions. Before updating, fully stop the app and back up `config.yaml`, your database (default `claude-miniapp.db`), and `workspace/`. Replace the executable and the entire `internal/static/` directory. Do not overwrite your configuration with the example. Restoring or moving agent conversations also requires preserving that agent's own local data. See [all features](features.md) for more.
