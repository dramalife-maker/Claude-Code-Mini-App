const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, '../internal/static/js/core/core.js'), 'utf8');
const context = vm.createContext({});
vm.runInContext(source.slice(source.indexOf('function formatKiroCreditUsed('), source.indexOf('function permModeLabelFor(')), context);
const compact = context.formatQuotaCompact;

const exhausted = 'Gemini 5h 0% · Gemini Week 0% · Claude/GPT 5h 100% · Claude/GPT Week 80%';
assert.equal(compact(exhausted, 'antigravity'), '100%');
assert.equal(compact(exhausted, 'gemini'), '100%');
assert.equal(compact('Gemini 5h 2% · Gemini Week 16%', 'antigravity'), '16%');
assert.equal(compact('Claude/GPT Week 99.5%', 'antigravity'), '99.5%');
assert.equal(compact('—', 'antigravity'), '');
assert.equal(compact('Total 30% · Auto 8% · API 100%', 'cursor'), 'Total 30%');
assert.equal(compact('FREE · Credits 9% (4.52/50)', 'kiro'), '9% (4.5/50)');
console.log('quota compact checks passed');
