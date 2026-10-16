// ── 桌面自我更新：狀態訂閱與 bridge 呼叫 ─────────────────────────────────────
// 只有桌面端（WebView2 注入 window.__desktopUpdater）才會有 bridge；
// 瀏覽器、Telegram、訪客頁面 bridge 為 false，UI 不應顯示任何更新控制。
// 依賴 core.js 的 React hook 別名（useState、useEffect、useRef、useCallback）。

/** 目前頁面是否已有桌面更新 bridge。 */
function desktopUpdateBridgeReady() {
  const b = typeof window !== 'undefined' ? window.__desktopUpdater : null;
  return !!(b && typeof b.request === 'function');
}

/**
 * 訂閱 Go 端推送的狀態快照，並提供 request(action)。
 * - desktop-update-state：快照（detail 為 Status JSON）。
 * - desktop-update-bridge：bridge 剛注入（導航完成後），此時要求一次 status。
 * - desktop-update-open：托盤「檢查更新」要求開啟設定的更新區（onOpen 由 App 提供）。
 * laterVersion 只存在前端記憶體中，不寫回 Go 端。
 */
function useDesktopUpdate({ onOpen } = {}) {
  const [status, setStatus] = useState(null);
  const [bridge, setBridge] = useState(desktopUpdateBridgeReady);
  const [laterVersion, setLaterVersion] = useState('');
  const [openNonce, setOpenNonce] = useState(0);
  const onOpenRef = useRef(onOpen);
  onOpenRef.current = onOpen;

  useEffect(() => {
    const onState = (e) => {
      if (e && e.detail) setStatus(e.detail);
    };
    const onBridge = () => {
      setBridge(true);
      window.__desktopUpdater.request('status');
    };
    const onOpenEvent = () => {
      setOpenNonce((n) => n + 1);
      if (onOpenRef.current) onOpenRef.current();
    };
    window.addEventListener('desktop-update-state', onState);
    window.addEventListener('desktop-update-bridge', onBridge);
    window.addEventListener('desktop-update-open', onOpenEvent);
    if (desktopUpdateBridgeReady()) onBridge();
    return () => {
      window.removeEventListener('desktop-update-state', onState);
      window.removeEventListener('desktop-update-bridge', onBridge);
      window.removeEventListener('desktop-update-open', onOpenEvent);
    };
  }, []);

  // 只轉送白名單內的 action；bridge 不存在時回傳 false，不做任何事。
  const request = useCallback((action) => {
    if (!desktopUpdateBridgeReady()) return false;
    return window.__desktopUpdater.request(action);
  }, []);

  const later = useCallback((version) => setLaterVersion(version || ''), []);
  const showOffer = useCallback(() => setLaterVersion(''), []);

  return { bridge, status, request, laterVersion, later, showOffer, openNonce };
}
