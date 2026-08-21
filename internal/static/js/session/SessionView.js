// ── Session 列表畫面 ──────────────────────────────────────────────────────────
function SessionView({ onEnter, onSessionsLoaded, onSortedSessionsChange, activeSessionId = null, onCreateNew }) {
  const [sessions, setSessions]   = useState([]);
  const [sessionSearch, setSessionSearch] = useState('');
  const [sessionSort, setSessionSort]     = useState('last_active_desc');
  const [groupByDir] = useState(true); // RemoteAgent：固定依工作目錄分組
  const [collapsedDirs, setCollapsedDirs] = useState(() => new Set());
  const [renamingId, setRenamingId] = useState(null);
  const [renameVal, setRenameVal]   = useState('');
  const renameInputRef = useRef(null);

  const load = async () => {
    try {
      const res = await apiFetch('/sessions');
      if (!res.ok) {
        console.warn('[sessions] load failed', res.status);
        return;
      }
      const data = await res.json();
      if (Array.isArray(data)) {
        const sorted = sortSessionsByWorkDirThenName(data);
        setSessions(sorted);
        onSessionsLoaded?.(sorted);
      } else {
        console.warn('[sessions] unexpected payload', data);
      }
    } catch (err) {
      console.warn('[sessions] load error', err);
    }
  };

  useEffect(() => { load(); }, []);

  useEffect(() => {
    const id = setInterval(() => { load(); }, 5000);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    if (renamingId) renameInputRef.current?.focus();
  }, [renamingId]);

  const del = async (id, e) => {
    e.stopPropagation();
    if (!confirm('確定刪除此 Session 及所有對話紀錄？')) return;
    await apiFetch(`/sessions/${id}`, { method: 'DELETE' });
    load();
  };

  const startRename = (s, e) => {
    e.stopPropagation();
    setRenamingId(s.id);
    setRenameVal(s.name || '');
  };

  const commitRename = async (id) => {
    const trimmed = renameVal.trim();
    if (trimmed) {
      await apiFetch(`/sessions/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: trimmed }),
      });
      await load();
    }
    setRenamingId(null);
    setRenameVal('');
  };

  const handleRenameKey = (e, id) => {
    if (e.key === 'Enter') { e.preventDefault(); commitRename(id); }
    if (e.key === 'Escape') { setRenamingId(null); setRenameVal(''); }
  };

  const sortOptions = [
    { value: 'last_active_desc', label: '活動 新→舊' },
    { value: 'last_active_asc', label: '活動 舊→新' },
    { value: 'name_asc', label: '名稱 A→Z' },
    { value: 'name_desc', label: '名稱 Z→A' },
    { value: 'agent_asc', label: 'Runner' },
  ];

  const displaySessions = useMemo(() => {
    const q = sessionSearch.trim().toLowerCase();
    let list = Array.isArray(sessions) ? sessions : [];
    if (q) {
      list = list.filter((s) => {
        const parts = [
          s.name,
          s.work_dir,
          s.git_branch,
          s.description,
          s.agent_type,
          s.agent_session_id,
          s.id,
          ...(Array.isArray(s.cli_extra_args) ? s.cli_extra_args : []),
        ]
          .filter(Boolean)
          .join(' ')
          .toLowerCase();
        return parts.includes(q);
      });
    }
    const out = [...list];
    const cmpName = (a, b) => (a.name || '').localeCompare(b.name || '', 'zh-Hant');
    const cmpAgent = (a, b) => (a.agent_type || '').localeCompare(b.agent_type || '');
    if (sessionSort === 'last_active_asc' || sessionSort === 'last_active_desc') {
      const groupMaxMs = new Map();
      for (const s of out) {
        const k = String(s.work_dir != null ? String(s.work_dir).trim() : '').toLowerCase();
        const t = sessionLastActiveMs(s);
        if (!groupMaxMs.has(k) || t > groupMaxMs.get(k)) groupMaxMs.set(k, t);
      }
      const sign = sessionSort === 'last_active_desc' ? 1 : -1;
      out.sort((a, b) => {
        const ka = String(a.work_dir != null ? String(a.work_dir).trim() : '').toLowerCase();
        const kb = String(b.work_dir != null ? String(b.work_dir).trim() : '').toLowerCase();
        const gd = sign * ((groupMaxMs.get(kb) ?? 0) - (groupMaxMs.get(ka) ?? 0));
        if (gd !== 0) return gd;
        return sign * (sessionLastActiveMs(b) - sessionLastActiveMs(a));
      });
    } else {
      out.sort((a, b) => {
        const wd = cmpSessionWorkDir(a, b);
        if (wd !== 0) return wd;
        switch (sessionSort) {
          case 'name_asc':  return cmpName(a, b);
          case 'name_desc': return cmpName(b, a);
          case 'agent_asc': return cmpAgent(a, b) || cmpName(a, b);
          default: return 0;
        }
      });
    }
    return out;
  }, [sessions, sessionSearch, sessionSort]);

  const sortedSessionsForForward = useMemo(() => {
    const out = [...(Array.isArray(sessions) ? sessions : [])];
    const cmpName = (a, b) => (a.name || '').localeCompare(b.name || '', 'zh-Hant');
    const cmpAgent = (a, b) => (a.agent_type || '').localeCompare(b.agent_type || '');
    if (sessionSort === 'last_active_asc' || sessionSort === 'last_active_desc') {
      const groupMaxMs = new Map();
      for (const s of out) {
        const k = String(s.work_dir != null ? String(s.work_dir).trim() : '').toLowerCase();
        const t = sessionLastActiveMs(s);
        if (!groupMaxMs.has(k) || t > groupMaxMs.get(k)) groupMaxMs.set(k, t);
      }
      const sign = sessionSort === 'last_active_desc' ? 1 : -1;
      out.sort((a, b) => {
        const ka = String(a.work_dir != null ? String(a.work_dir).trim() : '').toLowerCase();
        const kb = String(b.work_dir != null ? String(b.work_dir).trim() : '').toLowerCase();
        const gd = sign * ((groupMaxMs.get(kb) ?? 0) - (groupMaxMs.get(ka) ?? 0));
        if (gd !== 0) return gd;
        return sign * (sessionLastActiveMs(b) - sessionLastActiveMs(a));
      });
    } else {
      out.sort((a, b) => {
        const wd = cmpSessionWorkDir(a, b);
        if (wd !== 0) return wd;
        switch (sessionSort) {
          case 'name_asc':  return cmpName(a, b);
          case 'name_desc': return cmpName(b, a);
          case 'agent_asc': return cmpAgent(a, b) || cmpName(a, b);
          default: return 0;
        }
      });
    }
    return out;
  }, [sessions, sessionSort]);

  useEffect(() => { onSortedSessionsChange?.(sortedSessionsForForward); }, [sortedSessionsForForward]);

  const groupedSessions = useMemo(() => {
    if (!groupByDir) return null;
    const map = new Map();
    const unset = '（未設定工作目錄）';
    for (const s of displaySessions) {
      const raw = s.work_dir != null ? String(s.work_dir).trim() : '';
      const key = raw || unset;
      if (!map.has(key)) map.set(key, []);
      map.get(key).push(s);
    }
    return [...map.entries()].sort((a, b) => {
      const ka = String(a[0]);
      const kb = String(b[0]);
      if (ka === unset && kb !== unset) return 1;
      if (ka !== unset && kb === unset) return -1;
      if (sessionSort === 'last_active_asc' || sessionSort === 'last_active_desc') {
        const maxA = Math.max(0, ...a[1].map(sessionLastActiveMs));
        const maxB = Math.max(0, ...b[1].map(sessionLastActiveMs));
        if (maxA !== maxB) return sessionSort === 'last_active_desc' ? maxB - maxA : maxA - maxB;
      }
      return ka.localeCompare(kb, undefined, { numeric: true, sensitivity: 'base' });
    });
  }, [groupByDir, displaySessions, sessionSort]);

  const sessionSearchActive = sessionSearch.trim().length > 0;

  const toggleDirCollapsed = (dirKey) => {
    setCollapsedDirs((prev) => {
      const next = new Set(prev);
      if (next.has(dirKey)) next.delete(dirKey);
      else next.add(dirKey);
      return next;
    });
  };

  const openNewSessionForDir = (dirKey) => {
    const unset = '（未設定工作目錄）';
    const dir = dirKey === unset ? '' : dirKey;
    onCreateNew?.({ workDir: dir, name: workDirBasename(dirKey) });
  };

  // 未讀判斷：last_active 比 last_read_at 新即為未讀。任一值缺失時視為已讀（避免誤判）。
  // 兩者皆為 SQLite datetime('now') 字串，解析方式須與 sessionLastActiveMs 一致（同樣忽略時區，本地時間比較）。
  const isUnread = (s) => {
    const active = sessionLastActiveMs(s);
    if (!active) return false;
    const read = s.last_read_at ? Date.parse(String(s.last_read_at).trim().replace(' ', 'T')) : 0;
    return active > (Number.isNaN(read) ? 0 : read);
  };

  const hasUnread = useMemo(
    () => (Array.isArray(sessions) ? sessions : []).some((s) => activeSessionId !== s.id && isUnread(s)),
    [sessions, activeSessionId]
  );

  const handleReadAll = () => {
    // Optimistic：點了就消掉；背景打 API，不等回應、失敗不回滾，下次 5 秒 poll 覆蓋掉也沒差。
    const nowStr = formatSqliteNow();
    setSessions((prev) => prev.map((s) => ({ ...s, last_read_at: nowStr })));
    apiFetch('/sessions/read-all', { method: 'POST' }).catch((err) => {
      console.warn('[sessions] read-all error', err);
    });
  };

  // 進入 session 前先 optimistic 標已讀（點了就消掉）；真正的 mark_read 由
  // useChatSocket.js 的 WS onopen 背景送出，失敗或被下次 poll 覆蓋都沒差。
  const handleEnter = (s) => {
    if (isUnread(s)) {
      const nowStr = formatSqliteNow();
      setSessions((prev) => prev.map((row) => (row.id === s.id ? { ...row, last_read_at: nowStr } : row)));
    }
    onEnter(s);
  };

  const renderSessionRow = (s) => {
    const active = activeSessionId != null && s.id === activeSessionId;
    const extraArgN = Array.isArray(s.cli_extra_args) ? s.cli_extra_args.length : 0;
    const agentLabel = AGENT_LABEL[s.agent_type] || s.agent_type || 'claude';
    const unread = !active && isUnread(s);
    const rowTitle = [agentLabel, s.git_branch, extraArgN > 0 ? `+${extraArgN} CLI 引數` : '']
      .filter(Boolean)
      .join(' · ');
    return (
      <div
        key={s.id}
        onClick={() => renamingId !== s.id && handleEnter(s)}
        className={'ra-session-card group/card' + (active ? ' active' : '') + (unread ? ' unread' : '')}
      >
          {renamingId === s.id ? (
            <input
              ref={renameInputRef}
              value={renameVal}
              onChange={(e) => setRenameVal(e.target.value)}
              onKeyDown={(e) => handleRenameKey(e, s.id)}
              onBlur={() => commitRename(s.id)}
              onClick={(e) => e.stopPropagation()}
              className="w-full bg-[oklch(0.19_0.02_264)] border border-violet-500 rounded-lg px-2 py-1 text-sm text-[oklch(0.94_0.01_264)] focus:outline-none"
            />
          ) : (
            <div className="flex items-center gap-2 min-w-0" title={rowTitle}>
              <span className={`inline-flex shrink-0 items-center justify-center w-[22px] h-[22px] rounded-[6px] ${getAgentBadgeClass(s.agent_type)}`} title={agentLabel}>
                <AgentBadgeIcon agentType={s.agent_type} />
              </span>
              {unread ? (
                <span
                  className="shrink-0 w-[7px] h-[7px] rounded-full bg-violet-400"
                  aria-label="未讀"
                  title="未讀"
                />
              ) : null}
              <span
                className={
                  'flex-1 min-w-0 text-[13.5px] truncate leading-snug ' +
                  (unread ? 'font-semibold text-[oklch(0.94_0.01_264)]' : 'font-medium text-[oklch(0.75_0.01_264)]')
                }
              >
                {s.name || '未命名'}
              </span>
              <div className="flex items-center gap-0.5 shrink-0 sm:opacity-0 sm:group-hover/card:opacity-100 transition-opacity">
                <button
                  type="button"
                  onClick={(e) => startRename(s, e)}
                  aria-label="重新命名"
                  title="重新命名"
                  className="text-[oklch(0.5_0.01_264)] hover:text-violet-400 p-1 rounded transition-colors"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" className="w-3.5 h-3.5" aria-hidden="true">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                  </svg>
                </button>
                <button
                  type="button"
                  onClick={(e) => del(s.id, e)}
                  aria-label="刪除"
                  title="刪除"
                  className="text-[oklch(0.5_0.01_264)] hover:text-red-400 p-1 rounded transition-colors inline-flex items-center justify-center"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="w-3.5 h-3.5" aria-hidden="true">
                    <path d="M3 6h18" />
                    <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
                    <path d="M10 11v6" />
                    <path d="M14 11v6" />
                  </svg>
                </button>
              </div>
              <span
                className={sessionStatusDotClass(s) + ' sm:group-hover/card:opacity-0 transition-opacity'}
                title={sessionStatusLabel(s)}
              />
            </div>
        )}
      </div>
    );
  };

  return (
    <div className="flex flex-col h-app min-w-0 overflow-hidden">
      {/* Header — 設計 1a / 2a */}
      <div className="flex items-center justify-between px-5 pt-[18px] pb-3 shrink-0"
           style={{ paddingTop: 'calc(18px + env(safe-area-inset-top) + var(--tg-content-safe-top))' }}>
        <div className="flex items-center gap-2 min-w-0">
          <div className="w-6 h-6 rounded-[6px] bg-gradient-to-br from-[oklch(0.62_0.19_275)] to-[oklch(0.6_0.17_300)] flex items-center justify-center shrink-0" aria-hidden>
            <div className="w-[9px] h-[9px] rounded-[2px] bg-white" />
          </div>
          <span className="ra-display text-base text-[oklch(0.94_0.01_264)] tracking-tight">Agent Console</span>
        </div>
        <button
          type="button"
          onClick={() => onCreateNew?.()}
          aria-label="建立新 Session"
          title="新 Session"
          className="flex h-[30px] w-[30px] shrink-0 items-center justify-center rounded-[9px] bg-[oklch(0.62_0.19_275)] text-white text-lg leading-none transition hover:brightness-110 active:scale-95 focus:outline-none"
        >
          +
        </button>
      </div>

      {/* 搜尋／排序 — 設計列 */}
      {sessions.length > 0 && (
        <div className="px-5 pb-3 shrink-0 flex gap-2 min-w-0 items-center">
          <div className="flex-1 min-w-0 flex items-center gap-1.5 bg-[oklch(0.19_0.02_264)] border border-[oklch(0.28_0.02_264)] rounded-[9px] px-2.5 py-2">
            <span className="text-[oklch(0.5_0.01_264)] text-sm shrink-0" aria-hidden>⌕</span>
            <input
              type="search"
              value={sessionSearch}
              onChange={(e) => setSessionSearch(e.target.value)}
              placeholder="搜尋 session…"
              autoComplete="off"
              className="flex-1 min-w-0 bg-transparent border-0 p-0 text-[13px] text-[oklch(0.9_0.01_264)] placeholder-[oklch(0.5_0.01_264)] focus:outline-none"
            />
          </div>
          {/* 排序下拉選單暫時隱藏（無人使用），固定用預設的 last_active_desc；邏輯保留供日後復原。 */}
          {false && (
            <select
              id="session-sort"
              value={sessionSort}
              onChange={(e) => setSessionSort(e.target.value)}
              title="排序"
              aria-label="排序"
              className="shrink-0 bg-[oklch(0.19_0.02_264)] border border-[oklch(0.28_0.02_264)] rounded-[9px] px-2.5 py-2 text-xs text-[oklch(0.65_0.01_264)] focus:outline-none cursor-pointer max-w-[6.5rem]"
            >
              {sortOptions.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label.replace('活動 ', '')}</option>
              ))}
            </select>
          )}
          {hasUnread && (
            <button
              type="button"
              onClick={handleReadAll}
              aria-label="全部標記已讀"
              title="全部標記已讀"
              className="shrink-0 rounded-[9px] border border-[oklch(0.28_0.02_264)] bg-[oklch(0.19_0.02_264)] px-2.5 py-2 text-xs text-[oklch(0.65_0.01_264)] hover:text-violet-300 hover:border-violet-500/50 transition-colors focus:outline-none"
            >
              全讀
            </button>
          )}
        </div>
      )}

      {/* Session 列表 */}
      <div className="flex-1 overflow-y-auto app-scroll px-3.5 pb-5">
        {sessions.length === 0 ? (
          <div className="text-center text-[oklch(0.5_0.01_264)] mt-24 text-sm px-4">尚無 Session，點右上角 + 建立</div>
        ) : displaySessions.length === 0 ? (
          <div className="text-center text-gray-600 mt-24 text-sm px-4">沒有符合搜尋條件的 Session</div>
        ) : groupByDir && groupedSessions ? (
          groupedSessions.map(([dirKey, dirSessions]) => {
            const expanded = sessionSearchActive || !collapsedDirs.has(dirKey);
            const shortLabel = workDirGroupShortLabel(dirKey);
            // 同一 work_dir 即同一 git working tree，分支必然相同，取第一筆即可
            const dirBranch = (dirSessions.find((s) => s.git_branch) || {}).git_branch;
            return (
              <div key={dirKey} className="space-y-0">
                <div className="group/dir flex w-full min-w-0 items-center gap-1 px-2 py-1.5 mt-2 transition-colors hover:bg-violet-500/[0.06]">
                  <button
                    type="button"
                    onClick={() => toggleDirCollapsed(dirKey)}
                    className="flex min-w-0 flex-1 items-center gap-1.5 rounded-md px-1 py-1 text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-violet-500/50 focus-visible:ring-offset-1 focus-visible:ring-offset-[oklch(0.13_0.02_264)]"
                  >
                    <span className="shrink-0 text-[oklch(0.5_0.01_264)] w-3 text-center text-[10px]" aria-hidden>
                      {expanded ? '▾' : '▸'}
                    </span>
                    <span className="shrink-0 text-[13px] leading-none" aria-hidden>📁</span>
                    <span
                      className="min-w-0 flex-1 truncate text-[12.5px] text-[oklch(0.72_0.01_264)] font-bold"
                      title={dirKey + (expanded ? '' : `（${dirSessions.length}）`)}
                    >
                      {shortLabel}
                    </span>
                  </button>
                  {dirBranch ? (
                    <span
                      className="ra-mono shrink-0 max-w-[8rem] truncate text-[11px] text-[oklch(0.5_0.01_264)]"
                      title={dirBranch}
                    >
                      {dirBranch}
                    </span>
                  ) : null}
                  <button
                    type="button"
                    onClick={() => openNewSessionForDir(dirKey)}
                    aria-label="在此目錄建立新 Session"
                    title="在此目錄建立新 Session"
                    className="shrink-0 rounded-md px-1.5 py-1 text-[13px] leading-none font-semibold text-violet-400 hover:bg-violet-500/10 hover:text-violet-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-violet-500/50 sm:opacity-0 sm:group-hover/dir:opacity-100 transition-opacity"
                  >
                    ＋
                  </button>
                </div>
                {expanded && (
                  <div className="ml-[18px] border-l border-[oklch(0.26_0.02_264)] pl-2.5">
                    {dirSessions.map((s) => renderSessionRow(s))}
                  </div>
                )}
              </div>
            );
          })
        ) : (
          displaySessions.map((s) => renderSessionRow(s))
        )}
      </div>
    </div>
  );
}

