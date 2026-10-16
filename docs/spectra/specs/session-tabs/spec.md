# session-tabs Specification

## Purpose

Defines VSCode-style session tabs for the desktop chat layout: tab bars per group, tab open and close rules, drag-and-drop from the sidebar and between groups, persistence, and unchanged behavior on narrow screens.

## Requirements

### Requirement: Tab bar per group on wide screens

On viewports at least 1024px wide, each session group SHALL display a tab bar listing its open sessions, each tab showing the session name, agent icon, a status dot for running or awaiting-approval sessions, and a close control, with the active tab visually distinct.

#### Scenario: Single group shows a tab bar

- **WHEN** the user has opened sessions A and B on a wide screen with no split
- **THEN** one tab bar shows tabs A and B with the active one highlighted above the chat

#### Scenario: Tab bar overflows

- **WHEN** the open tabs are wider than the tab bar
- **THEN** the tab bar scrolls horizontally and no tab wraps to a second row

#### Scenario: Status dot on a background tab

- **WHEN** a non-active tab's session is running
- **THEN** that tab shows the running status dot


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Opening a session reuses its existing tab

Opening a session SHALL place it as a tab in the focused group, and MUST NOT create a second tab for a session that is already open in any group.

#### Scenario: Click a new session in the sidebar

- **WHEN** the user clicks session C in the sidebar and C is not open
- **THEN** a tab for C is added to the focused group and becomes active

#### Scenario: Click an already open session

- **WHEN** the user clicks session A in the sidebar and A is already open in the other group
- **THEN** the other group gains focus with A as its active tab and no new tab is created


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Only the active tab of a group is mounted

Each group SHALL mount a chat view only for its active tab, so at most two chat WebSocket connections exist at once.

#### Scenario: Switch tab in a group

- **WHEN** the user clicks an inactive tab B in a group whose active tab is A
- **THEN** the group shows session B's chat and session A's chat is unmounted


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Closing tabs

A tab SHALL be closable with its close control or a middle-click. Closing the active tab SHALL activate the tab to its right, or the tab to its left when none is to the right, and closing the last tab of a group SHALL remove that group.

#### Scenario: Close the active tab

- **WHEN** the user closes the active tab that has a tab to its right
- **THEN** the right-hand tab becomes active

#### Scenario: Close the last tab of the second group

- **WHEN** the second group has a single tab and the user closes it
- **THEN** the second group disappears and the layout shows one group

#### Scenario: Close every tab

- **WHEN** the user closes the only remaining tab
- **THEN** no session is selected and the empty state is shown


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Drag a session from the sidebar

Dragging a session from the sidebar list SHALL allow dropping it on a tab bar, on the center of a group body, or on a screen edge, with a drop preview shown while dragging.

#### Scenario: Drop on a tab bar

- **WHEN** the user drops session C between tabs A and B of a group
- **THEN** C is inserted as a tab between A and B and becomes active

#### Scenario: Drop on the center of a group body

- **WHEN** the user drops session C on the center area of a group body
- **THEN** C is added as a tab to that group and becomes active

#### Scenario: Drop on the right edge with one group

- **WHEN** only one group exists and the user drops session C on the right 25% of the group body
- **THEN** a second group is created on the right with C as its only tab and the layout orientation is horizontal

#### Scenario: Drop on the bottom edge with one group

- **WHEN** only one group exists and the user drops session C on the bottom 25% of the group body
- **THEN** a second group is created below with C as its only tab and the layout orientation is vertical

#### Scenario: Edge drop with two groups

- **WHEN** two groups exist and the user drops session C on any part of a group body
- **THEN** C is added as a tab to that group and no third group is created

#### Scenario: Drag leaves without dropping

- **WHEN** the user cancels the drag or leaves every drop target
- **THEN** the drop preview disappears and no tab changes

#### Scenario: Dropped session is unknown

- **WHEN** a drop carries a session id that is not in the session list
- **THEN** the drop is ignored


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Drag tabs to reorder and move between groups

A tab SHALL be draggable to reorder within its tab bar or to move into the other group, and a group left with no tabs SHALL be removed.

#### Scenario: Reorder within a group

- **WHEN** the user drags tab B before tab A in the same tab bar
- **THEN** the tab order becomes B then A

#### Scenario: Move the last tab out of a group

- **WHEN** the user drags the only tab of the second group into the first group
- **THEN** the tab joins the first group and the second group disappears


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Tab state persists across reloads

The application SHALL persist each group's tab order, active tab, the focused group, the orientation, and the ratio in local storage, and MUST remove tabs whose sessions no longer exist once the session list has loaded.

#### Scenario: Reload restores tabs

- **WHEN** the user reloads the page with two groups open
- **THEN** both groups return with the same tabs, active tabs, orientation, and ratio

#### Scenario: Deleted session disappears

- **WHEN** a persisted tab's session was deleted on the server
- **THEN** that tab is removed after the session list loads, and a group left empty is removed

#### Scenario: Corrupt stored state

- **WHEN** the stored tab state cannot be parsed
- **THEN** the application starts with a single empty group and shows no error


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Legacy split state migrates to tabs

When no tab state is stored but the legacy split state exists, the application SHALL build the tab groups from it, placing the legacy primary session in the first group and, if split was enabled with a secondary session, that session in a second group.

#### Scenario: Migrate an enabled split

- **WHEN** the legacy state has split enabled with a secondary session B and the primary session was A
- **THEN** group one holds A and group two holds B with the legacy orientation and ratio


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Keyboard shortcuts for groups

`Ctrl+\` SHALL move the active tab into a new second group when there is one group with at least two tabs, and SHALL merge the second group into the first when two groups exist. `Ctrl+Alt+Left` and `Ctrl+Alt+Right` SHALL move focus between groups.

#### Scenario: Split with Ctrl+backslash

- **WHEN** one group has tabs A and B with B active and the user presses `Ctrl+\`
- **THEN** B moves to a new second group and A stays in the first

#### Scenario: Merge with Ctrl+backslash

- **WHEN** two groups exist and the user presses `Ctrl+\`
- **THEN** the second group's tabs are appended to the first group and one group remains


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Narrow screens keep the current behavior

On viewports narrower than 1024px the application SHALL NOT render tab bars or accept drag-and-drop, and SHALL keep the existing single-pane behavior. The sidebar "open in secondary view" button has been removed; a second group is opened by drag-and-drop or `Ctrl+\` on wide screens, and narrow screens only ever show a single pane.

#### Scenario: Resize below the breakpoint

- **WHEN** the viewport is narrower than 1024px
- **THEN** no tab bar is shown and selecting a session opens it as before


<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->

---
### Requirement: Tab group reducer is a pure function

The tab group state transitions SHALL be implemented as a pure function of the previous state and an action, so the same inputs always produce the same output and the input state is never modified.

#### Scenario: Reducer does not mutate input

- **WHEN** an action is applied to a state object
- **THEN** the original state object is unchanged and a new state is returned

<!-- @trace
source: session-tabs
updated: 2026-10-16
code:
  - README.md
  - internal/static/js/hooks/tabGroups.js
  - internal/quota/kiro.go
  - docs/getting-started.md
  - internal/static/js/ui/ui-atoms.js
  - internal/antigravity/models.go
  - README.zh-TW.md
  - internal/ws/handler.go
  - internal/static/js/hooks/useChatSocket.js
  - scripts/tab-groups.test.cjs
  - internal/static/AGENT.md
  - .github/release-header.md
  - internal/server/server.go
  - internal/quota/codex.go
  - internal/static/js/ui/SessionTabs.js
  - internal/quota/windows.go
  - internal/static/js/core/core.js
  - .github/workflows/release.yml
  - internal/quota/antigravity.go
  - AGENTS.md
  - internal/version/version.go
  - config.example.yaml
  - internal/static/js/ui/SettingsModal.js
  - internal/quota/cursor.go
  - internal/static/js/session/SessionView.js
  - internal/quota/claude.go
  - docs/getting-started.zh-TW.md
  - internal/static/js/app.js
  - CLAUDE.md
  - internal/static/js/ui/SplitLayout.js
  - internal/quota/result.go
  - internal/static/js/chat/chat-header.js
  - internal/static/index.html
  - internal/static/js/chat/ChatView.js
tests:
  - internal/server/models_sync_test.go
  - internal/quota/windows_test.go
  - internal/version/version_test.go
  - internal/ws/queue_test.go
  - internal/antigravity/models_test.go
  - docs/spec/antigravity-cli.md
  - internal/antigravity/runner_test.go
-->