const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

// 1. Check index.html script tag ordering
const indexHtml = fs.readFileSync(path.join(__dirname, '../internal/static/index.html'), 'utf8');
const popoverIdx = indexHtml.indexOf('SelectionQuotePopover.js');
const chatViewIdx = indexHtml.indexOf('ChatView.js');
const coreIdx = indexHtml.indexOf('core.js');

assert.ok(coreIdx > 0, 'core.js must be loaded in index.html');
assert.ok(popoverIdx > 0, 'SelectionQuotePopover.js must be loaded in index.html');
assert.ok(chatViewIdx > 0, 'ChatView.js must be loaded in index.html');
assert.ok(coreIdx < popoverIdx, 'core.js must be loaded before SelectionQuotePopover.js');
assert.ok(popoverIdx < chatViewIdx, 'SelectionQuotePopover.js must be loaded before ChatView.js');

// 2. Check SelectionQuotePopover.js exports and functions
const popoverSrc = fs.readFileSync(path.join(__dirname, '../internal/static/js/chat/SelectionQuotePopover.js'), 'utf8');
assert.ok(popoverSrc.includes('function computePopoverPosition'), 'computePopoverPosition function exists');
assert.ok(popoverSrc.includes('function detectMessageSelection'), 'detectMessageSelection function exists');
assert.ok(popoverSrc.includes('function SelectionQuotePopover'), 'SelectionQuotePopover component exists');
assert.ok(popoverSrc.includes('e.preventDefault()'), 'SelectionQuotePopover prevents default on mouseDown');

// 3. Check ChatView.js integration
const chatViewSrc = fs.readFileSync(path.join(__dirname, '../internal/static/js/chat/ChatView.js'), 'utf8');
assert.ok(chatViewSrc.includes('quoteSelection'), 'ChatView includes quoteSelection state');
assert.ok(chatViewSrc.includes('detectMessageSelection'), 'ChatView uses detectMessageSelection');
assert.ok(chatViewSrc.includes('injectMarkdownQuote'), 'ChatView uses injectMarkdownQuote');
assert.ok(chatViewSrc.includes('SelectionQuotePopover'), 'ChatView renders SelectionQuotePopover');
assert.ok(chatViewSrc.includes('focusChatInputToEnd'), 'ChatView focuses cursor to end of input');

// 4. Verify brackets balance in modified JS files
for (const file of [
  '../internal/static/js/core/core.js',
  '../internal/static/js/chat/SelectionQuotePopover.js',
  '../internal/static/js/chat/ChatView.js',
]) {
  const content = fs.readFileSync(path.join(__dirname, file), 'utf8');
  let parens = 0, braces = 0, brackets = 0;
  for (const ch of content) {
    if (ch === '(') parens++;
    else if (ch === ')') parens--;
    else if (ch === '{') braces++;
    else if (ch === '}') braces--;
    else if (ch === '[') brackets++;
    else if (ch === ']') brackets--;
  }
  assert.equal(parens, 0, `Unbalanced parentheses in ${file}`);
  assert.equal(braces, 0, `Unbalanced braces in ${file}`);
  assert.equal(brackets, 0, `Unbalanced brackets in ${file}`);
}

console.log('all integration and syntax checks passed successfully!');
