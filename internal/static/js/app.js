// ── 密碼輸入畫面（非 TMA 環境）────────────────────────────────────────────────
function PasswordView({ onSuccess }) {
  const [pwd, setPwd]     = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    if (!pwd.trim()) return;
    setLoading(true);
    setError('');
    const res = await fetch(resolveApiUrl('/auth/login'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: pwd }),
    });
    setLoading(false);
    if (res.ok) {
      const data = await res.json().catch(() => ({}));
      if (data.token) {
        try {
          localStorage.setItem(WEB_SESSION_STORAGE_KEY, data.token);
        } catch (_) {}
      }
      onSuccess();
    } else {
      const data = await res.json().catch(() => ({}));
      setError(data.error || '密碼錯誤，請重試');
    }
  };

  const handleKey = (e) => {
    if (e.key === 'Enter') submit();
  };

  return (
    <div className="flex items-center justify-center h-app">
      <div className="w-72 space-y-4">
        <div className="text-center">
          <div className="flex flex-col items-center gap-2 mb-1">
            <div className="w-10 h-10 rounded-[10px] bg-gradient-to-br from-violet-500 to-fuchsia-600 flex items-center justify-center" aria-hidden>
              <div className="w-3.5 h-3.5 rounded-[3px] bg-white" />
            </div>
            <div className="ra-display text-2xl" style={{color:'oklch(0.94 0.01 264)'}}>Agent Console</div>
          </div>
          <div className="text-gray-500 text-sm">Claude Code · Remote Agent</div>
          <div className="text-gray-600 text-xs">請輸入存取密碼</div>
        </div>
        <input
          type="password"
          value={pwd}
          onChange={e => setPwd(e.target.value)}
          onKeyDown={handleKey}
          placeholder="密碼"
          autoFocus
          className="w-full bg-gray-800 border border-gray-700 rounded-lg px-4 py-2.5 text-sm text-gray-200 placeholder-gray-500 focus:outline-none focus:border-violet-600"
        />
        {error && <div className="text-red-400 text-xs text-center">{error}</div>}
        <button onClick={submit} disabled={loading || !pwd.trim()}
          className="w-full py-2.5 bg-violet-600 hover:bg-violet-500 disabled:opacity-40 text-white rounded-lg text-sm font-medium">
          {loading ? '驗證中…' : '進入'}
        </button>
      </div>
    </div>
  );
}

// ── Debug Banner（開發用，上線前移除）────────────────────────────────────────
const DEBUG_BANNER = false;
function DebugBanner() {
  if (!DEBUG_BANNER) return null;
  return (
    <div style={{position:'fixed',top:0,left:0,right:0,background:'#1a1a00',color:'#ffff00',fontSize:'10px',padding:'4px 8px',zIndex:9999,wordBreak:'break-all'}}>
      isTMA={String(isTMA)} | TG={String(isTelegram)} | initData={initData ? initData.slice(0,40)+'…' : '(empty)'}
    </div>
  );
}

// ── Root ──────────────────────────────────────────────────────────────────────
function App() {
  const [session, setSession] = useState(null);
  const [creating, setCreating] = useState(false);
  const [composerPrefill, setComposerPrefill] = useState(null);
  const [authed, setAuthed]   = useState(isTelegram);
  const [settingsOpen, setSettingsOpen] = useState(false);

  // 外觀設定（Markdown 顏色 + 自訂 CSS）：mount 時套用 localStorage 已存的值；
  // 沒存過時 applyAppearance 內部會 fallback 成 index.html :root 的預設值。
  useEffect(() => {
    applyAppearance(readStoredAppearance());
  }, []);

  const openComposer = useCallback((prefill) => {
    setComposerPrefill(prefill || {});
    setCreating(true);
  }, []);
  const closeComposer = useCallback(() => setCreating(false), []);
  const handleCreated = useCallback((s) => {
    setCreating(false);
    setSession(s);
  }, []);
  const selectSession = useCallback((s) => {
    setCreating(false);
    setSession(s);
  }, []);
  const [isWideScreen, setIsWideScreen] = useState(() => window.matchMedia('(min-width: 1024px)').matches);
  // 非 TMA 環境：mount 時試探 cookie 是否仍有效
  const [checking, setChecking] = useState(!isTelegram);

  const [sidebarWidthPx, setSidebarWidthPx] = useState(() =>
    typeof window !== 'undefined' ? clampSidebarWidthPx(readStoredSidebarWidthPx()) : SIDEBAR_WIDTH_DEFAULT
  );
  const lastSidebarWidthRef = useRef(sidebarWidthPx);

  useEffect(() => {
    lastSidebarWidthRef.current = sidebarWidthPx;
  }, [sidebarWidthPx]);

  const [sidebarCollapsed, setSidebarCollapsed] = useState(readStoredSidebarCollapsed);

  const toggleSidebarCollapsed = useCallback(() => {
    setSidebarCollapsed((prev) => {
      const next = !prev;
      try { localStorage.setItem(SIDEBAR_COLLAPSED_KEY, next ? '1' : '0'); } catch (_) {}
      return next;
    });
  }, []);

  useEffect(() => {
    const onResize = () => {
      setSidebarWidthPx((w) => clampSidebarWidthPx(w));
    };
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  const handleSidebarResizeStart = useCallback((e) => {
    if (e.button !== undefined && e.button !== 0) return;
    e.preventDefault();
    const getX = (ev) => {
      if (ev.touches && ev.touches.length) return ev.touches[0].clientX;
      return ev.clientX;
    };
    const startX = getX(e);
    const startW = lastSidebarWidthRef.current;

    const onMove = (ev) => {
      ev.preventDefault();
      const x = getX(ev);
      const next = clampSidebarWidthPx(startW + (x - startX));
      setSidebarWidthPx(next);
      lastSidebarWidthRef.current = next;
    };

    const onUp = () => {
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
      document.removeEventListener('touchmove', onMove);
      document.removeEventListener('touchend', onUp);
      document.removeEventListener('touchcancel', onUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
      try {
        localStorage.setItem(SIDEBAR_WIDTH_STORAGE_KEY, String(lastSidebarWidthRef.current));
      } catch (_) {}
    };

    if (e.type === 'touchstart') {
      document.addEventListener('touchmove', onMove, { passive: false });
      document.addEventListener('touchend', onUp);
      document.addEventListener('touchcancel', onUp);
    } else {
      document.addEventListener('mousemove', onMove);
      document.addEventListener('mouseup', onUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    }
  }, []);

  const [sidebarSortedSessions, setSidebarSortedSessions] = useState([]);

  /** 側欄輪詢 /sessions 時同步目前聊天室 Session 的 git_branch、work_dir */
  const mergeSessionMetaFromList = useCallback((list) => {
    if (!Array.isArray(list)) return;
    setSession((prev) => {
      if (!prev) return null;
      const row = list.find((s) => s.id === prev.id);
      if (!row) return prev;
      if (row.git_branch === prev.git_branch && row.work_dir === prev.work_dir) return prev;
      return { ...prev, git_branch: row.git_branch, work_dir: row.work_dir };
    });
  }, []);

  useEffect(() => {
    if (isTelegram) return;
    registerSessionExpiredHandler(() => setAuthed(false));
    return () => registerSessionExpiredHandler(null);
  }, []);

  useEffect(() => {
    if (isTelegram) return;
    apiFetch('/sessions').then((res) => {
      if (res.ok) setAuthed(true);
    }).finally(() => setChecking(false));
  }, []);

  useEffect(() => {
    const media = window.matchMedia('(min-width: 1024px)');
    const handleChange = (e) => setIsWideScreen(e.matches);
    setIsWideScreen(media.matches);
    if (media.addEventListener) {
      media.addEventListener('change', handleChange);
    } else {
      media.addListener(handleChange);
    }
    return () => {
      if (media.removeEventListener) {
        media.removeEventListener('change', handleChange);
      } else {
        media.removeListener(handleChange);
      }
    };
  }, []);

  if (checking) {
    return <div className="flex items-center justify-center h-app"><div className="text-gray-500 text-sm">載入中…</div></div>;
  }

  const renderAuthedLayout = () => {
    if (!isWideScreen) {
      if (creating) {
        return <NewSessionComposer sessions={sidebarSortedSessions} prefill={composerPrefill} onCreated={handleCreated} onCancel={closeComposer} />;
      }
      if (session) {
        return <ChatView session={session} onBack={() => selectSession(null)} usePermModeDropdown onJumpToSession={selectSession} allSessions={sidebarSortedSessions} />;
      }
      return <SessionView onEnter={selectSession} onSessionsLoaded={mergeSessionMetaFromList} onSortedSessionsChange={setSidebarSortedSessions} activeSessionId={session?.id} onCreateNew={openComposer} onOpenSettings={() => setSettingsOpen(true)} />;
    }

    return (
      <div className="h-app flex bg-[oklch(0.15_0.02_264)] min-w-0 relative">
        {!sidebarCollapsed && (
          <aside
            style={{ width: sidebarWidthPx, flexShrink: 0 }}
            className="min-w-0 bg-[oklch(0.13_0.02_264)] border-r border-[oklch(0.26_0.02_264)] flex flex-col"
          >
            <SessionView onEnter={selectSession} onSessionsLoaded={mergeSessionMetaFromList} onSortedSessionsChange={setSidebarSortedSessions} activeSessionId={session?.id} onCreateNew={openComposer} onOpenSettings={() => setSettingsOpen(true)} />
          </aside>
        )}
        {!sidebarCollapsed && (
          <div
            role="separator"
            aria-orientation="vertical"
            aria-valuenow={sidebarWidthPx}
            aria-valuemin={SIDEBAR_WIDTH_MIN}
            aria-valuemax={sidebarWidthMaxPx()}
            title="拖曳調整側欄寬度"
            className="group relative w-1.5 shrink-0 cursor-col-resize select-none touch-none bg-transparent hover:bg-[oklch(0.62_0.19_275_/15%)]"
            onMouseDown={handleSidebarResizeStart}
            onTouchStart={handleSidebarResizeStart}
          >
            <span
              className="pointer-events-none absolute left-1/2 top-0 bottom-0 w-px -translate-x-1/2 bg-gray-800 group-hover:bg-violet-500/50"
              aria-hidden
            />
          </div>
        )}
        <button
          type="button"
          onClick={toggleSidebarCollapsed}
          title={sidebarCollapsed ? '展開會話列表' : '收合會話列表'}
          aria-label={sidebarCollapsed ? '展開會話列表' : '收合會話列表'}
          style={{ left: sidebarCollapsed ? 0 : sidebarWidthPx }}
          className="absolute top-1/2 -translate-y-1/2 z-20 flex items-center justify-center w-5 h-10 rounded-r-md bg-[oklch(0.19_0.02_264)] border border-l-0 border-[oklch(0.28_0.02_264)] text-gray-500 hover:text-gray-300 hover:bg-[oklch(0.24_0.02_264)] transition-colors"
        >
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" className={'w-3.5 h-3.5 transition-transform duration-150 ' + (sidebarCollapsed ? 'rotate-180' : '')} aria-hidden>
            <path fillRule="evenodd" d="M12.79 5.23a.75.75 0 01.02 1.06L8.832 10l3.938 3.71a.75.75 0 11-1.04 1.08l-4.5-4.25a.75.75 0 010-1.08l4.5-4.25a.75.75 0 011.06.02z" clipRule="evenodd" />
          </svg>
        </button>
        <main className="flex-1 min-w-0 min-h-0">
          {creating ? (
            <NewSessionComposer sessions={sidebarSortedSessions} prefill={composerPrefill} onCreated={handleCreated} onCancel={closeComposer} />
          ) : session ? (
            <ChatView session={session} onBack={() => selectSession(null)} showBack={false} fullHeight={false} onJumpToSession={selectSession} allSessions={sidebarSortedSessions} />
          ) : (
            <div className="h-full flex flex-col items-center justify-center gap-3 bg-[oklch(0.15_0.02_264)]">
              <div className="w-10 h-10 rounded-[10px] bg-gradient-to-br from-[oklch(0.62_0.19_275)] to-[oklch(0.6_0.17_300)] flex items-center justify-center opacity-80" aria-hidden>
                <div className="w-3.5 h-3.5 rounded-[3px] bg-white" />
              </div>
              <div className="text-[oklch(0.55_0.01_264)] text-sm">從左側選擇 Session，或按 + 建立新對話</div>
            </div>
          )}
        </main>
      </div>
    );
  };

  return (
    <>
      <DebugBanner />
      {!authed
        ? <PasswordView onSuccess={() => setAuthed(true)} />
        : renderAuthedLayout()
      }
      <SettingsModal open={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<App />);
