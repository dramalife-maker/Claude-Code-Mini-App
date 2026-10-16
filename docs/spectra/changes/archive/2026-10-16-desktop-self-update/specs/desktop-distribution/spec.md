## Purpose

Defines how production desktop builds carry their own web assets and how GitHub Releases deliver compatible installation bundles and a verifiable desktop update executable. It preserves disk-based development and existing server distribution.

## ADDED Requirements

### Requirement: Production embedded assets

Production desktop builds SHALL embed the application HTML, JavaScript, and favicon in the executable. The application SHALL serve the home page, share page, and desktop icon from the same embedded asset source without depending on an adjacent internal/static directory. An adjacent stale directory SHALL NOT override embedded assets. Embedded assets MUST NOT include configuration, databases, uploads, logs, Go source files, or development documentation.

#### Scenario: Run without disk frontend

- **WHEN** a production desktop executable starts with a valid configuration and without a disk frontend directory
- **THEN** the home page, JavaScript, favicon, and share page are served from the executable

#### Scenario: Ignore stale frontend files

- **WHEN** an older internal/static directory remains beside a newer production desktop executable
- **THEN** the application serves the embedded version and ignores the stale files

#### Scenario: Preserve share page behavior

- **WHEN** a visitor opens a valid share URL with or without a trailing slash
- **THEN** the page retains its relative-asset base behavior, no-store and no-referrer headers, and existing authorization restrictions

### Requirement: Development disk assets

Non-production builds SHALL read web assets from the development disk directory without requiring a frontend build step. Static serving SHALL preserve API routing, correct JavaScript content types, and rejection of paths outside the public asset root.

#### Scenario: Edit frontend during development

- **WHEN** a developer changes a disk JavaScript asset and refreshes a non-production build
- **THEN** the changed asset is served without rebuilding the Go executable

#### Scenario: Preserve API routing and asset boundaries

- **WHEN** a client requests an existing API route, an unknown API route, or a path outside the public asset root
- **THEN** the existing API route keeps its behavior, the unknown route does not become an HTML success response, and no file outside the public asset root is returned

### Requirement: Compatible release artifacts

Each supported release SHALL retain the Windows installation ZIP and Linux server tar.gz and SHALL add exactly one Windows amd64 desktop update executable named claude-miniapp-desktop-v<VERSION>-windows-amd64.exe. The Windows ZIP desktop executable SHALL contain the same bytes as that update executable. The ZIP SHALL continue to provide the server executable and its disk frontend. The Linux package SHALL retain its server executable, configuration example, and disk frontend. The updater SHALL NOT update the standalone server.

#### Scenario: Inspect release package contents

- **WHEN** release assets for version 1.0.1 are assembled
- **THEN** claude-miniapp-v1.0.1-windows-amd64.zip, claude-miniapp-v1.0.1-linux-amd64.tar.gz, and claude-miniapp-desktop-v1.0.1-windows-amd64.exe exist, and the desktop executable in the ZIP is byte-identical to the standalone desktop asset

### Requirement: Release identity and integrity

Release generation MUST reject a mismatch between the tag, root VERSION, and version.Version. Production desktop build metadata MUST expose the commit referenced by the triggering tag. The release SHALL contain checksums.txt in SHA256SUMS format covering the installation ZIP, Linux tar.gz, and standalone desktop executable.

#### Scenario: Reject inconsistent version inputs

- **WHEN** the tag is v1.0.1 but VERSION or version.Version is 1.0.0
- **THEN** release generation fails before publishing a public release

#### Scenario: Verify build identity and checksums

- **WHEN** release assets for a tag are inspected
- **THEN** the desktop reports the matching version and tag commit SHA, and every downloadable program asset has a matching SHA256 entry in checksums.txt

### Requirement: Complete release publication and bootstrap

The release pipeline SHALL upload and verify all required assets on a draft release before publishing it as a stable latest release. It SHALL reuse only an existing draft for the same tag, SHALL NOT overwrite assets of an already published release, and SHALL leave an incomplete release unpublished. Distribution documentation SHALL explain the one-time manual upgrade needed to acquire the updater, writable installation-directory requirements, retained user data, and manual recovery of the desktop executable.

#### Scenario: Asset upload fails

- **WHEN** an asset upload or checksum verification fails during release preparation
- **THEN** the release remains a draft and is not advertised as a latest stable update

#### Scenario: Retry an interrupted release

- **WHEN** the pipeline is retried for a tag that already has a draft release
- **THEN** it completes and verifies that draft before publication without creating a duplicate release

#### Scenario: Reject a published release rewrite

- **WHEN** the pipeline is rerun for a tag whose release is already public
- **THEN** it stops without replacing published assets

#### Scenario: Upgrade from a version without an updater

- **WHEN** a user follows the release documentation from a desktop version without update support
- **THEN** the instructions provide a manual upgrade that retains configuration and data and enables subsequent in-app updates

