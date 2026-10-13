const assert = require('node:assert/strict');

function detectMessageSelection(sel, containerEl) {
  if (!sel || sel.isCollapsed || sel.rangeCount === 0) return null;
  const text = (sel.toString ? sel.toString() : '').trim();
  if (!text) return null;
  const range = sel.getRangeAt(0);
  if (!range) return null;
  const common = range.commonAncestorContainer;
  if (!containerEl || !containerEl.contains(common)) return null;
  const ancestorEl = common.nodeType === 1 ? common : common.parentElement;
  if (ancestorEl && ancestorEl.closest && ancestorEl.closest('input, textarea, button, [data-chat-input]')) return null;
  const rect = range.getBoundingClientRect ? range.getBoundingClientRect() : null;
  if (!rect || (rect.width === 0 && rect.height === 0)) return null;
  return { text, targetRect: rect };
}

// 1. Collapsed selection returns null
assert.equal(detectMessageSelection({ isCollapsed: true, rangeCount: 1 }, {}), null);

// 2. Selection outside container returns null
const mockContainer = {
  contains: (node) => node.id === 'inside',
};
const outsideSel = {
  isCollapsed: false,
  rangeCount: 1,
  toString: () => 'hello',
  getRangeAt: () => ({
    commonAncestorContainer: { id: 'outside', nodeType: 1 },
    getBoundingClientRect: () => ({ top: 10, left: 10, width: 50, height: 20 }),
  }),
};
assert.equal(detectMessageSelection(outsideSel, mockContainer), null);

// 3. Selection inside textarea returns null
const insideTextareaSel = {
  isCollapsed: false,
  rangeCount: 1,
  toString: () => 'hello',
  getRangeAt: () => ({
    commonAncestorContainer: {
      id: 'inside',
      nodeType: 1,
      closest: (sel) => (sel.includes('textarea') ? {} : null),
    },
    getBoundingClientRect: () => ({ top: 10, left: 10, width: 50, height: 20 }),
  }),
};
assert.equal(detectMessageSelection(insideTextareaSel, mockContainer), null);

// 4. Valid selection inside container message returns result
const validSel = {
  isCollapsed: false,
  rangeCount: 1,
  toString: () => 'hello world',
  getRangeAt: () => ({
    commonAncestorContainer: {
      id: 'inside',
      nodeType: 1,
      closest: () => null,
    },
    getBoundingClientRect: () => ({ top: 10, left: 10, width: 50, height: 20 }),
  }),
};
const res = detectMessageSelection(validSel, mockContainer);
assert.ok(res);
assert.equal(res.text, 'hello world');
assert.equal(res.targetRect.width, 50);

console.log('detectMessageSelection checks passed');
