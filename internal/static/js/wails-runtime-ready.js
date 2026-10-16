/**
 * 桌面版 WebView2 的就緒訊號。
 *
 * 桌面視窗載入的是外部 URL（本機 server 首頁），不經 Wails assetserver，因此 Wails 自己的
 * runtime JS 不會被載入，也就永遠不會送出 'wails:runtime:ready'。Go 端的 ExecJS 在收到該訊號前
 * 只會把 JS 排進佇列（bridge 腳本與 desktop-update-state 狀態事件都依賴它），所以首頁必須自行送出一次。
 *
 * 只在「擁有者首頁（路徑 /）」且 WebView2 可用（window.chrome.webview.postMessage）時送出；
 * 一般瀏覽器、分享頁等其他情況都是 no-op。flush 的內容自帶來源檢查（BridgeScript 只在頂層、
 * 同 origin、路徑 / 時安裝），因此送出訊號本身不會擴大權限。
 */
function announceWailsRuntimeReady(win) {
  if (!win || win.__cmaRuntimeReadyAnnounced) return false;
  if (!win.location || win.location.pathname !== '/') return false;
  const wv = win.chrome && win.chrome.webview;
  if (!wv || typeof wv.postMessage !== 'function') return false;
  win.__cmaRuntimeReadyAnnounced = true;
  wv.postMessage('wails:runtime:ready');
  return true;
}

if (typeof window !== 'undefined') {
  announceWailsRuntimeReady(window);
}
