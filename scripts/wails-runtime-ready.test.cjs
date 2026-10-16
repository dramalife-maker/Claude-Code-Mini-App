const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// 視窗載入外部 URL（本機 server 首頁）時，Wails runtime 不會被載入，Go 端 ExecJS 會排隊等待
// 'wails:runtime:ready' 才 flush；這裡確認頁面只在擁有者首頁、WebView2 可用時送出一次該訊號。
const targetFile = path.join(__dirname, '../internal/static/js/wails-runtime-ready.js');
const indexFile = path.join(__dirname, '../internal/static/index.html');
const source = fs.readFileSync(targetFile, 'utf8');

// 在 vm 中載入腳本；win 為模擬的 window（腳本會以全域 window 執行自動宣告）。
function load(win) {
  const context = vm.createContext({ window: win, console: console });
  vm.runInContext(source, context);
  return context;
}

function fakeWindow({ pathname = '/', webview = true, postMessage } = {}) {
  const calls = [];
  const win = {
    location: { pathname },
    chrome: webview ? { webview: { postMessage: postMessage || ((m) => calls.push(m)) } } : undefined,
  };
  return { win, calls };
}

// 1. WebView2 + 擁有者首頁 → 送出一次 wails:runtime:ready。
{
  const { win, calls } = fakeWindow({ pathname: '/' });
  load(win);
  assert.deepStrictEqual(calls, ['wails:runtime:ready']);
}

// 2. 同一視窗再次載入腳本不重複送出。
{
  const { win, calls } = fakeWindow({ pathname: '/' });
  load(win);
  load(win);
  assert.strictEqual(calls.length, 1);
}

// 3. 一般瀏覽器（沒有 chrome.webview）→ 不送出、不丟例外。
{
  const { win, calls } = fakeWindow({ pathname: '/', webview: false });
  assert.doesNotThrow(() => load(win));
  assert.deepStrictEqual(calls, []);
}

// 4. chrome.webview 存在但 postMessage 不是函式 → 不送出、不丟例外。
{
  const win = { location: { pathname: '/' }, chrome: { webview: { postMessage: 'nope' } } };
  assert.doesNotThrow(() => load(win));
}

// 5. 非擁有者首頁（分享頁等）→ 不送出。
for (const pathname of ['/share/abc', '/share/abc/', '/index.html', '/js/app.js', '']) {
  const { win, calls } = fakeWindow({ pathname });
  assert.doesNotThrow(() => load(win), `pathname=${pathname}`);
  assert.deepStrictEqual(calls, [], `pathname=${pathname} should not announce`);
}

// 6. 缺少 location 也不丟例外。
{
  assert.doesNotThrow(() => load({ chrome: { webview: { postMessage: () => {} } } }));
}

// 7. index.html 把腳本放在最前面（任何 CDN／Babel／React 腳本之前）。
{
  const html = fs.readFileSync(indexFile, 'utf8');
  const tag = html.indexOf('src="./js/wails-runtime-ready.js"');
  assert.ok(tag > 0, 'index.html must load js/wails-runtime-ready.js');
  const firstOther = html.search(/<script\s[^>]*src="https?:|<script\s[^>]*type="text\/babel"/);
  assert.ok(firstOther === -1 || tag < firstOther, 'runtime-ready script must precede other scripts');
  assert.ok(html.indexOf('<head>') < tag, 'runtime-ready script must be inside <head>');
}

console.log('wails-runtime-ready tests passed');
