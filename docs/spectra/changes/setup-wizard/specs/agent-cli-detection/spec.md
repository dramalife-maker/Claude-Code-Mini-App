## Purpose

Tell the user which agent command-line tools are available on this machine, so they know what to install before creating a session, using the same lookup rules the app uses when it launches them.

## ADDED Requirements

### Requirement: Detect each agent CLI with the app's own lookup rules

The system SHALL report, for each of Claude (`claude`), Codex (`codex`), Cursor (`cursor-agent`), Kiro (`kiro-cli`) and Antigravity (`agy`), whether the command can be found, its resolved path, and its version. It SHALL resolve each command with the same rules the app uses to launch it: Codex via `CODEX_BIN`, the `PATH`, then the Windows default install location; Antigravity via `CC_AGY_BIN` then the `PATH`; the others via the `PATH`. A tool not found SHALL be reported as `found: false` without raising an error.

#### Scenario: Command on the PATH

- **WHEN** an executable named `claude` is on the `PATH`
- **THEN** the result for `claude` has `found: true` and its resolved path

#### Scenario: Codex via CODEX_BIN

- **WHEN** `CODEX_BIN` points to an existing file and `codex` is not on the `PATH`
- **THEN** the result for `codex` is found with that path

#### Scenario: Antigravity override

- **WHEN** `CC_AGY_BIN` points to an executable
- **THEN** the result for `agy` is found with that path

#### Scenario: Missing tool

- **WHEN** no `kiro-cli` exists anywhere the app looks
- **THEN** the result for `kiro-cli` is `found: false` with an `install_url`, and the other results are unaffected

### Requirement: Version probing is bounded and non-fatal

The system SHALL obtain each found tool's version by running it with `--version`, SHALL run the probes concurrently, SHALL stop a probe that takes longer than 3 seconds, and SHALL start the probe processes without showing a console window on Windows. A probe that fails, times out or prints nothing SHALL leave that tool's `version` empty while keeping `found: true`, and SHALL NOT affect any other tool.

#### Scenario: Slow tool does not block the rest

- **WHEN** one tool's version command hangs
- **THEN** the overall check returns after about the 3-second limit with that tool's version empty and the other tools' versions filled in

### Requirement: Install guidance without blocking setup

Each result SHALL include an `install_url` pointing at the vendor's official instructions, and the Cursor result SHALL carry a note that the official command is `agent` while this app calls `cursor-agent`. The wizard SHALL display the results as information only: it SHALL NOT prevent saving when no tool is found, and SHALL recommend (not require) having at least one tool available before creating a session.

#### Scenario: No tool found

- **WHEN** none of the five tools is found
- **THEN** the wizard shows install links and a recommendation, and the save button still works
