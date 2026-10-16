## Purpose

Defines where the model, effort, and permission-mode selectors appear in the chat UI. They live in the composer action row beside the attachment button, so a user can adjust them next to the message being sent.

## ADDED Requirements

### Requirement: Composer action row hosts model, effort, and permission controls

The chat composer SHALL render a permission-mode selector, a model selector, and an effort selector in its action row, immediately after the attachment button, for the owner of a session in agent input mode.

#### Scenario: Controls visible in agent mode

- **WHEN** the session owner views a Claude session with the composer in agent input mode
- **THEN** the composer action row shows the permission-mode, model, and effort selectors after the attachment button

#### Scenario: Controls hidden in shell mode

- **WHEN** the composer is switched to shell input mode
- **THEN** none of the three selectors is rendered

#### Scenario: Controls hidden for guests

- **WHEN** a guest opens a shared chat
- **THEN** none of the three selectors is rendered

### Requirement: Selector availability follows agent type

The composer SHALL hide a selector that the session's agent type does not support.

#### Scenario: Cursor session has no effort selector

- **WHEN** the session agent type is `cursor`
- **THEN** the effort selector is not rendered and the model and permission selectors remain

#### Scenario: Codex or Kiro session has no permission selector

- **WHEN** the session agent type is `codex` or `kiro`
- **THEN** the permission-mode selector is not rendered

### Requirement: Selections apply on next send

Changing a selector SHALL only update the draft selection; the corresponding `set_model`, `set_effort`, or `set_mode` message MUST be sent when the next message is submitted, matching existing behavior.

#### Scenario: Change model then send

- **WHEN** the user picks a different model and then submits a message
- **THEN** the client sends `set_model` with the chosen model before the message and the session reports it as the active model

#### Scenario: Change without sending

- **WHEN** the user picks a different effort and does not submit
- **THEN** no `set_effort` message is sent

### Requirement: Selectors disabled while the agent is busy

The three selectors MUST be disabled whenever the session state is THINKING, STREAMING, AWAITING_CONFIRM, or any shell-running or shell-confirm state.

#### Scenario: Agent is streaming

- **WHEN** the session state is STREAMING
- **THEN** the permission, model, and effort selectors cannot be changed

#### Scenario: Agent is idle

- **WHEN** the session state is IDLE
- **THEN** the three selectors can be changed

### Requirement: Dangerous permission mode is visually emphasized

The permission selector SHALL use the amber danger style when the current mode is `bypassPermissions` or `yolo`.

#### Scenario: Bypass mode selected

- **WHEN** the permission mode is `bypassPermissions`
- **THEN** the permission selector renders in the amber danger style

### Requirement: Chat header no longer duplicates the controls

The chat header SHALL NOT render model, effort, or permission-mode selectors, nor a collapse toggle for them, on desktop or mobile; the repo, model label, and quota sub-line MUST remain.

#### Scenario: Desktop header

- **WHEN** a session owner views the chat on a viewport at least 640px wide
- **THEN** the header shows no model, effort, or permission selectors while the repo, model label, and quota sub-line remain

#### Scenario: Mobile header

- **WHEN** a session owner views the chat on a viewport under 640px wide
- **THEN** the header has no expand/collapse button and no permission icon chip, and the selectors are reachable in the composer without extra taps

### Requirement: Selectors in other screens are unchanged

The new-session form and the forward dialog SHALL keep their existing full-width model, effort, and permission selectors.

#### Scenario: New session form

- **WHEN** the user opens the new-session form
- **THEN** the model, effort, and permission selectors look and behave as before
