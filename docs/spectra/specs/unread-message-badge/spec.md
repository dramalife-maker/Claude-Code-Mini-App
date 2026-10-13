# unread-message-badge Specification

## Purpose

Provides a chat-application-style unread message counter badge on session cards and sidebars, displaying the exact count of new incoming messages since the session was last read.

## Requirements

### Requirement: Unread message count calculation
The system SHALL compute `unread_count` for each session representing the total number of non-user messages created after `last_read_at`.

#### Scenario: Accurate count of unread incoming messages
- **WHEN** a session receives new assistant, shell, or agent messages after `last_read_at`
- **THEN** the session query SHALL return `unread_count` matching the exact number of incoming messages

#### Scenario: User messages excluded from unread count
- **WHEN** a user posts an outgoing prompt
- **THEN** the system SHALL NOT count user-authored messages toward `unread_count`


<!-- @trace
source: post-message-and-unread-badge
updated: 2026-10-13
code:
  - scripts/selection-detect.test.cjs
  - internal/static/js/chat/SelectionQuotePopover.js
  - scripts/split-pane-quote.test.cjs
  - internal/mcp/server.go
  - internal/static/index.html
  - internal/mcp/tools.go
  - internal/server/server.go
  - internal/static/js/core/core.js
  - scripts/popover-position.test.cjs
  - internal/db/message.go
  - internal/ws/handler.go
  - scripts/quote-reply-action.test.cjs
  - scripts/format-quote.test.cjs
  - internal/static/js/chat/ChatView.js
  - scripts/selection-quote-integration.test.cjs
  - internal/ws/broadcast.go
  - internal/db/session.go
  - scripts/textarea-autogrow.test.cjs
  - internal/static/js/session/SessionView.js
tests:
  - internal/mcp/post_message_test.go
  - internal/db/unread_smoke_test.go
  - internal/api/session_test.go
-->

---
### Requirement: Unread count badge rendering in UI
The frontend SHALL display a pill-shaped unread badge on each session card and sidebar icon when `unread_count > 0`.

#### Scenario: Display exact numeric badge
- **WHEN** a session has an `unread_count` between 1 and 99
- **THEN** the UI SHALL render the numeric badge showing that exact number

#### Scenario: Display overflow badge
- **WHEN** a session has an `unread_count` greater than 99
- **THEN** the UI SHALL render `99+` inside the badge

#### Scenario: Optimistic clear on entering session
- **WHEN** the user navigates into or clicks a session with unread messages
- **THEN** the UI SHALL optimistically hide the badge and reset `unread_count` to zero immediately

<!-- @trace
source: post-message-and-unread-badge
updated: 2026-10-13
code:
  - scripts/selection-detect.test.cjs
  - internal/static/js/chat/SelectionQuotePopover.js
  - scripts/split-pane-quote.test.cjs
  - internal/mcp/server.go
  - internal/static/index.html
  - internal/mcp/tools.go
  - internal/server/server.go
  - internal/static/js/core/core.js
  - scripts/popover-position.test.cjs
  - internal/db/message.go
  - internal/ws/handler.go
  - scripts/quote-reply-action.test.cjs
  - scripts/format-quote.test.cjs
  - internal/static/js/chat/ChatView.js
  - scripts/selection-quote-integration.test.cjs
  - internal/ws/broadcast.go
  - internal/db/session.go
  - scripts/textarea-autogrow.test.cjs
  - internal/static/js/session/SessionView.js
tests:
  - internal/mcp/post_message_test.go
  - internal/db/unread_smoke_test.go
  - internal/api/session_test.go
-->