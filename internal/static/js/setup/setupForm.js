// 設定精靈的表單純函式。不碰 DOM。
// 這個檔會以傳統 script 載入，也會被 Node vm 載入；要給測試用的只能是 function 宣告。

function setupPasswordError(pw) {
  var text = String(pw == null ? '' : pw).trim();
  if (text === '') return '密碼不可為空';
  if (text === 'change-me') return '密碼不可使用 change-me';
  if (Array.from(text).length < 8) return '密碼至少需要 8 個字元';
  return '';
}

function generateSetupPassword() {
  // 排除 0 O o 1 l I。56 個字元不能整除 256，超過 224 的位元組丟掉，避免取餘數偏掉。
  var alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789';
  var out = '';
  var buf = new Uint8Array(32);
  while (out.length < 20) {
    crypto.getRandomValues(buf);
    for (var i = 0; i < buf.length && out.length < 20; i++) {
      if (buf[i] >= 224) continue;
      out += alphabet[buf[i] % 56];
    }
  }
  return out;
}

function parseWhitelist(text) {
  var raw = String(text == null ? '' : text);
  if (raw.trim() === '') return { ids: [], error: '' };
  var parts = raw.split(/[\s,]+/).filter(Boolean);
  var ids = [];
  var seen = {};
  for (var i = 0; i < parts.length; i++) {
    var item = parts[i];
    if (!setupWhitelistItemOK(item)) {
      return { ids: [], error: '「' + item + '」不是正整數' };
    }
    var n = Number(item);
    if (seen[n]) continue;
    seen[n] = true;
    ids.push(n);
  }
  return { ids: ids, error: '' };
}

function setupWhitelistItemOK(item) {
  if (!/^[1-9][0-9]*$/.test(item)) return false;
  var max = '9007199254740991';
  if (item.length > max.length) return false;
  if (item.length === max.length && item > max) return false;
  return true;
}

function defaultSetupForm(port) {
  return {
    password: '',
    port: port,
    db_path: './claude-miniapp.db',
    allow_lan: false,
    miniapp: { enabled: false, bot_token: '', whitelist_text: '' },
  };
}

function validateSetupForm(form) {
  var errors = {};
  var passwordError = setupPasswordError(form.password);
  if (passwordError) errors.password = passwordError;
  var portError = setupPortError(form.port);
  if (portError) errors.port = portError;
  var dbError = setupDBPathError(form.db_path);
  if (dbError) errors.db_path = dbError;
  var mini = form.miniapp || {};
  if (mini.enabled === true) {
    if (String(mini.bot_token == null ? '' : mini.bot_token).trim() === '') {
      errors['miniapp.bot_token'] = '開啟 Mini App 時 bot_token 不可為空';
    }
    var parsed = parseWhitelist(mini.whitelist_text);
    if (parsed.error) errors['miniapp.whitelist_tg_ids'] = parsed.error;
  }
  return { ok: Object.keys(errors).length === 0, errors: errors };
}

function setupPortError(port) {
  var bad = '連接埠必須是 1 到 65535 的整數';
  if (typeof port === 'number') {
    if (Number.isInteger(port) && port >= 1 && port <= 65535) return '';
    return bad;
  }
  if (typeof port === 'string' && /^(0|[1-9][0-9]*)$/.test(port.trim())) {
    var n = Number(port.trim());
    if (n >= 1 && n <= 65535) return '';
  }
  return bad;
}

function setupDBPathError(dbPath) {
  var text = String(dbPath == null ? '' : dbPath).trim();
  if (text === '') return '資料庫路徑不可為空';
  if (text.endsWith('/') || text.endsWith('\\')) return '資料庫路徑請指定檔名，不要以斜線結尾';
  return '';
}

function buildSetupPayload(form) {
  var mini = form.miniapp || {};
  var payload = {
    password: form.password,
    port: setupPortNumber(form.port),
    db_path: String(form.db_path == null ? '' : form.db_path).trim(),
    allow_lan: form.allow_lan === true,
    miniapp: { enabled: false },
  };
  if (mini.enabled === true) {
    var parsed = parseWhitelist(mini.whitelist_text);
    payload.miniapp = {
      enabled: true,
      bot_token: String(mini.bot_token == null ? '' : mini.bot_token).trim(),
      whitelist_tg_ids: parsed.ids,
    };
  }
  return payload;
}

function setupPortNumber(port) {
  if (typeof port === 'number') return port;
  if (typeof port === 'string') return Number(port.trim());
  return Number(port);
}

function readSetupToken(search) {
  var q = String(search == null ? '' : search);
  if (q.charAt(0) === '?') q = q.slice(1);
  if (q === '') return '';
  var parts = q.split('&');
  for (var i = 0; i < parts.length; i++) {
    var eq = parts[i].indexOf('=');
    var key = eq < 0 ? parts[i] : parts[i].slice(0, eq);
    if (key !== 'setup') continue;
    var token = eq < 0 ? '' : parts[i].slice(eq + 1);
    if (/^[0-9a-f]{64}$/.test(token)) return token;
    return '';
  }
  return '';
}

function stripSetupToken(href) {
  var raw = String(href == null ? '' : href);
  var hash = '';
  var hashAt = raw.indexOf('#');
  if (hashAt >= 0) {
    hash = raw.slice(hashAt);
    raw = raw.slice(0, hashAt);
  }
  var qAt = raw.indexOf('?');
  if (qAt < 0) return raw + hash;
  var kept = [];
  var parts = raw.slice(qAt + 1).split('&');
  for (var i = 0; i < parts.length; i++) {
    if (parts[i] === '') continue;
    var eq = parts[i].indexOf('=');
    var key = eq < 0 ? parts[i] : parts[i].slice(0, eq);
    if (key === 'setup') continue;
    kept.push(parts[i]);
  }
  if (kept.length === 0) return raw.slice(0, qAt) + hash;
  return raw.slice(0, qAt) + '?' + kept.join('&') + hash;
}

function shouldShowSetup(status) {
  return !!status && status.needs_setup === true;
}

// 狀態還沒回來時是 pending，不掛正常 App。查詢失敗不是首次設定，也不要空白。
function setupBootView(state) {
  if (!state || state.pending) return 'pending';
  if (state.error) return 'error';
  if (shouldShowSetup(state.status)) return 'wizard';
  if (state.status && state.status.needs_setup === false) return 'app';
  return 'error';
}

function setupTokenHint(token) {
  if (token) return '';
  return '請在這台電腦開啟啟動時印出的設定網址（網址裡有一次性憑證）。沒有憑證就不能儲存。';
}

function setupCliAdvisory(tools) {
  var list = Array.isArray(tools) ? tools : [];
  var any = false;
  for (var i = 0; i < list.length; i++) {
    if (list[i] && list[i].found) any = true;
  }
  return {
    blockSave: false,
    recommend: !any,
    message: any ? '' : '這台電腦還沒找到代理程式。仍可先完成設定；建立對話前建議至少安裝一個。',
  };
}

function setupAccountReady(form) {
  return setupPasswordError(form.password) === '' && setupPortError(form.port) === '' && setupDBPathError(form.db_path) === '';
}

function setupCanSave(form) {
  return validateSetupForm(form).ok === true;
}

function setupApplyGeneratedPassword(form) {
  var next = {};
  for (var key in form) next[key] = form[key];
  next.password = generateSetupPassword();
  return next;
}

function setupSaveOutcome(status, body) {
  var data = body || {};
  if (status === 200 && data.applied === 'switched') return { kind: 'switched', url: data.url || '/' };
  if (status === 200 && data.applied === 'restart_required') return { kind: 'restart', url: data.url || '', resend: false };
  if (status === 500 && data.code === 'start_failed') {
    return { kind: 'start_failed', error: data.error || '', resend: false };
  }
  if (status === 400) return { kind: 'invalid', field: data.field || '', error: data.error || '' };
  if (status === 403) return { kind: 'forbidden', error: data.error || '' };
  if (status === 409) return { kind: 'exists', error: data.error || '設定檔已存在', resend: false };
  if (status === 500) return { kind: 'write_failed', error: data.error || '', resend: true };
  return { kind: 'error', error: data.error || '儲存失敗', resend: true };
}

function setupWizardSteps() {
  return ['welcome', 'account', 'miniapp', 'review'];
}
