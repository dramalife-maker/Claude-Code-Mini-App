const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, '../internal/static/js/core/core.js'), 'utf8');
const context = vm.createContext({
  React: {
    useState: () => {},
    useEffect: () => {},
    useLayoutEffect: () => {},
    useRef: () => ({ current: null }),
    useCallback: (fn) => fn,
    useMemo: (fn) => fn(),
  },
  window: { location: { pathname: '', protocol: 'http:', host: 'localhost' } },
  location: { pathname: '', protocol: 'http:', host: 'localhost' },
  sessionStorage: { getItem: () => null, setItem: () => {}, removeItem: () => {} },
  localStorage: { getItem: () => null, setItem: () => {}, removeItem: () => {} },
  document: { documentElement: { style: { setProperty: () => {} } } },
  marked: { setOptions: () => {}, use: () => {}, parse: (s) => s },
  DOMPurify: { sanitize: (s) => s },
});
vm.runInContext(source, context);

const { formatMarkdownQuote, injectMarkdownQuote } = context;

// Test empty input -> quote inserted
let input = '';
input = injectMarkdownQuote(input, 'Selected error line');
assert.equal(input, '> Selected error line\n\n');

// Test existing draft -> appended with double newline
input = injectMarkdownQuote(input, 'Second quote line');
assert.equal(input, '> Selected error line\n\n> Second quote line\n\n');

// Test mock element setSelectionRange simulation
const mockEl = {
  value: input,
  disabled: false,
  selectionStart: 0,
  selectionEnd: 0,
  focus: () => {},
  setSelectionRange: function(start, end) {
    this.selectionStart = start;
    this.selectionEnd = end;
  },
};

const len = mockEl.value.length;
mockEl.setSelectionRange(len, len);
assert.equal(mockEl.selectionStart, len);
assert.equal(mockEl.selectionEnd, len);

console.log('quote reply action checks passed');
