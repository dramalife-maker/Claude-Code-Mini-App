# selection-quote-reply Specification

## Purpose

Provides contextual text selection detection in chat messages and displays a floating quote action button. Clicking the quote action formats the selected text into a Markdown blockquote, inserts it into the active composer input, and positions the focus cursor for immediate follow-up input.

## Requirements

### Requirement: Text selection detection and quote popover display
The system SHALL detect non-empty text selections inside chat message content and display a floating quote popover near the selected range. The system SHALL NOT display the quote popover for selections within text input or textarea elements.

#### Scenario: Display popover on message text selection
- **WHEN** user selects non-empty text inside an assistant or user message
- **THEN** floating quote popover appears adjacent to the selected text containing a quote action button

##### Example: Text selection in assistant response
- **GIVEN** an assistant message containing "function calculateTotal() { return 42; }"
- **WHEN** user selects "calculateTotal()"
- **THEN** floating popover appears above the selection with button "引言"

#### Scenario: Ignore selection inside chat input
- **WHEN** user selects text inside the chat textarea
- **THEN** floating quote popover does not appear

##### Example: Selection inside textarea draft
- **GIVEN** the composer textarea contains "Draft message text"
- **WHEN** user highlights "Draft message" inside the textarea
- **THEN** no quote popover is rendered

#### Scenario: Dismiss popover on selection collapse or outside click
- **WHEN** user clicks outside the popover or clears the current text selection
- **THEN** floating quote popover disappears

##### Example: Clicking empty space outside popover
- **GIVEN** a floating quote popover is currently visible
- **WHEN** user clicks on the chat background area
- **THEN** selection is collapsed and the popover unmounts

#### Scenario: Dismiss popover on container scroll
- **WHEN** user scrolls the chat message container while popover is active
- **THEN** floating quote popover is dismissed

##### Example: Scrolling while popover is active
- **GIVEN** a floating quote popover is displayed
- **WHEN** user scrolls down 100 pixels in the chat message container
- **THEN** the popover immediately hides


<!-- @trace
source: selection-quote-reply
updated: 2026-10-13
code:
  - scripts/split-pane-quote.test.cjs
  - internal/static/js/chat/SelectionQuotePopover.js
  - internal/static/index.html
  - scripts/popover-position.test.cjs
  - scripts/selection-detect.test.cjs
  - scripts/quote-reply-action.test.cjs
  - scripts/textarea-autogrow.test.cjs
  - internal/static/js/core/core.js
  - scripts/selection-quote-integration.test.cjs
  - scripts/format-quote.test.cjs
  - internal/static/js/chat/ChatView.js
-->

---
### Requirement: Markdown quote insertion and input focus
The system SHALL format selected text into Markdown blockquote format and insert it into the active chat input when quote action is invoked.

#### Scenario: Insert quoted text into empty composer
- **WHEN** user clicks the quote action button while composer input is empty
- **THEN** composer input value is updated with the selected text formatted as Markdown blockquote lines
- **THEN** composer input receives focus with cursor placed after the quoted block

##### Example: Quote single line into empty composer
- **GIVEN** composer input is empty
- **WHEN** user clicks quote button with selected text "Hello world"
- **THEN** composer input value becomes "> Hello world\n\n"
- **THEN** cursor position is set to index 15 (end of the string)

#### Scenario: Append quote to existing composer draft
- **WHEN** user clicks the quote action button while composer input contains existing text
- **THEN** composer input appends the Markdown blockquote separated by newline characters
- **THEN** composer input receives focus with cursor placed after the appended block

##### Example: Quote into existing draft
- **GIVEN** composer input contains "My question:"
- **WHEN** user clicks quote button with selected text "const x = 1;"
- **THEN** composer input value becomes "My question:\n\n> const x = 1;\n\n"
- **THEN** cursor position is set to the end of the text


<!-- @trace
source: selection-quote-reply
updated: 2026-10-13
code:
  - scripts/split-pane-quote.test.cjs
  - internal/static/js/chat/SelectionQuotePopover.js
  - internal/static/index.html
  - scripts/popover-position.test.cjs
  - scripts/selection-detect.test.cjs
  - scripts/quote-reply-action.test.cjs
  - scripts/textarea-autogrow.test.cjs
  - internal/static/js/core/core.js
  - scripts/selection-quote-integration.test.cjs
  - scripts/format-quote.test.cjs
  - internal/static/js/chat/ChatView.js
-->

---
### Requirement: Target pane isolation in split view
The system SHALL route the quoted text strictly to the composer input belonging to the pane where the selection occurred.

#### Scenario: Quote text in secondary pane
- **WHEN** user selects text and invokes quote action within the secondary split pane
- **THEN** quoted text is inserted into the secondary pane composer input without altering the primary pane composer input

##### Example: Quoting in secondary split pane
- **GIVEN** primary pane has draft "Left draft" and secondary pane has draft ""
- **WHEN** user quotes "Error: 500" from the secondary pane message
- **THEN** secondary pane composer becomes "> Error: 500\n\n"
- **THEN** primary pane composer remains "Left draft"

<!-- @trace
source: selection-quote-reply
updated: 2026-10-13
code:
  - scripts/split-pane-quote.test.cjs
  - internal/static/js/chat/SelectionQuotePopover.js
  - internal/static/index.html
  - scripts/popover-position.test.cjs
  - scripts/selection-detect.test.cjs
  - scripts/quote-reply-action.test.cjs
  - scripts/textarea-autogrow.test.cjs
  - internal/static/js/core/core.js
  - scripts/selection-quote-integration.test.cjs
  - scripts/format-quote.test.cjs
  - internal/static/js/chat/ChatView.js
-->