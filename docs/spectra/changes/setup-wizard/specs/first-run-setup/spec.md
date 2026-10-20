## Purpose

Let a person who has just downloaded the app finish configuration from the screen on the first launch, without hand-editing `config.yaml`, while keeping the unauthenticated setup surface from being taken over by other machines or web pages.

## ADDED Requirements

### Requirement: Setup mode only when the config file is missing

The system SHALL enter setup mode only when `config.yaml` does not exist in the working directory. A `config.yaml` that exists but cannot be read or parsed SHALL keep failing startup with its error and SHALL NOT enter setup mode.

#### Scenario: Missing config file starts setup mode

- **WHEN** the application starts and no `config.yaml` exists
- **THEN** it starts in setup mode and serves the setup wizard instead of exiting with an error

#### Scenario: Broken config file is not treated as first run

- **WHEN** `config.yaml` exists but contains invalid YAML
- **THEN** startup fails with the parse error and the setup wizard is not shown

### Requirement: Setup mode exposes a minimal surface

In setup mode the system SHALL serve only the public front-end assets and the three setup endpoints (`GET /setup/status`, `GET /setup/cli-check`, `POST /setup/save`). It SHALL NOT open the database, start agent services or MCP, or register any other API. Every other path SHALL respond 503 with code `setup_required`.

#### Scenario: Other APIs are unavailable

- **WHEN** a client requests `GET /sessions` while in setup mode
- **THEN** the response is 503 with code `setup_required`

#### Scenario: Normal mode hides the setup endpoints

- **WHEN** a config file exists and the application runs in normal mode
- **THEN** `GET /setup/status` returns `{"needs_setup": false}` and `GET /setup/cli-check` and `POST /setup/save` return 404

### Requirement: Setup API is restricted to the local machine with a one-time token

In setup mode the system SHALL accept setup API requests only when the connection's actual remote address is a loopback address, the `Host` header is `127.0.0.1`, `localhost` or `[::1]` followed by the listening port, and, for `/setup/cli-check` and `/setup/save`, the `X-Setup-Token` header equals a random 32-byte hexadecimal token generated at startup. The token comparison SHALL be constant-time. Forwarded-address headers SHALL be ignored. Rejected requests SHALL receive 403 with code `setup_forbidden`.

#### Scenario: Remote machine is rejected

- **WHEN** a request to `/setup/save` comes from a non-loopback address, even with a valid token
- **THEN** the response is 403 with code `setup_forbidden` and no file is written

#### Scenario: Spoofed forwarding header does not grant access

- **WHEN** a non-loopback client sends `X-Forwarded-For: 127.0.0.1`
- **THEN** the request is still rejected with 403

#### Scenario: Rebinding Host is rejected

- **WHEN** a loopback connection carries `Host: attacker.example:8080`
- **THEN** the response is 403 with code `setup_forbidden`

#### Scenario: Missing or wrong token is rejected

- **WHEN** `/setup/save` is called from loopback without `X-Setup-Token`, or with a different value
- **THEN** the response is 403 with code `setup_forbidden` and no file is written

#### Scenario: Token reaches the user without a manual copy step

- **WHEN** setup mode starts
- **THEN** the log contains a setup URL with the token in its query string, the desktop window opens that same URL, and the front end removes the token from the address bar after reading it

### Requirement: Telegram Mini App is optional

The system SHALL start with an empty `bot_token` when `no_auth` is false and `web.password` is valid (non-empty, not `change-me`, at least 8 characters). Otherwise it SHALL fail startup with a message naming the missing requirement. When `bot_token` is empty the system SHALL ignore Telegram `initData` entirely (header and query parameter) and SHALL NOT verify it, and Telegram notifications SHALL be skipped.

#### Scenario: Password-only configuration starts

- **WHEN** `config.yaml` has `web.password` set to a valid password and no `bot_token`
- **THEN** the application starts and password login works

#### Scenario: No credentials at all fails

- **WHEN** `config.yaml` has neither `bot_token` nor a valid `web.password` and `no_auth` is false
- **THEN** startup fails with a message that either a bot token or a web password is required

#### Scenario: Forged initData cannot authenticate without a bot token

- **WHEN** `bot_token` is empty, the whitelist contains Telegram ID 123456789, and a request carries `initData` signed with the empty-token key for user 123456789
- **THEN** the request is treated as unauthenticated and does not gain access

### Requirement: Setup input validation

`POST /setup/save` SHALL validate its input before writing anything and SHALL respond 400 with code `invalid_input` and the offending `field` for: a password shorter than 8 characters, equal to `change-me`, or only whitespace; a port outside 1 to 65535; a database path whose directory does not exist or is not writable; Mini App enabled with an empty `bot_token`; or a whitelist entry that is not a positive integer.

#### Scenario: Example password is rejected

- **WHEN** the password is `change-me`
- **THEN** the response is 400 with `field: "password"` and no file is written

#### Scenario: Mini App enabled requires a bot token

- **WHEN** `miniapp.enabled` is true and `miniapp.bot_token` is empty
- **THEN** the response is 400 with `field: "miniapp.bot_token"`

#### Scenario: Unwritable database directory

- **WHEN** `db_path` points into a directory that does not exist
- **THEN** the response is 400 with `field: "db_path"`

### Requirement: Config file is written minimally, atomically, and never overwritten

The system SHALL write `config.yaml` containing `web.password`, `web.allowed_cidrs`, `server.port`, `db.path` and `no_auth: false`, plus `bot_token` and `whitelist_tg_ids` only when the Mini App option is enabled. `web.allowed_cidrs` SHALL contain only `127.0.0.0/8` and `::1/128` unless the user chose to allow LAN devices, in which case the three private IPv4 ranges are added. The file SHALL appear atomically (never partially written), SHALL fail if a `config.yaml` already exists without altering it, and SHALL have permission 0600 on Unix. The written file SHALL load successfully through the normal configuration loader.

#### Scenario: Mini App off writes no Telegram fields

- **WHEN** the user saves with the Mini App option off
- **THEN** the written file has no `bot_token` and no `whitelist_tg_ids`, and loads as valid

#### Scenario: Existing config is never overwritten

- **WHEN** `config.yaml` appears between the wizard opening and saving
- **THEN** the save responds 409 with code `already_configured` and the existing file is unchanged

#### Scenario: Concurrent saves

- **WHEN** two valid saves race
- **THEN** exactly one succeeds and the file content equals the successful request's content

### Requirement: Switch to normal mode after saving

After a successful save the system SHALL, when the configured port equals the current port, stop the setup listener and start the normal server on that port within the same process, keeping the same server handle valid, and respond `applied: "switched"`. When the port differs it SHALL respond `applied: "restart_required"` with the new URL and SHALL NOT switch. If starting normal mode fails, the response SHALL be 500 with code `start_failed` and the error, and the saved file SHALL remain.

#### Scenario: Same port switches in place

- **WHEN** the user saves with the port unchanged
- **THEN** the response is `applied: "switched"` and, after the front end reloads, password login with the new password succeeds without restarting the program

#### Scenario: Changed port needs a restart

- **WHEN** the user saves with a different port
- **THEN** the response is `applied: "restart_required"` with the new URL and the program keeps running in setup mode until restarted

### Requirement: Desktop first run writes next to the executable

When no `config.yaml` exists in the working directory and a `config.example.yaml` exists next to the desktop executable, the desktop application SHALL switch its working directory to the executable's directory before loading configuration, so the wizard writes `config.yaml` there. Otherwise the working directory SHALL be unchanged.

#### Scenario: Launched from a shortcut

- **WHEN** the packaged desktop executable is launched with a different working directory and has `config.example.yaml` beside it
- **THEN** the saved `config.yaml` is created beside the executable

### Requirement: Guided wizard flow

The wizard SHALL present, in order: a welcome step with the local CLI check results; the login password (with a generate-random-password action), the port and the database path, and an option to allow LAN devices to log in (off by default); the Telegram Mini App switch (off by default); and a review-and-save step. The switch, when off, SHALL hide the bot token and whitelist fields. When on, it SHALL show those fields and explain that an externally reachable HTTPS address (for example a tunnel) is needed and that the wizard does not create it. The wizard SHALL NOT offer a no-authentication option.

#### Scenario: Mini App off needs only a password

- **WHEN** the user keeps the Mini App switch off and enters a valid password
- **THEN** the save button is enabled without any bot token or whitelist input

#### Scenario: Generated password satisfies the rules

- **WHEN** the user presses the generate-password action
- **THEN** the field is filled with a random password that passes the password validation
