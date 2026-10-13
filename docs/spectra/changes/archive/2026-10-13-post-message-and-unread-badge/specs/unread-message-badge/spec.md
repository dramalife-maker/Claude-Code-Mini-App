## Purpose

Provides a chat-application-style unread message counter badge on session cards and sidebars, displaying the exact count of new incoming messages since the session was last read.

## ADDED Requirements

### Requirement: Unread message count calculation
The system SHALL compute `unread_count` for each session representing the total number of non-user messages created after `last_read_at`.

#### Scenario: Accurate count of unread incoming messages
- **WHEN** a session receives new assistant, shell, or agent messages after `last_read_at`
- **THEN** the session query SHALL return `unread_count` matching the exact number of incoming messages

#### Scenario: User messages excluded from unread count
- **WHEN** a user posts an outgoing prompt
- **THEN** the system SHALL NOT count user-authored messages toward `unread_count`

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
