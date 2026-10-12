## 1. 介面佈局與分割容器元件 (UI & Layout)

- [x] 1.1 建立雙欄分割容器元件 `SplitChatContainer`，依據 `Decision: Independent React component instances for ChatView` 實作雙欄渲染與折疊邏輯，達成 `Split pane container layout` 需求；以 Chrome 檢視並確認切換分割按鈕時雙欄顯示與全寬還原正常。
- [x] 1.2 實作可拖曳分割條元件 `SplitDivider`，支援滑鼠拖曳變更欄寬比例、雙擊重設為 50/50，並加入最小 320px 邊界限制，達成 `Resizable pane divider` 需求；手動拖曳驗證比例變化並確認不會縮小至 320px 以下。
- [x] 1.3 實作面板焦點追蹤與視覺高亮，在點擊或熱鍵切換時標示當前活動欄位，達成 `Active pane focus indication` 需求；透過點擊與熱鍵驗證焦點切換與快捷鍵路由。
- [x] 1.4 實作視窗寬度監聽與自適應收合，落實 `Decision: Responsive collapse below 900px breakpoint`，在螢幕寬度小於 900px 或 TMA 模式下自動折疊為分頁切換，達成 `Responsive narrow viewport fallback` 需求；以 DevTools 模擬小螢幕驗證自動折疊與分頁標籤切換。

## 2. 會話與連線狀態管理 (State & WebSocket)

- [x] 2.1 在 `App.js` 擴充副面板會話狀態管理，支援點選不同會話與快顯選單指定開啟面板，達成 `Independent session selection per pane` 需求；建立兩個不同會話並驗證左右兩欄各自顯示獨立的會話內容。
- [x] 2.2 串接面板獨立的 WebSocket 連線，依據 `Decision: Pane-scoped WebSocket state management` 確保雙向通訊各自獨立，達成 `Concurrent WebSocket streaming` 需求；同時在兩欄觸發不同 agent 任務，確認 token 增量串流與工具卡片同時獨立更新且無競爭。
- [x] 2.3 確保輸入框、中斷停止按鈕與授權彈窗完全隔離，達成 `Independent prompt submission and tool approval` 需求；在一欄跳出工具授權確認時，確認另一欄仍可正常打字與送出 prompt。
- [x] 2.4 實作分割狀態與面板比例的持久化儲存，落實 `Decision: LocalStorage layout persistence with safe fallback`，在重新整理時還原狀態並在會話失效時安全退回單欄；透過重新整理瀏覽器驗證持久化還原與容錯處理，達成 `Split view persistence` 需求。

## 3. 整合驗證與品質把關 (Verification & Audit)

- [x] 3.1 執行全專案自動化測試 `go test ./...`，驗證後端無回歸問題；確認全部測試 100% 通過。
- [x] 3.2 執行 ponytail-review 審查前端分割實作的複雜度與過度工程，刪除投機性抽象（避免自造複雜狀態機或多餘包裝層），維持極簡架構；完成 ponytail-review 檢查並確認符合精簡標準。
- [x] 3.3 執行 Spectra 分析與驗證指令 `spectra.exe analyze split-chat-view` 與 `spectra.exe validate split-chat-view`，確認工件一致性與規範無缺漏。
