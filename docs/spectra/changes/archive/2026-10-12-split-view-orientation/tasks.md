## 1. 雙向版面與分割條元件擴充 (Layout & Components)

- [x] 1.1 在 `SplitChatContainer` 實作雙向排列切換，依據 `Decision: Dual-direction Flexbox container in SplitChatContainer` 支援 `orientation` 為 `horizontal`（左右並排）與 `vertical`（上下堆疊）之切換，落實 `Split pane container layout` 需求；以 DevTools 檢視並確認 vertical 模式下主副面板垂直堆疊且高度各佔 50%。
- [x] 1.2 擴充 `SplitDivider` 支援軸向自適應與 Y 軸拖曳，落實 `Decision: Orientation-aware SplitDivider with dynamic drag plane`，在直向模式下呈現水平分隔線並計算 Y 軸高度比例（施加最小 200px 高度限制），落實 `Resizable pane divider` 需求；拖曳分隔線驗證上下高度平滑調整與邊界保護。
- [x] 1.3 實作分割方向切換按鈕與快捷鍵 `Ctrl+Alt+O`，落實 `Decision: Explicit orientation toggle control and shortcut Ctrl+Alt+O`，提供直觀一鍵切換左右或上下分屏，落實 `Split orientation switching` 需求；透過點擊頂欄按鈕及按下快捷鍵驗證佈局即時切換。

## 2. 狀態管理與持久化儲存 (State & Persistence)

- [x] 2.1 在 `App.js` 擴充 `splitOrientation` 狀態與偏好持久化，落實 `Decision: LocalStorage state persistence for orientation`，將方向設定寫入並讀取 `miniapp_split_chat_state.orientation`，落實 `Split orientation persistence` 需求；透過重新整理瀏覽器驗證上下分屏設定維持不變。
- [x] 2.2 串接 `chat-header.js` 與頂欄控制元件，讓使用者在主副視窗皆可直觀觀察當前分割狀態與觸發方向切換；點擊操作驗證選單反饋與狀態同步。

## 3. 品質把關與驗證 (Verification & Audit)

- [x] 3.1 執行全專案後端自動化測試 `go test ./...`，驗證伺服器與 WebSocket 後端無任何回歸問題；確認全部測試通過。
- [x] 3.2 執行 ponytail-review 審查前端分割實作，依據 `Decision: Ponytail-review simplicity criteria for orientation handling` 確認零額外依賴、無冗餘狀態機與乾淨的 Flexbox 佈局；完成審查並確認架構維持極簡。
- [x] 3.3 執行 Spectra 分析與驗證指令 `spectra.exe analyze split-view-orientation` 與 `spectra.exe validate split-view-orientation`，確認工件一致性與規範無缺漏。
