## MODIFIED Requirements

### Requirement: Unread message count calculation

The system SHALL compute `unread_count` for each session representing the total number of non-user messages created after `last_read_at`. Telegram sessions SHALL always report an `unread_count` of 0, because their messages are not stored locally.

#### Scenario: Accurate count of unread incoming messages

- **WHEN** a session receives new assistant, shell, or agent messages after `last_read_at`
- **THEN** the session query SHALL return `unread_count` matching the exact number of incoming messages

#### Scenario: User messages excluded from unread count

- **WHEN** a user posts an outgoing prompt
- **THEN** the system SHALL NOT count user-authored messages toward `unread_count`

#### Scenario: Telegram sessions report no unread count

- **WHEN** a Telegram session is listed
- **THEN** its `unread_count` SHALL be 0 regardless of incoming Telegram messages
