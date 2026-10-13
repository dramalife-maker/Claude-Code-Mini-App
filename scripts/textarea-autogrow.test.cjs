const assert = require('node:assert/strict');

function adjustChatInputHeight(el, maxHeightCap = 240) {
  if (!el) return;
  el.style.height = 'auto';
  let cap = maxHeightCap;
  if (typeof getComputedStyle === 'function') {
    const maxStr = getComputedStyle(el).maxHeight;
    const maxPx = parseFloat(maxStr);
    if (Number.isFinite(maxPx) && maxPx > 0) {
      cap = maxPx;
    }
  }
  const needed = el.scrollHeight;
  if (needed > cap) {
    el.style.height = `${cap}px`;
    el.style.overflowY = 'auto';
  } else {
    el.style.height = `${needed}px`;
    el.style.overflowY = 'hidden';
  }
}

// Mock element helper
function createMockTextarea(scrollHeight, maxHeightStr = '240px') {
  return {
    scrollHeight,
    style: {
      height: '',
      overflowY: '',
    },
    maxHeightStr,
  };
}

global.getComputedStyle = (el) => ({
  maxHeight: el.maxHeightStr || '240px',
});

// Test 1: Single line (needed: 34, cap: 240) -> height 34px, overflowY hidden
const el1 = createMockTextarea(34);
adjustChatInputHeight(el1);
assert.equal(el1.style.height, '34px');
assert.equal(el1.style.overflowY, 'hidden');

// Test 2: Multi-line below cap (needed: 96, cap: 240) -> height 96px, overflowY hidden
const el2 = createMockTextarea(96);
adjustChatInputHeight(el2);
assert.equal(el2.style.height, '96px');
assert.equal(el2.style.overflowY, 'hidden');

// Test 3: Exceeding cap (needed: 350, cap: 240) -> height 240px, overflowY auto (scrollbar enabled!)
const el3 = createMockTextarea(350);
adjustChatInputHeight(el3);
assert.equal(el3.style.height, '240px');
assert.equal(el3.style.overflowY, 'auto');

// Test 4: Shrinking back from tall to single line
el3.scrollHeight = 34;
adjustChatInputHeight(el3);
assert.equal(el3.style.height, '34px');
assert.equal(el3.style.overflowY, 'hidden');

console.log('adjustChatInputHeight checks passed');
