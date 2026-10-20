/**
 * Telegram 時間線合併（純函式，無 React 依賴）。
 * HTTP 分頁、WS sync、live upsert、待傳取代與送達事件，全部依穩定識別合併，不覆蓋已載入的舊頁。
 * 穩定識別：telegram {account_id, peer_id, message_id}；尚未取得遠端識別的本機待傳項目用本機 id／client_message_id。
 * 項目是 mapMessageRow 的輸出（含 telegram、delivery、clientMessageId、replacesClientMessageId）。
 */

// 送達狀態只往前進：sent 不被過期的 pending／failed 降級。
const TG_DELIVERY_RANK = { '': 0, pending: 1, unknown: 2, failed: 2, sent: 3 };

function tgRefKey(ref) {
  return ref ? `${ref.account_id}:${ref.peer_id}:${ref.message_id}` : '';
}

function tgScopeKey(scope) {
  return scope ? `${scope.account_id}:${scope.peer_id}` : '';
}

function tgTimeValue(m) {
  const s = m && m.createdAt;
  if (!s) return Infinity;
  // 後端的 SQLite 時間（UTC，無時區標記）與 ISO 字串都轉成毫秒再比較。
  const iso = /^\d{4}-\d\d-\d\d \d\d:\d\d:\d\d$/.test(s) ? `${s.replace(' ', 'T')}Z` : s;
  const v = Date.parse(iso);
  return Number.isNaN(v) ? Infinity : v;
}

function tgCmpNum(a, b) {
  return a === b ? 0 : (a < b ? -1 : 1);
}

/**
 * 只保留屬於目前已驗證 scope 的列（沒有 telegram 識別的待傳／本機列不受影響）。
 * scopeKey 為空（尚未收到已驗證 scope）時不過濾。用於 HTTP 與 live 事件：
 * 帳號切換後才到的舊帳號資料不得回填。
 */
function tgFilterScope(rows, scopeKey) {
  if (!scopeKey) return rows;
  return rows.filter((r) => !r.telegram || `${r.telegram.account_id}:${r.telegram.peer_id}` === scopeKey);
}

/** 排序：原始時間 → Telegram message_id（待傳排在同時間之後）→ 本機 id。 */
function tgCompare(a, b) {
  return tgCmpNum(tgTimeValue(a), tgTimeValue(b))
    || tgCmpNum(a.telegram ? a.telegram.message_id : Infinity, b.telegram ? b.telegram.message_id : Infinity)
    || tgCmpNum(a.id != null ? a.id : Infinity, b.id != null ? b.id : Infinity);
}

function tgIsSame(m, row, refKey, replaces) {
  if (refKey && tgRefKey(m.telegram) === refKey) return true;
  if (row.id != null && m.id === row.id) return true;
  return !!replaces && (String(m.id) === replaces || m.clientMessageId === replaces);
}

/**
 * 合併單列（不排序）。同一遠端訊息只留一個；replaces 指向的待傳氣泡一併移除。
 * 已有遠端識別的項目維持原本的 id，遲到的重播不會把它換成另一個本機 id。
 */
function tgUpsertUnsorted(list, row) {
  const refKey = tgRefKey(row.telegram);
  const replaces = row.replacesClientMessageId ? String(row.replacesClientMessageId) : '';
  const matched = list.filter((m) => tgIsSame(m, row, refKey, replaces));
  if (matched.length === 0) {
    return [...list, replaces ? { ...row, replacesClientMessageId: '' } : row];
  }
  const base = matched.find((m) => row.id != null && m.id === row.id) || matched[0];
  // 過期的待傳表示（沒有遠端識別）晚於已確認的訊息到達：忽略。
  if (!row.telegram && base.telegram) return list;
  const keepOld = (TG_DELIVERY_RANK[base.delivery || ''] || 0) > (TG_DELIVERY_RANK[row.delivery || ''] || 0);
  const delivery = keepOld ? base.delivery : (row.delivery || base.delivery || '');
  const sameRemote = refKey && tgRefKey(base.telegram) === refKey;
  const merged = {
    ...base,
    ...row,
    id: sameRemote ? base.id : row.id,
    delivery,
    deliveryError: delivery === 'sent' ? '' : (keepOld ? base.deliveryError : (row.deliveryError || '')),
    attachments: row.attachments && row.attachments.length ? row.attachments : (base.attachments || []),
    replacesClientMessageId: '',
  };
  return [...list.filter((m) => !matched.includes(m) && !(replaces && (String(m.id) === replaces || m.clientMessageId === replaces))), merged];
}

/** 合併一批列（HTTP 分頁、sync）。既有項目不會因為這批沒有它而消失。 */
function tgMergeList(list, rows) {
  let next = list;
  for (const row of rows) next = tgUpsertUnsorted(next, row);
  return next === list ? list : [...next].sort(tgCompare);
}

/** 合併單列（live 完整 upsert、user_message）。 */
function tgUpsert(list, row) {
  return tgMergeList(list, [row]);
}

/** 送達事件：把狀態與錯誤呈現在原待傳項目上；找不到項目不新增；已 sent 不降級。 */
function tgApplyDelivery(list, evt) {
  const id = evt.clientMessageId != null ? String(evt.clientMessageId) : '';
  if (!id) return list;
  let changed = false;
  const next = list.map((m) => {
    if (changed || (m.clientMessageId !== id && String(m.id) !== id)) return m;
    if ((TG_DELIVERY_RANK[m.delivery || ''] || 0) > (TG_DELIVERY_RANK[evt.delivery] || 0) || m.delivery === 'sent') return m;
    changed = true;
    return { ...m, delivery: evt.delivery, deliveryError: evt.error || '' };
  });
  return changed ? next : list;
}

/**
 * 改綁：scope 與前一次不同，或已載入的列有不屬於新 scope 的 Telegram 項目（例如 HTTP 先載入了舊帳號資料）。
 * 無 scope（離線／暫時無法驗證）不算改綁。
 */
function tgScopeChanged(list, scope, prevScopeKey) {
  if (!scope) return false;
  const key = tgScopeKey(scope);
  return (!!prevScopeKey && prevScopeKey !== key)
    || list.some((m) => m.telegram && `${m.telegram.account_id}:${m.telegram.peer_id}` !== key);
}

/**
 * 分頁請求失敗 → 頂部控制要顯示的狀態。錯誤永遠不會變成 has_more=false（呼叫端只更新 error／retryAt／needsInit）。
 * 429：依 retry_after_seconds（或 Retry-After 標頭）算出 retryAt，期間不得再請求；
 * 400：游標失效，重試時改取最新頁（needsInit）；其餘（503、網路）用同一個游標重試。
 */
function tgPageError(status, body, retryAfterHeader, nowMs) {
  const b = body && typeof body === 'object' ? body : {};
  if (status === 429) {
    const secs = Number(b.retry_after_seconds != null ? b.retry_after_seconds : retryAfterHeader) || 1;
    return { message: `Telegram 要求稍後再試（約 ${secs} 秒後可重試）`, retryAt: nowMs + secs * 1000, needsInit: false };
  }
  if (status === 400) return { message: '游標已失效，將重新載入最新訊息', retryAt: 0, needsInit: true };
  if (status === 503) return { message: 'Telegram 未連線，請重試', retryAt: 0, needsInit: false };
  return { message: '載入較早訊息失敗，請重試', retryAt: 0, needsInit: false };
}

/**
 * 初次 HTTP 失敗時的分頁狀態：游標未知（hasMore=null，不是到頂），needsInit 讓下一次載入改取最新頁，
 * 並顯示明確的重試入口。
 */
function tgInitFailed(err) {
  return { nextCursor: null, hasMore: null, syncState: '', loading: false, error: { message: err.message }, needsInit: true, retryAt: err.retryAt };
}

/**
 * WS sync（Telegram 只帶最新有界頁）：併入而不取代。
 * scope＝目前已驗證的 account/peer（未驗證或離線為 null：保留已有資料）。
 * 改綁（scope 與前一次不同）時，丟棄不屬於新 scope 的項目與舊 scope 的待傳，避免帶進新聊天。
 */
function tgApplySync(list, rows, scope, prevScopeKey) {
  let base = list;
  if (scope) {
    const key = tgScopeKey(scope);
    const changed = tgScopeChanged(list, scope, prevScopeKey);
    base = list.filter((m) => (m.telegram
      ? m.telegram.account_id === scope.account_id && m.telegram.peer_id === scope.peer_id
      : !changed));
  }
  return tgMergeList(base, rows);
}
