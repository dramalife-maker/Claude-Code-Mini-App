## Purpose

Allows external agents to actively push notifications, progress updates, and task results directly into a target session chat history and UI without triggering the target session's agent CLI execution.

## ADDED Requirements

### Requirement: Post message tool execution
The MCP server SHALL provide a tool named `post_message` that accepts `session_id`, `text`, optional `role` (defaulting to "assistant"), optional `author`, and optional `notify` (defaulting to false).

#### Scenario: Post message to idle session
- **WHEN** an external agent invokes `post_message` with a valid `session_id` and non-empty `text`
- **THEN** the system SHALL store the message in the database with status `done`, broadcast the message to WebSocket subscribers, and return `{ ok: true }` without spawning any agent CLI process

#### Scenario: Post message while target session is running
- **WHEN** an external agent invokes `post_message` while the target session has a task running
- **THEN** the system SHALL immediately persist and broadcast the message without interrupting, canceling, or altering the ongoing task execution

#### Scenario: Telegram push notification dispatch
- **WHEN** an external agent invokes `post_message` with `notify: true` and the session has a bound Telegram user
- **THEN** the system SHALL dispatch a notification message containing the text content to the user via Telegram Bot

#### Scenario: Invalid session rejection
- **WHEN** an external agent invokes `post_message` with a non-existent `session_id`
- **THEN** the system SHALL return an error indicating that the target session was not found
