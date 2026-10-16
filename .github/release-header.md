## 下載

| 平台 | 檔案 | 啟動 |
|---|---|---|
| Windows（桌面版視窗＋系統匣，推薦） | `claude-miniapp-@TAG@-windows-amd64.zip` | 雙擊 `claude-miniapp-desktop.exe` |
| Windows（純 server，用瀏覽器／Telegram 連） | 同上 zip | 執行 `claude-miniapp.exe` |
| Linux（純 server） | `claude-miniapp-@TAG@-linux-amd64.tar.gz` | `./claude-miniapp` |

## 首次設定

完整教學：[繁體中文](https://github.com/jerry12122/Claude-Code-Mini-App/blob/@TAG@/docs/getting-started.zh-TW.md) · [English](https://github.com/jerry12122/Claude-Code-Mini-App/blob/@TAG@/docs/getting-started.md)。解壓後也可直接開啟 `docs/getting-started.zh-TW.md`。

1. **解壓整個資料夾**：執行檔必須與 `internal/static/`（前端）放在一起，不能只搬 exe。
2. 在同一台機器、同一使用者下安裝並登入一個代理（`claude`、`codex`、`kiro-cli`、`cursor-agent`、`agy`），先在終端機確認可以對話。
3. 把 `config.example.yaml` 複製為 `config.yaml`，依教學填入 `bot_token`、更換 `web.password`。目前即使只用桌面／網頁密碼登入仍需 Bot Token；Telegram 入口另需白名單與 BotFather 選單設定。
4. 桌面與 server 二選一啟動。server 版在主機開 <http://127.0.0.1:8080>；桌面版自動開視窗。輸入 `web.password` 後建立 Session，填主機上的工作目錄並送測試訊息。

`no_auth: true` 僅供已隔離的本機開發環境：它不會限制 server 只監聽 localhost，不可搭配公開 Tunnel。

> **Windows 注意**
> - 執行檔未簽章，SmartScreen 可能跳「Windows 已保護您的電腦」→ 點「其他資訊」→「仍要執行」。
> - 桌面版需要 WebView2 Runtime：Windows 11 內建，Windows 10 請先[安裝](https://developer.microsoft.com/microsoft-edge/webview2/)。
> - 桌面版關閉視窗只會縮到系統匣，要結束請在匣圖示按右鍵 →「結束」（或 Ctrl+Q）。
> - 驗證檔案完整性：`checksums.txt` 內有 SHA256。

---

## 更新內容
