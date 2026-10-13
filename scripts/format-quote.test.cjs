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
    useRef: () => {},
    useCallback: () => {},
    useMemo: () => {},
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

assert.equal(typeof formatMarkdownQuote, 'function', 'formatMarkdownQuote should be a function');
assert.equal(typeof injectMarkdownQuote, 'function', 'injectMarkdownQuote should be a function');

// 1. Single line quote
assert.equal(formatMarkdownQuote('Hello world'), '> Hello world\n\n');

// 2. Multi-line quote
assert.equal(
  formatMarkdownQuote('const a = 1;\nconst b = 2;'),
  '> const a = 1;\n> const b = 2;\n\n'
);

// 3. Multi-line with blank lines
assert.equal(
  formatMarkdownQuote('Line 1\n\nLine 2'),
  '> Line 1\n>\n> Line 2\n\n'
);

// 4. Empty or whitespace-only
assert.equal(formatMarkdownQuote(''), '');
assert.equal(formatMarkdownQuote('   \n  \t '), '');

// 5. injectMarkdownQuote into empty draft
assert.equal(injectMarkdownQuote('', 'Hello world'), '> Hello world\n\n');

// 6. injectMarkdownQuote into existing draft
assert.equal(
  injectMarkdownQuote('My question:', 'Hello world'),
  'My question:\n\n> Hello world\n\n'
);

// 7. injectMarkdownQuote into existing draft that already ends with newline
assert.equal(
  injectMarkdownQuote('My question:\n', 'Hello world'),
  'My question:\n\n> Hello world\n\n'
);

console.log('format quote checks passed');
