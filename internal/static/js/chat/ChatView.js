function ChatView({ session, onBack, showBack = true, fullHeight = true, usePermModeDropdown = false, onJumpToSession, allSessions }) {
  const jumpToSession = typeof onJumpToSession === 'function' ? onJumpToSession : () => {};
  const agentType = session.agent_type || 'claude';
  // Claude / Cursor / Antigravity 皆支援 mode 切換（Codex 暫無對應概念）
  const showPermModeSelect = agentType !== 'codex' && agentType !== 'kiro' && agentType !== 'kiroacp';
  const showEffortSelect = agentType !== 'cursor';
  const [messages, setMessages]   = useState([]);
  const [input, setInput]         = useState('');
  const [state, setState]         = useState('IDLE');
  const [permTools, setPermTools] = useState([]);
  const [mode, setMode]           = useState(() => normalizePermMode(agentType, session.permission_mode || 'default'));
  const [modelSel, setModelSel]   = useState(() => session.model || '');
  const [effortSel, setEffortSel] = useState(() => session.effort || '');
  const [histLoaded, setHistLoaded] = useState(false);
  const [inputMode, setInputMode] = useState(() => readInputModeForSession(session.id));
  const [shellType, setShellType] = useState('bash');
  const [shellPendingCmd, setShellPendingCmd] = useState(null);
  const [shellRequest, setShellRequest] = useState(null);
  const [quota, setQuota] = useState(null);
  const [quotaRefreshing, setQuotaRefreshing] = useState(false);
  const [sessionModel, setSessionModel] = useState(null);
  const [forwardModal, setForwardModal] = useState(null);
  const [forwardHints, setForwardHints] = useState({});
  const [slashMenuItems, setSlashMenuItems] = useState([]);
  const [slashActiveIdx, setSlashActiveIdx] = useState(0);
  const shellBuf = useRef([]);
  const shellRenderPending = useRef(false);
  const wsRef     = useRef(null);
  const bottomRef = useRef(null);
  const chatScrollRef = useRef(null);
  const chatNearBottomRef = useRef(true);
  const chatInputRef = useRef(null);
  const slashInputWrapRef = useRef(null);
  const streamBuf = useRef('');
  const thinkingMode = useRef(false);
  const [activityHint, setActivityHint] = useState('');
  const connGenRef = useRef(0);
  const activeSessionRef = useRef(session.id);
  const reconnectTimerRef = useRef(null);
  /** 後端已套用的權限／輸入模式（送出時若與畫面選擇不同才送 set_mode／set_input_mode） */
  const serverPermAppliedRef = useRef(null);
  const serverInputAppliedRef = useRef(null);
  const serverModelAppliedRef = useRef(null);
  const serverEffortAppliedRef = useRef(null);
  const { collapsed: headerCollapsed, toggle: toggleHeader, setCollapsed: setHeaderCollapsed } = useChatHeaderCollapsed();

  const syncChatNearBottom = useCallback(() => {
    const el = chatScrollRef.current;
    if (!el) return;
    const gap = el.scrollHeight - el.scrollTop - el.clientHeight;
    chatNearBottomRef.current = gap < 80;
  }, []);

  /** 進入會話或執行結束後將游標放回輸入框（雙 rAF 以配合 React commit／行動裝置鍵盤） */
  const focusChatInput = useCallback(() => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        const el = chatInputRef.current;
        if (!el || el.disabled) return;
        try {
          el.focus({ preventScroll: true });
        } catch (_) {
          el.focus();
        }
      });
    });
  }, []);

  // 捲到底
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    let id2;
    const id1 = requestAnimationFrame(() => {
      id2 = requestAnimationFrame(() => {
        syncChatNearBottom();
      });
    });
    return () => {
      cancelAnimationFrame(id1);
      if (id2 != null) cancelAnimationFrame(id2);
    };
  }, [messages, syncChatNearBottom]);

  // 待授權／Shell 確認區塊出現時捲到底（此狀態常不經 messages 更新而單獨出現）
  useEffect(() => {
    const showPermPanel =
      (state === 'AWAITING_CONFIRM' && permTools.length > 0) ||
      (state === 'SHELL_AWAITING_APPROVAL' && shellPendingCmd) ||
      (state === 'AWAITING_SHELL_CONFIRM' && shellRequest);
    if (!showPermPanel) return;
    let id2;
    const id1 = requestAnimationFrame(() => {
      id2 = requestAnimationFrame(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
      });
    });
    return () => {
      cancelAnimationFrame(id1);
      if (id2 != null) cancelAnimationFrame(id2);
    };
  }, [state, permTools.length, shellPendingCmd, shellRequest]);

  // 離開頁面確認（任務進行中或等待授權時）
  useEffect(() => {
    const handler = (e) => {
      if (state === 'THINKING' || state === 'STREAMING' || state === 'AWAITING_CONFIRM' || state === 'SHELL_RUNNING' || state === 'SHELL_AWAITING_APPROVAL' || state === 'AWAITING_SHELL_CONFIRM' || state === 'SHELL_EXEC') {
        e.preventDefault();
        e.returnValue = '';
      }
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [state]);

  // session 切換時重置顯示與狀態，避免殘留舊會話上下文
  useEffect(() => {
    activeSessionRef.current = session.id;
    streamBuf.current = '';
    thinkingMode.current = false;
    shellBuf.current = [];
    shellRenderPending.current = false;
    setMessages([]);
    setHistLoaded(false);
    setState('IDLE');
    setPermTools([]);
    setShellRequest(null);
    let draft = '';
    try {
      draft = localStorage.getItem(draftInputStorageKey(session.id)) || '';
    } catch (_) {}
    setInput(draft);
    setSlashMenuItems([]);
    setSlashActiveIdx(0);
    const im =
      session.input_mode === 'shell' || session.input_mode === 'agent'
        ? session.input_mode
        : readInputModeForSession(session.id);
    const pm = normalizePermMode(agentType, session.permission_mode || 'default');
    serverInputAppliedRef.current = im;
    serverPermAppliedRef.current = pm;
    serverModelAppliedRef.current = session.model || '';
    serverEffortAppliedRef.current = session.effort || '';
    setInputMode(im);
    setMode(pm);
    setModelSel(session.model || '');
    setEffortSel(session.effort || '');
    setShellPendingCmd(null);
  }, [session.id, session.permission_mode, session.model, session.effort, agentType, session.input_mode]);

  const prevChatStateRef = useRef(null);
  // 歷史載入完成＝畫面就緒，將 focus 預設在輸入框
  useEffect(() => {
    if (!histLoaded) return;
    focusChatInput();
  }, [histLoaded, session.id, focusChatInput]);

  // 狀態切回可輸入時（例如 THINKING／STREAMING 結束）自動 focus
  useEffect(() => {
    const prev = prevChatStateRef.current;
    prevChatStateRef.current = state;
    if (prev === null) return;
    if (prev === state) return;
    focusChatInput();
  }, [state, focusChatInput]);

  // 載入歷史 + 建立 WS
  useEffect(() => {
    connGenRef.current += 1;
    const effectGen = connGenRef.current;
    let cancelled = false;

    const isCurrent = () =>
      !cancelled && effectGen === connGenRef.current && activeSessionRef.current === session.id;

    const clearReconnectTimer = () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    };

    const init = async () => {
      // 載入歷史紀錄
      let hist = [];
      try {
        const res = await apiFetch(`/sessions/${session.id}/messages`);
        if (res.ok) {
          const data = await res.json();
          if (Array.isArray(data)) hist = data;
          else console.warn('[messages] unexpected payload', data);
        } else {
          console.warn('[messages] load failed', res.status);
        }
      } catch (err) {
        console.warn('[messages] load error', err);
      }
      if (isCurrent()) {
        setMessages(hist.map(mapMessageRow));
        const lastClaude = [...hist].reverse().find((m) => m.role === 'claude' && m.status === 'pending');
        streamBuf.current = lastClaude ? (lastClaude.content || '') : '';
        setHistLoaded(true);
      }

      // 建立 WebSocket
      connectWS();
    };

    const connectWS = () => {
      if (!isCurrent()) return;
      clearReconnectTimer();
      const ws = new WebSocket(wsURL(session.id));
      ws._sessionId = session.id;
      ws._gen = effectGen;
      wsRef.current = ws;
      let everConnected = false;

      ws.onopen = () => { everConnected = true; };

      ws.onmessage = (evt) => {
        if (!isCurrent() || wsRef.current !== ws) return;
        const msg = JSON.parse(evt.data);

        if (msg.type === 'sync') {
          setState(msg.value);
          if (msg.input_mode) {
            setInputMode(msg.input_mode);
            serverInputAppliedRef.current = msg.input_mode;
          }
          if (msg.shell_type) setShellType(msg.shell_type);
          if (msg.shell_pending) setShellPendingCmd(msg.shell_pending);
          if (msg.messages) {
            const rows = Array.isArray(msg.messages) ? msg.messages : JSON.parse(msg.messages);
            const mapped = rows.map(mapMessageRow);
            const lastClaude = [...rows].reverse().find((m) => m.role === 'claude' && (m.status || 'done') === 'pending');
            streamBuf.current = lastClaude ? (lastClaude.content || '') : '';
            setMessages((prev) => {
              const byId = new Map();
              for (const m of prev) {
                if (m.id != null && m.createdAt) byId.set(m.id, m.createdAt);
              }
              return mapped.map((m) => (
                m.createdAt || m.id == null || !byId.has(m.id)
                  ? m
                  : { ...m, createdAt: byId.get(m.id) }
              ));
            });
          }
          if (msg.quota) setQuota(msg.quota);
          if (msg.model) setSessionModel(msg.model);
          return;
        }

        if (msg.type === 'quota_update' && msg.quota) {
          setQuota(msg.quota);
          setQuotaRefreshing(false);
          return;
        }

        if (msg.type === 'model_update' && msg.model) {
          setSessionModel(msg.model);
          return;
        }

        if (msg.type === 'status') {
          setState(msg.value);
          if (msg.value !== 'AWAITING_SHELL_CONFIRM') {
            setShellRequest(null);
          }
          if (msg.value === 'THINKING') {
            streamBuf.current = '';
            thinkingMode.current = false;
            setActivityHint('');
            setMessages((prev) => {
              const last = prev[prev.length - 1];
              if (last && last.role === 'claude' && last.streaming && last.status === 'pending') {
                return prev;
              }
              return [...prev, { role: 'claude', content: '', html: null, streaming: true, status: 'pending', createdAt: new Date().toISOString() }];
            });
          }
          if (msg.value === 'SHELL_RUNNING' || msg.value === 'SHELL_EXEC') {
            shellBuf.current = [];
            setShellPendingCmd(null);
            setMessages((prev) => {
              const last = prev[prev.length - 1];
              if (last && last.role === 'shell' && last.streaming) return prev;
              return [...prev, { role: 'shell', content: '', html: null, streaming: true, status: 'pending', createdAt: new Date().toISOString() }];
            });
          }
          if (msg.value === 'SHELL_IDLE') {
            setShellPendingCmd(null);
            shellBuf.current = [];
            setMessages((prev) => prev.map((m, i) =>
              i === prev.length - 1 && m.role === 'shell' && m.streaming
                ? { ...m, streaming: false, status: 'done' }
                : m
            ));
          }
          if (msg.value === 'IDLE' || msg.value === 'AWAITING_CONFIRM') {
            setMessages((prev) => {
              const updated = prev.map((m, i) =>
                i === prev.length - 1 ? { ...m, streaming: false, status: m.status === 'pending' ? 'done' : m.status } : m
              );
              const last = updated[updated.length - 1];
              // 移除因中斷而從未收到任何內容的暫存 claude 泡泡（無 DB id、無內容）
              if (last && last.role === 'claude' && last.id == null && !last.content && !last.html) {
                return updated.slice(0, -1);
              }
              return updated;
            });
          }
        }

        if (msg.type === 'reset') {
          setMessages([]);
          streamBuf.current = '';
        }

        if (msg.type === 'user_message') {
          setMessages(prev => [...prev, { role: 'user', content: msg.content, createdAt: msg.created_at || new Date().toISOString() }]);
        }

        if (msg.type === 'activity') {
          if (msg.content) setActivityHint(msg.content);
          return;
        }

        if (msg.type === 'thinking') {
          thinkingMode.current = true;
          const text = msg.content || '';
          setMessages((prev) => prev.map((m, i) =>
            i === prev.length - 1 ? { ...m, content: text, html: null, thinking: true, streaming: true, status: 'pending' } : m
          ));
        }

        if (msg.type === 'delta') {
          if (thinkingMode.current) {
            thinkingMode.current = false;
            streamBuf.current = '';
          }
          streamBuf.current += msg.content;
          const html = parseMarkdown(streamBuf.current);
          setMessages((prev) => prev.map((m, i) =>
            i === prev.length - 1 ? { ...m, content: streamBuf.current, html, thinking: false, streaming: true, status: 'pending' } : m
          ));
        }

        if (msg.type === 'permission_request') {
          setPermTools(msg.tools || []);
        }

        if (msg.type === 'message_result_text') {
          const rid = msg.id;
          const rt = msg.content != null ? String(msg.content) : '';
          setMessages((prev) => {
            let idx = prev.findIndex((m) => m.id != null && rid != null && m.id === rid);
            if (idx < 0) {
              for (let i = prev.length - 1; i >= 0; i--) {
                if (prev[i].role === 'claude') {
                  idx = i;
                  break;
                }
              }
            }
            if (idx < 0) return prev;
            const copy = [...prev];
            const prevContent = (copy[idx].content || '').trim();
            copy[idx] = {
              ...copy[idx],
              resultText: rt,
              id: copy[idx].id != null ? copy[idx].id : rid,
              ...(prevContent ? {} : {
                content: rt,
                html: rt ? parseMarkdown(rt) : null,
                thinking: false,
                streaming: false,
              }),
            };
            return copy;
          });
        }

        if (msg.type === 'input_mode_changed') {
          setInputMode(msg.value);
          serverInputAppliedRef.current = msg.value;
          if (msg.shell_type) setShellType(msg.shell_type);
        }

        if (msg.type === 'shell_approval_request') {
          setShellPendingCmd({ command: msg.content, work_dir: msg.work_dir, shell_type: msg.shell_type });
        }

        if (msg.type === 'shell_approval_cancelled') {
          setShellPendingCmd(null);
        }

        if (msg.type === 'shell_command_request') {
          setShellRequest({
            command: msg.command || '',
            line: msg.line || '',
            workDirKey: msg.work_dir_key || '',
          });
        }

        if (msg.type === 'shell_delta') {
          if (!msg.content) return;
          shellBuf.current.push({ stream: msg.stream || 'stdout', text: msg.content });
          if (!shellRenderPending.current) {
            shellRenderPending.current = true;
            requestAnimationFrame(() => {
              shellRenderPending.current = false;
              const rendered = shellBuf.current.map(s => s.stream === 'stderr' ? '[stderr] ' + s.text : s.text).join('');
              setMessages(prev => prev.map((m, i) =>
                i === prev.length - 1 && m.role === 'shell' && m.streaming
                  ? { ...m, content: rendered }
                  : m
              ));
            });
          }
        }

        if (msg.type === 'shell_done') {
          const code = msg.exit_code;
          setMessages(prev => prev.map((m, i) =>
            i === prev.length - 1 && m.role === 'shell' && m.streaming
              ? { ...m, streaming: false, status: 'done', exitCode: code, id: msg.id != null ? msg.id : m.id }
              : m
          ));
          shellBuf.current = [];
        }

        if (msg.type === 'shell_error') {
          shellBuf.current.push({ stream: 'stderr', text: '[error] ' + (msg.content || '') + '\n' });
          const rendered = shellBuf.current.map(s => s.stream === 'stderr' ? '[stderr] ' + s.text : s.text).join('');
          setMessages(prev => prev.map((m, i) =>
            i === prev.length - 1 && m.role === 'shell' && m.streaming
              ? { ...m, content: rendered }
              : m
          ));
        }

        if (msg.type === 'shell_result') {
          setMessages((prev) => {
            const id = msg.id;
            const content = msg.content != null ? String(msg.content) : '';
            let found = false;
            const next = prev.map((m) => {
              if (found) return m;
              if (id != null && m.id === id) {
                found = true;
                return { ...m, role: 'shell', content, html: null, streaming: false, status: 'done' };
              }
              return m;
            });
            if (found) return next;
            for (let i = next.length - 1; i >= 0; i--) {
              if (next[i].role === 'shell' && next[i].status === 'pending') {
                const copy = [...next];
                copy[i] = { ...copy[i], id: id != null ? id : copy[i].id, content, html: null, streaming: false, status: 'done' };
                return copy;
              }
            }
            return [...next, { role: 'shell', id: id != null ? id : null, content, html: null, streaming: false, status: 'done' }];
          });
        }
      };

      ws.onerror = (evt) => {
        console.error('[ws] 連線錯誤', evt);
      };

      ws.onclose = (evt) => {
        console.warn('[ws] 連線關閉', evt.code, evt.reason);
        if (!isCurrent() || wsRef.current !== ws) return;
        const scheduleReconnect = () => {
          reconnectTimerRef.current = setTimeout(() => {
            reconnectTimerRef.current = null;
            connectWS();
          }, 2000);
        };
        if (!everConnected && !isTelegram) {
          apiFetch('/sessions').then((res) => {
            if (!isCurrent() || wsRef.current !== ws) return;
            if (!res.ok) return;
            try {
              if (!localStorage.getItem(WEB_SESSION_STORAGE_KEY)) return;
            } catch (_) { return; }
            scheduleReconnect();
          }).catch(() => {});
          return;
        }
        if (!isTelegram) {
          try {
            if (!localStorage.getItem(WEB_SESSION_STORAGE_KEY)) return;
          } catch (_) { return; }
        }
        scheduleReconnect();
      };
    };

    init();

    return () => {
      cancelled = true;
      clearReconnectTimer();
      if (wsRef.current && wsRef.current._gen === effectGen) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [session.id]);

  const send = useCallback((obj) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== 1) return false;
    if (ws._sessionId !== activeSessionRef.current || ws._sessionId !== session.id) return false;
    ws.send(JSON.stringify(obj));
    return true;
  }, [session.id]);

  const handleQuotaRefresh = useCallback(() => {
    if (quotaRefreshing) return;
    if (agentType === 'antigravity') return;
    setQuotaRefreshing(true);
    if (!send({ type: 'refresh_quota' })) {
      setQuotaRefreshing(false);
      return;
    }
    setTimeout(() => setQuotaRefreshing(false), 10000);
  }, [quotaRefreshing, agentType, send]);

  const flushPendingModes = useCallback(() => {
    const wantPerm = normalizePermMode(agentType, mode);
    const wantInput = inputMode === 'shell' || inputMode === 'agent' ? inputMode : 'agent';
    if (showPermModeSelect && wantPerm !== serverPermAppliedRef.current) {
      if (!send({ type: 'set_mode', mode: wantPerm })) return false;
      serverPermAppliedRef.current = wantPerm;
    }
    if (wantInput !== serverInputAppliedRef.current) {
      if (!send({ type: 'set_input_mode', mode: wantInput })) return false;
      serverInputAppliedRef.current = wantInput;
    }
    if (modelSel !== serverModelAppliedRef.current) {
      if (!send({ type: 'set_model', model: modelSel })) return false;
      serverModelAppliedRef.current = modelSel;
    }
    if (showEffortSelect && effortSel !== serverEffortAppliedRef.current) {
      if (!send({ type: 'set_effort', effort: effortSel })) return false;
      serverEffortAppliedRef.current = effortSel;
    }
    return true;
  }, [send, agentType, mode, inputMode, showPermModeSelect, modelSel, effortSel, showEffortSelect]);

  const handleSend = (overrideText) => {
    const raw = overrideText !== undefined && overrideText !== null ? String(overrideText) : input;
    const trimmed = raw.trim();
    if (!trimmed) return;
    if (trimmed === '/reset' || trimmed === '/clear') {
      if (state !== 'IDLE' && state !== 'SHELL_IDLE') return;
      if (!flushPendingModes()) return;
      send({ type: 'reset_context' });
      clearDraftInputForSession(session.id);
      setInput('');
      setSlashMenuItems([]);
      return;
    }
    if (inputMode === 'shell') {
      if (state !== 'SHELL_IDLE' && state !== 'IDLE') return;
      if (!flushPendingModes()) return;
      send({ type: 'shell_exec', data: trimmed });
      clearDraftInputForSession(session.id);
      setInput('');
      setSlashMenuItems([]);
      return;
    }
    if (state !== 'IDLE' && state !== 'SHELL_IDLE') return;
    if (!flushPendingModes()) return;
    if (!send({ type: 'input', data: trimmed })) return;
    clearDraftInputForSession(session.id);
    setInput('');
    setSlashMenuItems([]);
  };

  const handleSlashSelect = (command) => {
    setInput(command);
    setSlashMenuItems([]);
    handleSend(command);
  };

  const handleKeyDown = (e) => {
    const slashMenuOpen = slashMenuItems.length > 0;
    if (slashMenuOpen) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSlashActiveIdx((i) => Math.min(i + 1, slashMenuItems.length - 1));
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSlashActiveIdx((i) => Math.max(i - 1, 0));
        return;
      }
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        const item = slashMenuItems[slashActiveIdx];
        if (item) handleSlashSelect(item.command);
        return;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        setSlashMenuItems([]);
        return;
      }
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleAllowOnce = () => {
    send({ type: 'allow_once', tools: permTools.map(t => t.tool_name) });
    setPermTools([]);
  };

  const handleDenyOnce = () => {
    send({ type: 'deny_once' });
    setPermTools([]);
  };

  /** 僅更新畫面選擇，送出訊息時才會送 set_mode */
  const handlePermModeDraftChange = (newMode) => {
    setMode(normalizePermMode(agentType, newMode));
  };
  /** 僅更新畫面選擇，送出訊息時才會送 set_model / set_effort */
  const handleModelDraftChange = (newModel) => setModelSel(newModel);
  const handleEffortDraftChange = (newEffort) => setEffortSel(newEffort);
  /** 授權面板「允許並記住」：須立即寫入後端並重試 */
  const handlePermModeCommitNow = (newMode) => {
    const n = normalizePermMode(agentType, newMode);
    setMode(n);
    if (!send({ type: 'set_mode', mode: n })) return;
    serverPermAppliedRef.current = n;
    setPermTools([]);
  };

  const handleInterrupt = () => send({ type: 'interrupt' });

  const handleInputModeChange = (newMode) => {
    const busy = ['THINKING', 'STREAMING', 'AWAITING_CONFIRM', 'SHELL_RUNNING', 'SHELL_AWAITING_APPROVAL', 'SHELL_EXEC', 'AWAITING_SHELL_CONFIRM'].includes(state);
    if (busy) return;
    setInputMode(newMode);
    try {
      localStorage.setItem(inputModeStorageKey(session.id), newMode);
    } catch (_) {}
    if (input.startsWith('/')) {
      const q = input.toLowerCase();
      const filtered = SLASH_COMMANDS.filter(
        (c) => (!c.modes || c.modes.includes(newMode)) && c.command.toLowerCase().startsWith(q)
      );
      setSlashMenuItems(filtered);
      setSlashActiveIdx(0);
    } else {
      setSlashMenuItems([]);
    }
  };

  const handleShellApprove = () => {
    send({ type: 'shell_approve' });
    setShellPendingCmd(null);
  };

  const handleShellCancel = () => {
    send({ type: 'shell_cancel' });
    setShellPendingCmd(null);
  };

  const isDisabled = state === 'THINKING' || state === 'STREAMING' || state === 'AWAITING_CONFIRM' || state === 'SHELL_RUNNING' || state === 'SHELL_AWAITING_APPROVAL' || state === 'SHELL_EXEC' || state === 'AWAITING_SHELL_CONFIRM';
  const modeSwitchDisabled = ['THINKING', 'STREAMING', 'AWAITING_CONFIRM', 'SHELL_RUNNING', 'SHELL_AWAITING_APPROVAL', 'SHELL_EXEC', 'AWAITING_SHELL_CONFIRM'].includes(state);
  const slashMenuOpen = slashMenuItems.length > 0;

  useEffect(() => {
    if (isDisabled) setSlashMenuItems([]);
  }, [isDisabled]);

  useEffect(() => {
    if (!slashMenuOpen) return;
    const onDocMouseDown = (e) => {
      if (slashInputWrapRef.current && !slashInputWrapRef.current.contains(e.target)) {
        setSlashMenuItems([]);
      }
    };
    document.addEventListener('mousedown', onDocMouseDown);
    return () => document.removeEventListener('mousedown', onDocMouseDown);
  }, [slashMenuOpen]);

  /** 聊天輸入框：依內容動態增高；未達上限不出現卷軸 */
  useLayoutEffect(() => {
    const el = chatInputRef.current;
    if (!el || state === 'THINKING' || state === 'STREAMING' || state === 'SHELL_RUNNING' || state === 'SHELL_EXEC' || state === 'AWAITING_SHELL_CONFIRM') return;
    el.style.height = 'auto';
    const maxStr = getComputedStyle(el).maxHeight;
    const maxPx = parseFloat(maxStr);
    const cap = Number.isFinite(maxPx) && maxPx > 0 ? maxPx : 12 * 16;
    const needed = el.scrollHeight;
    el.style.height = `${Math.min(needed, cap)}px`;
    el.style.overflowY = needed > cap + 1 ? 'auto' : 'hidden';
  }, [input, state]);

  return (
    <div className={`flex flex-col ${fullHeight ? 'h-app' : 'h-full'}`}>
      <ChatSessionHeader
        session={session}
        showBack={showBack}
        onBack={onBack}
        agentType={agentType}
        state={state}
        activityHint={activityHint}
        sessionModel={sessionModel}
        quota={quota}
        quotaRefreshing={quotaRefreshing}
        onQuotaRefresh={handleQuotaRefresh}
        headerCollapsed={headerCollapsed}
        onToggleHeader={toggleHeader}
        onExpandHeader={() => setHeaderCollapsed(false)}
        showPermModeSelect={showPermModeSelect}
        inputMode={inputMode}
        usePermModeDropdown={usePermModeDropdown}
        mode={mode}
        modeSwitchDisabled={modeSwitchDisabled}
        onPermModeChange={handlePermModeDraftChange}
        showEffortSelect={showEffortSelect}
        modelSel={modelSel}
        effortSel={effortSel}
        onModelChange={handleModelDraftChange}
        onEffortChange={handleEffortDraftChange}
      />

      {/* 訊息列表 */}
      <div
        ref={chatScrollRef}
        onScroll={syncChatNearBottom}
        className="flex-1 overflow-y-auto app-scroll px-4 py-[18px] sm:px-8 sm:py-7 flex flex-col gap-5"
      >
        {histLoaded && messages.length === 0 && (
          <div className="text-center text-gray-600 mt-20 text-sm">輸入指令開始對話</div>
        )}
        {messages.map((m, i) => {
          const msgKey = m.id != null ? String(m.id) : `idx-${i}`;
          const forwardBody =
            m.role === 'claude' && String(m.resultText || '').trim() !== ''
              ? m.resultText
              : (m.content || '').trim();
          const canForwardShellOrAgent =
            (m.role === 'claude' || m.role === 'shell') && !m.streaming && !!String(forwardBody || '').trim();
          const showStreamingTail =
            (state === 'THINKING' || state === 'STREAMING') &&
            m.role === 'claude' &&
            i === messages.length - 1;
          const timeLabel = formatMessageTime(m.createdAt);
          return (
          <div
            key={m.id != null ? `m-${m.id}` : i}
            className={`flex w-full min-w-0 ${m.role === 'user' ? 'justify-end' : 'justify-start'} ${m.role === 'claude' || m.role === 'shell' ? 'group' : ''}`}
          >
            <div className={`flex flex-col min-w-0 ${m.role === 'user' ? 'items-end max-w-[60%]' : m.role === 'shell' ? 'items-start max-w-[85%]' : 'items-start max-w-[78%]'}`}>
            {m.role === 'user' ? (
              <div className="bubble-user text-white px-4 py-3 text-sm w-fit max-w-full min-w-0">
                <div className="whitespace-pre-wrap break-words leading-relaxed">{m.content}</div>
              </div>
            ) : m.role === 'shell' ? (
              <div className="bubble-shell px-4 py-3 text-sm w-fit max-w-full min-w-0">
                <div className="flex items-start justify-between gap-2 mb-1.5 min-w-0">
                  <span className="inline-flex items-center gap-1 font-mono text-amber-500/80 text-xs">
                    <span>&gt;_</span>
                    <span>{shellType || 'shell'}</span>
                  </span>
                  <div className="shrink-0 flex items-center gap-0.5 opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity">
                    <MessageCopyButton text={forwardBody} className="p-1 rounded-md text-gray-400 hover:text-gray-200" />
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); setForwardModal({ messageKey: msgKey, messageContent: forwardBody }); }}
                      disabled={!canForwardShellOrAgent}
                      title="轉發到其他會話"
                      className="inline-flex items-center gap-0.5 px-1 py-0.5 rounded-md text-gray-400 hover:text-cyan-400 disabled:opacity-30"
                    >
                      <span className="text-[10px] font-medium">Forward</span>
                    </button>
                  </div>
                </div>
                <ShellOutput content={m.content || ''} exitCode={m.exitCode} streaming={m.streaming} />
                {forwardHints[msgKey] ? (
                  <div className="mt-2 pt-2 border-t border-gray-700/80 text-[11px] text-gray-500">
                    已 Forward 到「{forwardHints[msgKey].label}」
                    <button type="button" className="ml-1 text-violet-400 hover:text-violet-300 font-medium" onClick={() => jumpToSession(forwardHints[msgKey].session)}>前往查看 →</button>
                  </div>
                ) : null}
              </div>
            ) : (
              /* Claude — 設計 1a 文件流：頭像列 + 無邊框正文 */
              <div className="bubble-claude w-fit max-w-full min-w-0 flex flex-col gap-3">
                <div className="flex items-center justify-between gap-2 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className={`inline-flex items-center justify-center w-[22px] h-[22px] rounded-md shrink-0 ${getAgentBadgeClass(agentType)}`} aria-hidden>
                      <AgentBadgeIcon agentType={agentType} />
                    </span>
                    <span className="text-[13px] font-bold text-[oklch(0.85_0.01_264)]">{AGENT_LABEL[agentType] || agentType || 'Claude'}</span>
                  </div>
                  <div className="shrink-0 flex items-center gap-0.5 opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity">
                    <MessageCopyButton text={forwardBody} className="p-1 rounded-md text-gray-400 hover:text-gray-200" />
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); setForwardModal({ messageKey: msgKey, messageContent: forwardBody }); }}
                      disabled={!canForwardShellOrAgent}
                      title="轉發到其他會話"
                      className="inline-flex items-center gap-0.5 px-1 py-0.5 rounded-md text-gray-400 hover:text-cyan-400 disabled:opacity-30"
                    >
                      <span className="text-[10px] font-medium">Forward</span>
                    </button>
                  </div>
                </div>
                {m.thinking ? (
                  <div className="flex flex-row items-start gap-1.5">
                    <span className="text-violet-400/60 text-xs mt-0.5 shrink-0 select-none">💭</span>
                    <span className="text-[oklch(0.55_0.01_264)] text-sm italic leading-relaxed line-clamp-3 min-w-0">{m.content}</span>
                  </div>
                ) : (m.html || showStreamingTail) ? (
                  <div className="flex flex-row items-start gap-1">
                    {m.html && (
                      <div className="prose text-[oklch(0.85_0.01_264)] text-sm leading-[1.8] flex-1 min-w-0" dangerouslySetInnerHTML={{ __html: m.html }} />
                    )}
                    {showStreamingTail && (
                      <span className="streaming-tail shrink-0 select-none" aria-hidden="true">...</span>
                    )}
                  </div>
                ) : (
                  <div className="flex gap-1 py-1"><span className="dot"/><span className="dot"/><span className="dot"/></div>
                )}
                {forwardHints[msgKey] ? (
                  <div className="pt-1 text-[11px] text-[oklch(0.5_0.01_264)]">
                    已 Forward 到「{forwardHints[msgKey].label}」
                    <button type="button" className="ml-1 text-violet-400 hover:text-violet-300 font-medium" onClick={() => jumpToSession(forwardHints[msgKey].session)}>前往查看 →</button>
                  </div>
                ) : null}
              </div>
            )}
            {timeLabel ? (
              <div className="mt-1 px-0.5 text-[10px] leading-none text-[oklch(0.48_0.01_264)] tabular-nums select-none" title={String(m.createdAt || '')}>
                {timeLabel}
              </div>
            ) : null}
            </div>
          </div>
          );
        })}

        {/* Shell 指令批准對話框 */}
        {state === 'SHELL_AWAITING_APPROVAL' && shellPendingCmd && (
          <div className="rounded-xl border border-amber-700/60 bg-amber-950/30 px-4 py-3 text-sm mr-8">
            <div className="text-amber-400 font-semibold mb-2">⚠️ 即將執行 Shell 指令</div>
            <div className="text-gray-400 text-xs mb-1">Shell：{shellPendingCmd.shell_type || shellType}</div>
            {shellPendingCmd.work_dir && (
              <div className="text-gray-400 text-xs mb-2">目錄：<span className="font-mono text-gray-300">{shellPendingCmd.work_dir}</span></div>
            )}
            <pre className="bg-gray-900/60 rounded px-3 py-2 text-xs font-mono text-gray-200 mb-3 whitespace-pre-wrap break-words">{shellPendingCmd.command}</pre>
            <div className="flex gap-2">
              <button onClick={handleShellApprove}
                className="px-3 py-1.5 bg-amber-700 hover:bg-amber-600 text-white rounded-lg text-xs font-medium">
                確認執行
              </button>
              <button onClick={handleShellCancel}
                className="px-3 py-1.5 bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg text-xs">
                取消
              </button>
            </div>
          </div>
        )}

        {/* AI 授權確認 */}
        {state === 'AWAITING_CONFIRM' && permTools.length > 0 && (
          <div className="rounded-xl border border-yellow-700 bg-yellow-950/40 px-4 py-3 text-sm mr-8">
            <div className="text-yellow-400 font-semibold mb-2">需要授權</div>
            {permTools.map((t, i) => (
              <div key={i} className="text-gray-300 text-xs font-mono mb-1">
                {t.tool_name}
                {t.tool_input && (
                  <span className="text-gray-500"> — {JSON.stringify(t.tool_input).slice(0, 80)}</span>
                )}
              </div>
            ))}
            <div className="flex gap-2 mt-3">
              <button onClick={handleAllowOnce}
                className="px-3 py-1.5 bg-yellow-700 hover:bg-yellow-600 text-white rounded-lg text-xs">
                允許此操作
              </button>
              <button onClick={() => handlePermModeCommitNow('acceptEdits')}
                className="px-3 py-1.5 bg-orange-700 hover:bg-orange-600 text-white rounded-lg text-xs">
                允許並記住
              </button>
              <button onClick={handleDenyOnce}
                className="px-3 py-1.5 bg-red-900 hover:bg-red-800 text-white rounded-lg text-xs">
                拒絕
              </button>
            </div>
          </div>
        )}

        {state === 'AWAITING_SHELL_CONFIRM' && shellRequest && (
          <div className="rounded-xl border border-orange-700/90 bg-orange-950/35 px-4 py-3 text-sm mr-8">
            <div className="text-orange-300 font-semibold mb-2">允許執行 Shell 指令？</div>
            <div className="text-gray-200 text-xs font-mono break-words mb-1 whitespace-pre-wrap">{shellRequest.line}</div>
            <div className="text-gray-500 text-[10px] font-mono mb-2">
              指令名稱：<span className="text-orange-200/90">{shellRequest.command}</span>
              {shellRequest.workDirKey ? (
                <span className="block truncate mt-0.5" title={shellRequest.workDirKey}>目錄：{shellRequest.workDirKey}</span>
              ) : null}
            </div>
            <div className="flex flex-wrap gap-2 mt-2">
              <button type="button" onClick={() => { send({ type: 'shell_allow_once' }); }}
                className="px-3 py-1.5 bg-orange-800 hover:bg-orange-700 text-white rounded-lg text-xs">
                允許一次
              </button>
              <button type="button" onClick={() => { send({ type: 'shell_allow_remember_workdir' }); }}
                className="px-3 py-1.5 bg-amber-900 hover:bg-amber-800 text-amber-100 rounded-lg text-xs">
                允許並記住此目錄
              </button>
              <button type="button" onClick={() => { send({ type: 'shell_deny' }); }}
                className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg text-xs">
                拒絕
              </button>
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* 輸入區：水平內距略小於訊息列表，讓輸入框可用寬度較大 */}
      <div
        className={`shrink-0 px-4 sm:px-7 py-4 border-t transition-colors duration-200 ${inputMode === 'shell' ? 'bg-[oklch(0.15_0.02_264)] border-amber-900/40' : 'bg-[oklch(0.15_0.02_264)] border-[oklch(0.26_0.02_264)]'}`}
        style={{ paddingBottom: 'calc(0.75rem + env(safe-area-inset-bottom))' }}
      >
        {(state === 'STREAMING' || state === 'THINKING' || state === 'SHELL_RUNNING' || state === 'SHELL_EXEC') ? (
          <button onClick={handleInterrupt}
            className="w-full py-2 bg-red-900/60 hover:bg-red-800 text-red-300 rounded-lg text-sm">
            中斷
          </button>
        ) : (
          <div className={(inputMode === 'shell' ? 'ra-cmd-bar shell' : 'ra-cmd-bar') + ' w-full'}>
            <ModeToggleBtn
              value={inputMode}
              onChange={handleInputModeChange}
              disabled={modeSwitchDisabled}
              showLabel
              agentLabel={AGENT_LABEL[agentType] || 'Claude'}
            />
            <div className="relative flex flex-1 min-w-0 items-end" ref={slashInputWrapRef}>
              {slashMenuOpen && (
                <SlashCommandMenu
                  items={slashMenuItems}
                  activeIndex={slashActiveIdx}
                  onSelect={handleSlashSelect}
                />
              )}
              <textarea
                ref={chatInputRef}
                value={input}
                onChange={(e) => {
                  const value = e.target.value;
                  setInput(value);
                  try {
                    localStorage.setItem(draftInputStorageKey(session.id), value);
                  } catch (_) {}
                  if (value.startsWith('/')) {
                    const q = value.toLowerCase();
                    const filtered = SLASH_COMMANDS.filter(
                      (c) =>
                        (!c.modes || c.modes.includes(inputMode)) &&
                        c.command.toLowerCase().startsWith(q)
                    );
                    setSlashMenuItems(filtered);
                    setSlashActiveIdx(0);
                  } else {
                    setSlashMenuItems([]);
                  }
                }}
                onKeyDown={handleKeyDown}
                disabled={isDisabled}
                placeholder={inputMode === 'shell' ? `輸入 ${shellType || 'Shell'} 指令…` : '輸入指令…'}
                rows={1}
                className={[
                  'flex-1 min-w-0 w-full resize-none overflow-hidden border-0 bg-transparent px-1 py-1.5 text-[13.5px] leading-relaxed placeholder-[oklch(0.5_0.01_264)] focus:outline-none disabled:opacity-40 min-h-[2rem] max-h-[min(40vh,12rem)] box-border',
                  inputMode === 'shell' ? 'text-amber-50 font-mono placeholder-amber-900/60' : 'text-[oklch(0.9_0.01_264)]',
                ].join(' ')}
              />
            </div>
            <button type="button" onClick={() => handleSend()} disabled={isDisabled || !input.trim()}
              aria-label="送出"
              title="送出（Enter）"
              className={`shrink-0 flex items-center justify-center w-8 h-8 rounded-[8px] ${inputMode === 'shell' ? 'bg-amber-700 hover:bg-amber-600' : 'bg-[oklch(0.62_0.19_275)] hover:brightness-110'} disabled:opacity-30 text-white transition-colors text-sm`}>
              ➤
            </button>
          </div>
        )}
      </div>

      <ForwardModal
        payload={forwardModal}
        onClose={() => setForwardModal(null)}
        currentSessionId={session.id}
        allSessions={allSessions}
        usePermModeDropdown={usePermModeDropdown}
        onForwarded={({ messageKey, targetSession, jump }) => {
          setForwardHints((prev) => ({
            ...prev,
            [messageKey]: {
              label: `${targetSession.agent_type || 'claude'} / ${workDirGroupShortLabel(targetSession.work_dir)}`,
              session: targetSession,
            },
          }));
          if (jump) jumpToSession(targetSession);
        }}
      />
    </div>
  );
}

