const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// 1. Load core.js for injectMarkdownQuote
const coreSource = fs.readFileSync(path.join(__dirname, '../internal/static/js/core/core.js'), 'utf8');
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
vm.runInContext(coreSource, context);

// 2. Load SelectionQuotePopover.js for detectMessageSelection
const popoverSource = fs.readFileSync(path.join(__dirname, '../internal/static/js/chat/SelectionQuotePopover.js'), 'utf8');
const funcSource = popoverSource.slice(0, popoverSource.indexOf('function SelectionQuotePopover'));
vm.runInContext(funcSource, context);

const { detectMessageSelection, injectMarkdownQuote } = context;

// Simulate Pane A and Pane B
const paneA = {
  container: {
    id: 'pane-a-container',
    contains: (node) => node && node.pane === 'A',
  },
  input: 'Draft in Pane A',
};

const paneB = {
  container: {
    id: 'pane-b-container',
    contains: (node) => node && node.pane === 'B',
  },
  input: 'Draft in Pane B',
};

// User highlights text in Pane B
const selectionInPaneB = {
  isCollapsed: false,
  rangeCount: 1,
  toString: () => 'Error: 500 in agent response',
  getRangeAt: () => ({
    commonAncestorContainer: {
      pane: 'B',
      nodeType: 1,
      closest: () => null,
    },
    getBoundingClientRect: () => ({ top: 150, left: 600, width: 120, height: 24 }),
  }),
};

// Check pane isolation:
const resultForA = detectMessageSelection(selectionInPaneB, paneA.container);
const resultForB = detectMessageSelection(selectionInPaneB, paneB.container);

// Pane A should NOT detect selection
assert.equal(resultForA, null, 'Pane A container must ignore selection belonging to Pane B');

// Pane B should detect selection
assert.ok(resultForB, 'Pane B container must detect its own selection');
assert.equal(resultForB.text, 'Error: 500 in agent response');

// Simulate quote action in Pane B
paneB.input = injectMarkdownQuote(paneB.input, resultForB.text);

// Assert Pane B updated, Pane A completely unchanged
assert.equal(
  paneB.input,
  'Draft in Pane B\n\n> Error: 500 in agent response\n\n',
  'Pane B input must have quote injected'
);
assert.equal(
  paneA.input,
  'Draft in Pane A',
  'Pane A input must remain unaltered'
);

console.log('split pane quote isolation checks passed');
