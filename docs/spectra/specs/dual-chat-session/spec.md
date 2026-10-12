# dual-chat-session Specification

## Purpose

Manages independent session selection, concurrent dual WebSocket streams, isolated tool approvals, and layout persistence for split view. This ensures multiple agents run concurrently without state conflicts or race conditions.

## Requirements

### Requirement: Independent session selection per pane
Each pane SHALL maintain its own selected session identifier and independent conversation view.

#### Scenario: Select different session in secondary pane
- **WHEN** user chooses session B in secondary pane while primary pane displays session A
- **THEN** secondary pane loads session B while primary pane continues displaying session A

#### Scenario: Quick switcher target focused pane
- **WHEN** user triggers Ctrl+P quick switcher while secondary pane has active focus
- **THEN** selected session opens inside the secondary pane without changing primary pane session


<!-- @trace
source: split-chat-view
updated: 2026-10-12
code:
  - CLAUDE.md
  - docs/plan/done.md
  - internal/static/js/chat/ChatView.js
  - internal/static/js/ui/SplitLayout.js
  - internal/static/js/app.js
  - internal/static/index.html
  - .agents/skills/spectra-propose/SKILL.md
  - .cursorrules
  - .agents/skills/spectra-verify/SKILL.md
  - .agents/skills/spectra-discuss/SKILL.md
  - internal/static/js/chat/chat-header.js
  - .spectra.yaml
  - internal/static/js/session/SessionView.js
  - README.md
  - README.zh-TW.md
  - .agents/skills/spectra-audit/SKILL.md
  - .agents/skills/spectra-drift/SKILL.md
  - .agents/skills/spectra-archive/SKILL.md
  - .agents/skills/spectra-analyze/SKILL.md
  - .agents/skills/spectra-debug/SKILL.md
  - .agents/skills/spectra-review/SKILL.md
  - .agents/skills/spectra-commit/SKILL.md
  - .agents/skills/spectra-apply/SKILL.md
  - .agents/skills/spectra-ingest/SKILL.md
-->

---
### Requirement: Concurrent WebSocket streaming
Both panes SHALL establish and maintain concurrent WebSocket connections to their respective sessions without cross-talk or race conditions.

#### Scenario: Simultaneous streaming in both panes
- **WHEN** tasks run concurrently in both primary and secondary sessions
- **THEN** both panes stream token deltas and tool events into their respective message lists simultaneously

#### Scenario: Disconnect handling per pane
- **WHEN** network connectivity drops for one session's WebSocket
- **THEN** only that specific pane displays reconnecting status while the other pane continues operating normally


<!-- @trace
source: split-chat-view
updated: 2026-10-12
code:
  - CLAUDE.md
  - docs/plan/done.md
  - internal/static/js/chat/ChatView.js
  - internal/static/js/ui/SplitLayout.js
  - internal/static/js/app.js
  - internal/static/index.html
  - .agents/skills/spectra-propose/SKILL.md
  - .cursorrules
  - .agents/skills/spectra-verify/SKILL.md
  - .agents/skills/spectra-discuss/SKILL.md
  - internal/static/js/chat/chat-header.js
  - .spectra.yaml
  - internal/static/js/session/SessionView.js
  - README.md
  - README.zh-TW.md
  - .agents/skills/spectra-audit/SKILL.md
  - .agents/skills/spectra-drift/SKILL.md
  - .agents/skills/spectra-archive/SKILL.md
  - .agents/skills/spectra-analyze/SKILL.md
  - .agents/skills/spectra-debug/SKILL.md
  - .agents/skills/spectra-review/SKILL.md
  - .agents/skills/spectra-commit/SKILL.md
  - .agents/skills/spectra-apply/SKILL.md
  - .agents/skills/spectra-ingest/SKILL.md
-->

---
### Requirement: Independent prompt submission and tool approval
Each pane SHALL provide its own input composer, stop button, and permission approval dialogs without blocking the other pane.

#### Scenario: Tool approval in one pane while typing in other
- **WHEN** secondary session requests tool permission
- **THEN** approval dialog appears exclusively over the secondary pane while primary pane remains interactive

#### Scenario: Stop task in single pane
- **WHEN** user clicks stop button on primary pane
- **THEN** only the primary session task aborts while secondary session execution continues uninterrupted


<!-- @trace
source: split-chat-view
updated: 2026-10-12
code:
  - CLAUDE.md
  - docs/plan/done.md
  - internal/static/js/chat/ChatView.js
  - internal/static/js/ui/SplitLayout.js
  - internal/static/js/app.js
  - internal/static/index.html
  - .agents/skills/spectra-propose/SKILL.md
  - .cursorrules
  - .agents/skills/spectra-verify/SKILL.md
  - .agents/skills/spectra-discuss/SKILL.md
  - internal/static/js/chat/chat-header.js
  - .spectra.yaml
  - internal/static/js/session/SessionView.js
  - README.md
  - README.zh-TW.md
  - .agents/skills/spectra-audit/SKILL.md
  - .agents/skills/spectra-drift/SKILL.md
  - .agents/skills/spectra-archive/SKILL.md
  - .agents/skills/spectra-analyze/SKILL.md
  - .agents/skills/spectra-debug/SKILL.md
  - .agents/skills/spectra-review/SKILL.md
  - .agents/skills/spectra-commit/SKILL.md
  - .agents/skills/spectra-apply/SKILL.md
  - .agents/skills/spectra-ingest/SKILL.md
-->

---
### Requirement: Split view persistence
The application SHALL persist split view toggle state, pane ratio, and secondary session ID in local storage across browser reloads.

#### Scenario: Page reload restores split state
- **WHEN** user reloads the browser while split view is open
- **THEN** split view restores with the previously opened secondary session and saved pane ratio

#### Scenario: Auto clear invalid secondary session
- **WHEN** persisted secondary session was deleted on server
- **THEN** application automatically clears secondary pane selection and reverts to single pane mode gracefully

<!-- @trace
source: split-chat-view
updated: 2026-10-12
code:
  - CLAUDE.md
  - docs/plan/done.md
  - internal/static/js/chat/ChatView.js
  - internal/static/js/ui/SplitLayout.js
  - internal/static/js/app.js
  - internal/static/index.html
  - .agents/skills/spectra-propose/SKILL.md
  - .cursorrules
  - .agents/skills/spectra-verify/SKILL.md
  - .agents/skills/spectra-discuss/SKILL.md
  - internal/static/js/chat/chat-header.js
  - .spectra.yaml
  - internal/static/js/session/SessionView.js
  - README.md
  - README.zh-TW.md
  - .agents/skills/spectra-audit/SKILL.md
  - .agents/skills/spectra-drift/SKILL.md
  - .agents/skills/spectra-archive/SKILL.md
  - .agents/skills/spectra-analyze/SKILL.md
  - .agents/skills/spectra-debug/SKILL.md
  - .agents/skills/spectra-review/SKILL.md
  - .agents/skills/spectra-commit/SKILL.md
  - .agents/skills/spectra-apply/SKILL.md
  - .agents/skills/spectra-ingest/SKILL.md
-->