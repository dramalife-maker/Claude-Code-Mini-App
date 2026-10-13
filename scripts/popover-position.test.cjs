const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, '../internal/static/js/chat/SelectionQuotePopover.js'), 'utf8');
const context = vm.createContext({
  React: {
    useState: () => {},
    useEffect: () => {},
    useLayoutEffect: () => {},
    useRef: () => ({ current: null }),
    useCallback: (fn) => fn,
  },
  window: { innerWidth: 1024, innerHeight: 768 },
});
const funcSource = source.slice(0, source.indexOf('function SelectionQuotePopover'));
vm.runInContext(funcSource, context);

const { computePopoverPosition } = context;
assert.equal(typeof computePopoverPosition, 'function', 'computePopoverPosition should be a function');

// Test 1: Normal positioning above selection
const pos1 = computePopoverPosition(
  { top: 200, bottom: 230, left: 100, right: 300, width: 200, height: 30 },
  80, 36, 1024, 768
);
// center X is 100 + 200/2 = 200; left is 200 - 80/2 = 160
assert.equal(pos1.left, 160);
// top above: 200 - 36 - 8 = 156
assert.equal(pos1.top, 156);
assert.equal(pos1.placement, 'top');

// Test 2: Near top boundary -> flip to bottom
const pos2 = computePopoverPosition(
  { top: 20, bottom: 40, left: 100, right: 300, width: 200, height: 20 },
  80, 36, 1024, 768
);
// flips to bottom: 40 + 8 = 48
assert.equal(pos2.top, 48);
assert.equal(pos2.placement, 'bottom');

// Test 3: Horizontal clamping on left edge
const pos3 = computePopoverPosition(
  { top: 200, bottom: 230, left: 10, right: 30, width: 20, height: 30 },
  80, 36, 1024, 768
);
// center is 20, 20 - 40 = -20, clamped to min 12
assert.ok(pos3.left >= 12, 'should clamp left to at least 12');

// Test 4: Horizontal clamping on right edge
const pos4 = computePopoverPosition(
  { top: 200, bottom: 230, left: 1000, right: 1020, width: 20, height: 30 },
  80, 36, 1024, 768
);
// clamped to 1024 - 80 - 12 = 932
assert.ok(pos4.left <= 1024 - 80 - 12, 'should clamp right to viewport boundary');

console.log('popover position checks passed');
