const { useState, useEffect, useLayoutEffect, useRef, useCallback, useMemo } = React;

// ── Telegram WebApp 初始化 ────────────────────────────────────────────────────
const tgApp = window.Telegram?.WebApp;
// 判斷是否在 Telegram WebView 裡（即使 initData 暫時為空也算）
const isTMA = !!(tgApp);
const initData = isTMA ? (tgApp.initData || '') : '';
// 有 initData 才走 TG 驗證，否則 fallback 到密碼驗證
const isTelegram = isTMA && initData !== '';

if (isTMA) {
  tgApp.ready();
  tgApp.expand();
  tgApp.disableVerticalSwipes?.();
  const updateContentSafeArea = () => {
    const top = tgApp.contentSafeAreaInset?.top ?? 0;
    document.documentElement.style.setProperty('--tg-content-safe-top', `${top}px`);
  };
  updateContentSafeArea();
  tgApp.onEvent('contentSafeAreaChanged', updateContentSafeArea);
}

/** Web 登入 session token（localStorage；後端仍驗證於伺服器端 store） */
const WEB_SESSION_STORAGE_KEY = 'cc_web_session_token';

/** 桌面版側欄寬度（localStorage，px；與原 max-w-[48vw] 上限一致） */
const SIDEBAR_WIDTH_STORAGE_KEY = 'cc_sidebar_width_px';
const SIDEBAR_WIDTH_DEFAULT = 340;
const SIDEBAR_WIDTH_MIN = 260;

function readStoredSidebarWidthPx() {
  try {
    const raw = localStorage.getItem(SIDEBAR_WIDTH_STORAGE_KEY);
    const n = parseInt(raw, 10);
    if (Number.isFinite(n) && n > 0) return n;
  } catch (_) {}
  return SIDEBAR_WIDTH_DEFAULT;
}

/** Session 列表「依工作目錄分組」開關（localStorage） */
const SESSION_LIST_GROUP_BY_DIR_KEY = 'cc_session_list_group_by_dir';

function readStoredGroupByDir() {
  try {
    const raw = localStorage.getItem(SESSION_LIST_GROUP_BY_DIR_KEY);
    if (raw === '1' || raw === 'true') return true;
    if (raw === '0' || raw === 'false') return false;
  } catch (_) {}
  return true; /* RemoteAgent 設計預設依目錄分組 */
}

function clampSidebarWidthPx(w) {
  const max = Math.max(SIDEBAR_WIDTH_MIN, Math.floor(window.innerWidth * 0.48));
  return Math.min(Math.max(Math.round(w), SIDEBAR_WIDTH_MIN), max);
}

function sidebarWidthMaxPx() {
  return Math.max(SIDEBAR_WIDTH_MIN, Math.floor(window.innerWidth * 0.48));
}

/** 桌面版側欄收合狀態（localStorage） */
const SIDEBAR_COLLAPSED_KEY = 'cc_sidebar_collapsed';

function readStoredSidebarCollapsed() {
  try {
    return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === '1';
  } catch (_) {
    return false;
  }
}

/**
 * 正規化 API base path：無結尾 /；根為 ''；一律以 / 開頭（非空時）。
 */
function normalizeApiBasePath(raw) {
  if (raw == null) return null;
  let b = String(raw).trim();
  if (b === '' || b === '/') return '';
  b = b.replace(/\/+$/, '');
  if (!b.startsWith('/')) b = `/${b}`;
  return b;
}

/**
 * 可選覆寫：反代若無法由網址自動對齊 API 節點時使用。
 * - <meta name="cc-api-base" content="/myapp">（需在 bundle 執行前存在於 DOM）
 * - 或於任何 script 以前：window.__CC_API_BASE__ = '/myapp'
 */
function readApiBaseOverride() {
  try {
    const el = document.querySelector('meta[name="cc-api-base"]');
    if (el) {
      const c = el.getAttribute('content');
      if (c != null && String(c).trim() !== '') {
        return normalizeApiBasePath(c);
      }
    }
  } catch (_) {}
  if (typeof window.__CC_API_BASE__ === 'string' && window.__CC_API_BASE__.trim() !== '') {
    return normalizeApiBasePath(window.__CC_API_BASE__);
  }
  return null;
}

/**
 * 與 Go Fiber 同掛一條反代前綴時：目前 index 所在目錄為 API 前綴。
 * 若在 …/Nexus/、…/Focus/、…/Enterprise/、…/RemoteAgent/、…/v1/ 測試主題底下，再剝除該目錄，與同層主介面共用前綴。
 */
function inferApiBaseFromPathname() {
  let p = window.location.pathname || '/';
  if (/\/[^/]+\.html$/i.test(p)) {
    p = p.replace(/\/[^/]+\.html$/i, '');
  }
  p = p.replace(/\/+$/, '');
  p = p.replace(/\/nexus$/i, '').replace(/\/focus$/i, '').replace(/\/enterprise$/i, '').replace(/\/remoteagent$/i, '').replace(/\/v1$/i, '');
  p = p.replace(/\/+$/, '');
  if (p === '' || p === '/') return '';
  return p;
}

function getSpaBasePath() {
  const over = readApiBaseOverride();
  if (over !== null) return over;
  return inferApiBaseFromPathname();
}

/** API 路徑（必須以 / 開頭）加上 base，供反代剝離前綴時瀏覽器仍打對外完整路徑。 */
function appPath(path) {
  const rel = path.startsWith('/') ? path : `/${path}`;
  const base = getSpaBasePath();
  if (!base) return rel;
  return `${base}${rel}`;
}

function resolveApiUrl(url) {
  if (/^https?:\/\//i.test(url)) return url;
  if (url.startsWith('/')) return appPath(url);
  return url;
}

console.log(
  '[init] isTMA:', isTMA,
  'isTelegram:', isTelegram,
  'initData len:', initData.length,
  'protocol:', location.protocol,
  'spaBase:', getSpaBasePath() || '(root)',
);

// Web session 過期時統一登出（僅 Web 密碼登入，TMA 不受影響）
let onSessionExpired = null;
const registerSessionExpiredHandler = (fn) => { onSessionExpired = fn; };

const clearWebSession = () => {
  try {
    const t = localStorage.getItem(WEB_SESSION_STORAGE_KEY);
    localStorage.removeItem(WEB_SESSION_STORAGE_KEY);
    if (t) {
      fetch(resolveApiUrl('/auth/logout'), {
        method: 'POST',
        headers: { Authorization: `Bearer ${t}` },
      }).catch(() => {});
    }
  } catch (_) {}
  onSessionExpired?.();
};

const handleUnauthorized = (res) => {
  if (!isTelegram && res.status === 401) {
    clearWebSession();
    return true;
  }
  return false;
};

// 統一 fetch：TMA 帶 initData；Web 帶 Bearer（localStorage），舊版仍相容後端 Cookie
const apiFetch = async (url, opts = {}) => {
  const headers = { ...(opts.headers || {}) };
  if (isTelegram) {
    headers['X-Telegram-Init-Data'] = initData;
  } else {
    try {
      const t = localStorage.getItem(WEB_SESSION_STORAGE_KEY);
      if (t) headers['Authorization'] = `Bearer ${t}`;
    } catch (_) {}
  }
  const res = await fetch(resolveApiUrl(url), { ...opts, headers });
  handleUnauthorized(res);
  return res;
};

// WebSocket：同源 ws/wss；TMA 用 query initData；Web 用 query token（瀏覽器無法自訂 WS Header）
const wsURL = (sessionId) => {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  let url = `${proto}://${location.host}${appPath(`/sessions/${sessionId}/ws`)}`;
  const sep = (u) => (u.includes('?') ? '&' : '?');
  if (isTelegram) {
    return `${url}${sep(url)}initData=${encodeURIComponent(initData)}`;
  }
  try {
    const t = localStorage.getItem(WEB_SESSION_STORAGE_KEY);
    if (t) return `${url}${sep(url)}token=${encodeURIComponent(t)}`;
  } catch (_) {}
  return url;
};

// marked v5+ 已移除同步 `highlight` option（改走 extension/後處理），此 setOptions 的 highlight
// 已對新版 CDN marked 無效果；改在 parseMarkdown 輸出後手動跑 hljs（見下方）。
marked.setOptions({
  breaks: true,
});

const parseMarkdown = (text) => {
  const html = marked.parse(text);
  return html
    .replace(/<table>/g, '<div class="table-wrap"><table>')
    .replace(/<\/table>/g, '</table></div>')
    // <pre><code class="language-xxx">raw text</code></pre> → 用 hljs 上色 + 提升 data-lang 到 <pre> + 加複製按鈕。
    .replace(/<pre><code class="language-([\w+-]+)">([\s\S]*?)<\/code><\/pre>/g, (m, lang, inner) => {
      const raw = inner.replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"').replace(/&#39;/g, "'");
      let out;
      try {
        out = hljs.getLanguage(lang) ? hljs.highlight(raw, { language: lang }).value : hljs.highlightAuto(raw).value;
      } catch (_) {
        out = inner; // 高亮失敗時保留原輸出，不讓整段訊息渲染中斷。
      }
      // 原始文字以 base64 存進 data 屬性，避免複製鍵讀取時再度處理 HTML entity escape。
      const b64 = btoa(unescape(encodeURIComponent(raw)));
      const copyBtn = `<button type="button" class="code-copy-btn" data-copy-b64="${b64}" aria-label="複製程式碼" title="複製">`
        + `<svg class="icon-copy" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15V5a2 2 0 0 1 2-2h10"/></svg>`
        + `<svg class="icon-check" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" style="display:none"><path d="M20 6 9 17l-5-5"/></svg>`
        + `</button>`;
      return `<pre data-lang="${lang}">${copyBtn}<code class="language-${lang} hljs">${out}</code></pre>`;
    });
};

/** 顯示用：permission_mode 值 → 中文標籤。包含 Claude/Cursor 與 Gemini 兩套值。 */
const PERM_MODE_LABEL = {
  default: '預設',
  acceptEdits: '自動編輯',
  bypassPermissions: '⚠️ 跳過授權',
  auto_edit: '自動編輯',
  yolo: '⚠️ YOLO（全自動）',
  plan: '規劃模式',
};

const AGENT_LABEL = { claude: 'Claude', cursor: 'Cursor', codex: 'Codex', antigravity: 'Antigravity', kiro: 'Kiro', kiroacp: 'Kiro ACP' };

/** 聊天室頂欄狀態角標：中文短標（避免與權限列搶注意力） */
const SESSION_STATE_LABEL = {
  IDLE: '待命',
  THINKING: '思考中',
  STREAMING: '輸出中',
  AWAITING_CONFIRM: '待授權',
  SHELL_IDLE: '待命',
  SHELL_AWAITING_APPROVAL: '等待確認',
  SHELL_RUNNING: '執行中',
  AWAITING_SHELL_CONFIRM: '待確認 Shell',
  SHELL_EXEC: 'Shell 執行中',
};

const CHAT_HEADER_COLLAPSED_KEY = 'cc_chat_header_collapsed';

function readChatHeaderCollapsedPref() {
  try {
    const v = localStorage.getItem(CHAT_HEADER_COLLAPSED_KEY);
    if (v === '1') return true;
    if (v === '0') return false;
  } catch (_) {}
  return null;
}

function writeChatHeaderCollapsedPref(collapsed) {
  try {
    localStorage.setItem(CHAT_HEADER_COLLAPSED_KEY, collapsed ? '1' : '0');
  } catch (_) {}
}

function useMediaQuery(query) {
  const [matches, setMatches] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia(query).matches : false
  );
  useEffect(() => {
    if (typeof window === 'undefined') return undefined;
    const mq = window.matchMedia(query);
    const onChange = () => setMatches(mq.matches);
    mq.addEventListener('change', onChange);
    onChange();
    return () => mq.removeEventListener('change', onChange);
  }, [query]);
  return matches;
}

/** 手機預設收合；偏好寫入 localStorage */
function useChatHeaderCollapsed() {
  const isMobile = useMediaQuery('(max-width: 639px)');
  const [collapsed, setCollapsedState] = useState(() => {
    const saved = readChatHeaderCollapsedPref();
    if (saved != null) return saved;
    return typeof window !== 'undefined' && window.matchMedia('(max-width: 639px)').matches;
  });
  const setCollapsed = useCallback((v) => {
    setCollapsedState(v);
    writeChatHeaderCollapsedPref(v);
  }, []);
  return {
    collapsed: isMobile && collapsed,
    setCollapsed,
    isMobile,
    toggle: () => setCollapsed(!collapsed),
  };
}

function formatKiroCreditUsed(displayText) {
  const pct = String(displayText).match(/Credits\s+([\d.]+)\s*%/);
  const frac = String(displayText).match(/\(([\d.]+)\s*\/\s*([\d.]+)\)/);
  if (!pct && !frac) return '';
  const fmt = (n) => (Math.abs(n - Math.round(n)) < 0.05 ? String(Math.round(n)) : n.toFixed(1));
  if (pct && frac) return pct[1] + '% (' + fmt(parseFloat(frac[1])) + '/' + fmt(parseFloat(frac[2])) + ')';
  if (pct) return pct[1] + '%';
  return fmt(parseFloat(frac[1])) + '/' + fmt(parseFloat(frac[2]));
}

function formatQuotaCompact(displayText, agentType) {
  if (!displayText || displayText === '—') return '';
  if (String(agentType || '').toLowerCase() === 'kiro' || String(agentType || '').toLowerCase() === 'kiroacp') {
    const kiro = formatKiroCreditUsed(displayText);
    if (kiro) return kiro;
  }
  const first = String(displayText).split(' · ')[0].trim();
  if (first.length <= 11) return first;
  const m = first.match(/(\d+(?:\.\d+)?)\s*%/);
  return m ? m[0] : first.slice(0, 10) + '…';
}

function permModeLabelFor(agentType, value) {
  const hit = permModeOptionsFor(agentType).find((o) => o.value === value);
  return hit ? hit.label : value;
}

function permModeIsDanger(value) {
  return value === 'bypassPermissions' || value === 'yolo';
}

function sessionStateChipClass(state) {
  // 與 session 列表下方方塊 badge 同構：無邊框、軟底色
  if (state === 'IDLE' || state === 'SHELL_IDLE') return 'bg-[oklch(0.7_0.15_155_/16%)] text-[oklch(0.7_0.15_155)]';
  if (state === 'THINKING') return 'bg-violet-500/20 text-violet-300';
  if (state === 'STREAMING') return 'bg-sky-500/20 text-sky-300';
  if (state === 'SHELL_EXEC' || state === 'SHELL_RUNNING') return 'bg-amber-500/20 text-amber-200';
  if (state === 'AWAITING_SHELL_CONFIRM' || state === 'SHELL_AWAITING_APPROVAL' || state === 'AWAITING_CONFIRM') return 'bg-orange-500/20 text-orange-200';
  return 'bg-amber-500/20 text-amber-200';
}

