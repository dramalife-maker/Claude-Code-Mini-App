const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// 桌面更新的純函式（無 React 依賴）：快照 → 顯示文字／按鈕模型。
const targetFile = path.join(__dirname, '../internal/static/js/ui/desktop-update.js');
const source = fs.readFileSync(targetFile, 'utf8');
const context = vm.createContext({
  window: {},
  console: console,
});
vm.runInContext(source, context);

const {
  desktopUpdateStateLabel,
  desktopUpdateErrorText,
  desktopUpdateBusyReasonText,
  desktopUpdateProgressPercent,
  desktopUpdateView,
  desktopUpdateShortSha,
} = context;

// vm 上下文的陣列與主 realm 的陣列原型不同，這裡用 Array.from 轉成主 realm 陣列再比較。
function actionIds(view) {
  return Array.from(view.actions, (a) => a.id);
}

function findAction(view, id) {
  return view.actions.find((a) => a.id === id);
}

// 1. 狀態文字：每個 Go 端 State 都有對應文字，未知值不會拋錯。
{
  const states = ['idle', 'checking', 'up-to-date', 'available', 'downloading', 'verifying', 'ready', 'restarting', 'error'];
  for (const s of states) {
    const label = desktopUpdateStateLabel(s);
    assert.ok(label && label !== '未知狀態', `state ${s} should have a label`);
  }
  assert.strictEqual(desktopUpdateStateLabel('no-such-state'), '未知狀態');
}

// 2. 錯誤碼文字：每個設計中的錯誤碼都有人讀得懂的文字；未知碼回退為通用文字。
{
  const codes = [
    'check_failed',
    'asset_missing',
    'checksum_missing',
    'checksum_mismatch',
    'download_failed',
    'install_not_writable',
    'update_busy',
    'update_in_progress',
    'helper_failed',
    'update_check_failed',
  ];
  for (const c of codes) {
    assert.ok(desktopUpdateErrorText(c).length > 0, `error ${c} needs text`);
  }
  assert.strictEqual(desktopUpdateErrorText('mystery'), '更新失敗');
}

// 3. 進度百分比：總量未知或為 0 時為 0，超量時截在 100。
{
  assert.strictEqual(desktopUpdateProgressPercent(0, 0), 0);
  assert.strictEqual(desktopUpdateProgressPercent(50, 0), 0);
  assert.strictEqual(desktopUpdateProgressPercent(50, 200), 25);
  assert.strictEqual(desktopUpdateProgressPercent(200, 200), 100);
  assert.strictEqual(desktopUpdateProgressPercent(300, 200), 100);
  assert.strictEqual(desktopUpdateProgressPercent(undefined, undefined), 0);
}

// 4. SHA 顯示：取前 7 碼，空值顯示破折號（與設定頁版號旁的顯示一致）。
{
  assert.strictEqual(desktopUpdateShortSha('abcdef1234567'), 'abcdef1');
  assert.strictEqual(desktopUpdateShortSha(''), '—');
  assert.strictEqual(desktopUpdateShortSha(undefined), '—');
}

// 5. 狀態 → 動作模型。
{
  const idle = desktopUpdateView({ state: 'idle' }, '');
  assert.deepStrictEqual(actionIds(idle), ['check']);
  assert.strictEqual(findAction(idle, 'check').disabled, false);

  const upToDate = desktopUpdateView({ state: 'up-to-date' }, '');
  assert.deepStrictEqual(actionIds(upToDate), ['check']);

  const available = desktopUpdateView({ state: 'available', availableVersion: '1.0.1' }, '');
  assert.deepStrictEqual(actionIds(available), ['download', 'later', 'check']);
  assert.strictEqual(findAction(available, 'download').disabled, false);
  assert.strictEqual(available.offerVisible, true);
  assert.strictEqual(available.showNotes, true);
}

// 6. 忙碌狀態：下載與驗證中不可再次檢查或下載；ready 時檢查也停用（後端回 ErrBusy）。
{
  for (const state of ['checking', 'downloading', 'verifying', 'ready']) {
    const v = desktopUpdateView({ state, availableVersion: '1.0.1' }, '');
    assert.strictEqual(findAction(v, 'check').disabled, true, `check disabled in ${state}`);
    assert.ok(!actionIds(v).includes('download'), `no download action in ${state}`);
    assert.strictEqual(v.busy, true, `busy in ${state}`);
  }
}

// 7. ready 的重啟：可點（送出前由 DesktopUpdateSection 二次確認）；其他狀態不出現。
{
  const v = desktopUpdateView({ state: 'ready', availableVersion: '1.0.1' }, '');
  const restart = findAction(v, 'restart');
  assert.ok(restart, 'restart action shown in ready');
  assert.strictEqual(restart.disabled, false);
  assert.strictEqual(restart.label, '重新啟動並更新');
  for (const state of ['idle', 'available', 'downloading', 'verifying', 'restarting', 'error', 'up-to-date']) {
    const other = desktopUpdateView({ state, availableVersion: '1.0.1' }, '');
    assert.ok(!actionIds(other).includes('restart'), `no restart action in ${state}`);
  }
}

// 7b. restarting：後端已取得閘門並呼叫 Restart，前端一律忙碌、不可再檢查或下載，文字顯示重新啟動中。
{
  const v = desktopUpdateView({ state: 'restarting', availableVersion: '1.0.1' }, '');
  assert.strictEqual(v.busy, true, 'restarting is busy');
  assert.strictEqual(v.stateLabel, '重新啟動中…');
  assert.strictEqual(findAction(v, 'check').disabled, true, 'check disabled while restarting');
  assert.ok(!actionIds(v).includes('download'), 'no download while restarting');
  assert.ok(!actionIds(v).includes('restart'), 'no second restart while restarting');
}

// 7c. 忙碌拒絕（update_busy）：仍是 ready，重啟可再按；錯誤與原因文字都有內容。
{
  const v = desktopUpdateView({ state: 'ready', availableVersion: '1.0.1', errorCode: 'update_busy', busyReason: 'queued' }, '');
  assert.strictEqual(findAction(v, 'restart').disabled, false, 'restart can be retried after busy refusal');
  assert.ok(desktopUpdateErrorText('update_busy').includes('進行中的工作'), 'update_busy explains busy work');
  assert.ok(desktopUpdateErrorText('update_check_failed').length > 0, 'update_check_failed has text');
  assert.ok(desktopUpdateErrorText('helper_failed').includes('可再試'), 'helper_failed says retry is possible');
  assert.strictEqual(desktopUpdateBusyReasonText('queued'), '有排隊中的訊息');
  assert.strictEqual(desktopUpdateBusyReasonText('running_task'), '有任務執行中或仍在收尾');
  assert.strictEqual(desktopUpdateBusyReasonText('shell_task'), '有終端機指令執行中');
  assert.strictEqual(desktopUpdateBusyReasonText('pending_confirm'), '有等待授權或確認的操作');
  assert.strictEqual(desktopUpdateBusyReasonText('no-such-reason'), '');
  assert.strictEqual(desktopUpdateBusyReasonText(undefined), '');
}

// 8. 錯誤後重試：有可用版本時重試下載，否則重試檢查。
{
  const afterDownload = desktopUpdateView({ state: 'error', availableVersion: '1.0.1', errorCode: 'download_failed' }, '');
  assert.ok(actionIds(afterDownload).includes('download'), 'retry download after download failure');
  assert.strictEqual(findAction(afterDownload, 'download').label, '重試');

  const afterCheck = desktopUpdateView({ state: 'error', availableVersion: '', errorCode: 'check_failed' }, '');
  assert.ok(!actionIds(afterCheck).includes('download'), 'no download after check failure');
  assert.strictEqual(findAction(afterCheck, 'check').label, '重試');
  assert.strictEqual(actionIds(afterCheck).filter((id) => id === 'check').length, 1, 'retry replaces the plain check button');
}

// 9. 稍後：只影響顯示，同一版本延後後不再顯示下載提示，可再次顯示。
{
  const deferred = desktopUpdateView({ state: 'available', availableVersion: '1.0.1' }, '1.0.1');
  assert.strictEqual(deferred.deferred, true);
  assert.strictEqual(deferred.offerVisible, false);
  assert.strictEqual(deferred.showNotes, false);
  assert.ok(!actionIds(deferred).includes('download'), 'deferred offer hides download');
  assert.ok(actionIds(deferred).includes('showOffer'), 'deferred offer can be shown again');

  // 新版本不會被舊的延後版本吃掉。
  const newer = desktopUpdateView({ state: 'available', availableVersion: '1.0.2' }, '1.0.1');
  assert.strictEqual(newer.deferred, false);
  assert.strictEqual(newer.offerVisible, true);
}

// 10. 沒有快照（尚未收到 Go 端狀態）視為 idle，不拋錯。
{
  const v = desktopUpdateView(null, '');
  assert.strictEqual(v.stateLabel, desktopUpdateStateLabel('idle'));
  assert.ok(actionIds(v).includes('check'));
}

// 11. 只有 Go 端定義的動作 id 會出現；絕不出現由呼叫者指定 URL 或路徑的動作。
{
  const states = ['idle', 'checking', 'up-to-date', 'available', 'downloading', 'verifying', 'ready', 'restarting', 'error'];
  const allowed = new Set(['check', 'download', 'later', 'showOffer', 'restart']);
  for (const s of states) {
    for (const v of [desktopUpdateView({ state: s, availableVersion: '1.0.1' }, ''), desktopUpdateView({ state: s, availableVersion: '1.0.1' }, '1.0.1')]) {
      for (const id of actionIds(v)) {
        assert.ok(allowed.has(id), `unexpected action ${id} in ${s}`);
      }
    }
  }
}

console.log('desktop-update tests passed');
