const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { webcrypto } = require('node:crypto');
const { test } = require('node:test');

const targetFile = path.join(__dirname, '../internal/static/js/setup/setupForm.js');
const source = fs.readFileSync(targetFile, 'utf8');
const context = vm.createContext({ console, crypto: webcrypto });
vm.runInContext(source, context);

const {
  setupPasswordError,
  generateSetupPassword,
  parseWhitelist,
  defaultSetupForm,
  validateSetupForm,
  buildSetupPayload,
  readSetupToken,
  stripSetupToken,
  shouldShowSetup,
} = context;

const alphabet = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789';
const hex64 = 'a'.repeat(64);

function validForm(overrides) {
  const form = defaultSetupForm(8080);
  form.password = 'longpassword';
  return Object.assign(form, overrides);
}

test('TestPasswordRules', () => {
  assert.notEqual(setupPasswordError(''), '');
  assert.notEqual(setupPasswordError('   \t'), '');
  assert.notEqual(setupPasswordError('change-me'), '');
  assert.notEqual(setupPasswordError('  change-me  '), '');
  assert.notEqual(setupPasswordError('一二三四五六七'), '');
  assert.equal(setupPasswordError('一二三四五六七八'), '');
  assert.equal(setupPasswordError('  longpassword  '), '');
});

test('TestGeneratedPasswordValid', () => {
  const seen = new Set();
  for (let i = 0; i < 200; i++) {
    const pw = generateSetupPassword();
    assert.equal(setupPasswordError(pw), '');
    assert.equal(pw.length, 20);
    for (const ch of pw) {
      assert.ok(alphabet.includes(ch), ch);
    }
    seen.add(pw);
  }
  assert.ok(seen.size > 1);
});

test('TestPayloadMiniAppOff', () => {
  const form = validForm();
  form.miniapp.bot_token = '123:ABC';
  form.miniapp.whitelist_text = '7, 8';
  const payload = buildSetupPayload(form);
  const text = JSON.stringify(payload);
  assert.equal(payload.miniapp.enabled, false);
  assert.equal(Object.prototype.hasOwnProperty.call(payload.miniapp, 'bot_token'), false);
  assert.equal(Object.prototype.hasOwnProperty.call(payload.miniapp, 'whitelist_tg_ids'), false);
  assert.equal(text.includes('bot_token'), false);
  assert.equal(text.includes('whitelist_tg_ids'), false);
});

test('TestPayloadMiniAppOn', () => {
  const form = validForm();
  form.miniapp.enabled = true;
  form.miniapp.bot_token = '  123:ABC  ';
  form.miniapp.whitelist_text = '7, 8, 7';
  form.allow_lan = true;
  form.db_path = '  ./claude-miniapp.db  ';
  const payload = buildSetupPayload(form);
  assert.equal(payload.password, 'longpassword');
  assert.equal(payload.port, 8080);
  assert.equal(payload.db_path, './claude-miniapp.db');
  assert.equal(payload.allow_lan, true);
  assert.equal(payload.miniapp.enabled, true);
  assert.equal(payload.miniapp.bot_token, '123:ABC');
  assert.deepEqual(Array.from(payload.miniapp.whitelist_tg_ids), [7, 8]);
});

test('TestWhitelistParse', () => {
  const mixed = parseWhitelist('7, 8\n9\t7');
  assert.equal(mixed.error, '');
  assert.deepEqual(Array.from(mixed.ids), [7, 8, 9]);

  const empty = parseWhitelist('');
  assert.equal(empty.error, '');
  assert.deepEqual(Array.from(empty.ids), []);
  const blank = parseWhitelist('  \n, ');
  assert.equal(blank.error, '');
  assert.deepEqual(Array.from(blank.ids), []);

  for (const bad of ['01', '-3', '1.5', '9007199254740992', '0']) {
    const got = parseWhitelist('7, ' + bad);
    assert.notEqual(got.error, '', bad);
    assert.ok(got.error.includes(bad), got.error);
    assert.deepEqual(Array.from(got.ids), []);
  }
});

test('TestValidateSetupForm', () => {
  const off = validForm();
  off.miniapp.bot_token = '';
  off.miniapp.whitelist_text = '0';
  const offResult = validateSetupForm(off);
  assert.equal(offResult.ok, true);
  assert.equal(offResult.errors['miniapp.bot_token'], undefined);
  assert.equal(offResult.errors['miniapp.whitelist_tg_ids'], undefined);

  const on = validForm();
  on.miniapp.enabled = true;
  on.miniapp.bot_token = '   ';
  const onResult = validateSetupForm(on);
  assert.equal(onResult.ok, false);
  assert.ok(onResult.errors['miniapp.bot_token']);

  for (const port of [0, 65536, 1.5, '8080.0']) {
    const form = validForm({ port });
    const got = validateSetupForm(form);
    assert.equal(got.ok, false, String(port));
    assert.ok(got.errors.port, String(port));
  }

  const slash = validForm({ db_path: './data/' });
  const slashGot = validateSetupForm(slash);
  assert.equal(slashGot.ok, false);
  assert.ok(slashGot.errors.db_path);
  const back = validForm({ db_path: '.\\data\\' });
  assert.ok(validateSetupForm(back).errors.db_path);
});

test('TestReadSetupToken', () => {
  assert.equal(readSetupToken('?setup=' + hex64), hex64);
  assert.equal(readSetupToken('?x=1&setup=' + hex64), hex64);
  assert.equal(readSetupToken(''), '');
  assert.equal(readSetupToken('?setup=' + 'a'.repeat(63)), '');
  assert.equal(readSetupToken('?setup=' + 'A'.repeat(64)), '');
  assert.equal(readSetupToken('?setup=' + 'g'.repeat(64)), '');
  assert.equal(readSetupToken('?foo=1'), '');
});

test('TestStripSetupToken', () => {
  const token = '?setup=' + hex64;
  assert.equal(stripSetupToken('http://127.0.0.1:8080/' + token), 'http://127.0.0.1:8080/');
  assert.equal(
    stripSetupToken('http://127.0.0.1:8080/?x=1&setup=' + hex64 + '&y=2'),
    'http://127.0.0.1:8080/?x=1&y=2',
  );
  assert.equal(stripSetupToken('http://127.0.0.1:8080/?x=1'), 'http://127.0.0.1:8080/?x=1');
});

test('TestShouldShowSetup', () => {
  assert.equal(shouldShowSetup({ needs_setup: true }), true);
  assert.equal(shouldShowSetup({ needs_setup: false }), false);
  assert.equal(shouldShowSetup({ needs_setup: 'true' }), false);
  assert.equal(shouldShowSetup({}), false);
  assert.equal(shouldShowSetup(null), false);
  assert.equal(shouldShowSetup(undefined), false);
});
