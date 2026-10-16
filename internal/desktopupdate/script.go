package desktopupdate

import (
	"encoding/json"
)

// BridgeScript 回傳要在主視窗首頁載入完成後注入的 JS。
// 只在「頂層視窗、origin 與路徑為首頁、WebView2 可用」時安裝 window.__desktopUpdater，
// 其他頁面（分享頁、一般瀏覽器）不會得到任何 bridge。
// 注入端仍不信任 JS：Go 端會再以 Authorize 驗證每則訊息。
func BridgeScript(homeOrigin string) string {
	origin, _ := json.Marshal(homeOrigin) // 字串序列化不會失敗
	return `(function () {
  var OWNER_ORIGIN = ` + string(origin) + `;
  var ACTIONS = { check: true, download: true, status: true, restart: true };
  if (window.top !== window) return;
  if (location.origin !== OWNER_ORIGIN || location.pathname !== "/") return;
  var wv = window.chrome && window.chrome.webview;
  if (!wv || typeof wv.postMessage !== "function") return;
  if (window.__desktopUpdater) return;
  Object.defineProperty(window, "__desktopUpdater", {
    value: Object.freeze({
      request: function (action) {
        if (typeof action !== "string" || !Object.prototype.hasOwnProperty.call(ACTIONS, action)) return false;
        wv.postMessage("desktop-update:" + action);
        return true;
      }
    })
  });
  window.dispatchEvent(new CustomEvent("desktop-update-bridge"));
})();`
}

// StatusScript 回傳要在主視窗首頁執行的 JS：把快照以 desktop-update-state 事件送給前端。
// 沒有 bridge 的頁面不會派發事件。
func StatusScript(st Status) string {
	raw, _ := json.Marshal(st) // Status 只含可序列化欄位，不會失敗
	return `(function () {
  if (!window.__desktopUpdater) return;
  window.dispatchEvent(new CustomEvent("desktop-update-state", { detail: ` + string(raw) + ` }));
})();`
}

// OpenScript 回傳要求前端開啟設定面板並定位到更新區塊的 JS（托盤「檢查更新」使用）。
func OpenScript() string {
	return `(function () {
  if (!window.__desktopUpdater) return;
  window.dispatchEvent(new CustomEvent("desktop-update-open"));
})();`
}
