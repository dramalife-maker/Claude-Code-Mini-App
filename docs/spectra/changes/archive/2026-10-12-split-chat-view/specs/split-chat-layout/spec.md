## Purpose

Provides responsive split-pane UI containers, a resizable divider, active pane focus indication, and automatic single-pane fallback on narrow viewports. This enables comfortable side-by-side agent supervision on desktop screens without breaking mobile usability.

## ADDED Requirements

### Requirement: Split pane container layout
The UI SHALL render two chat panes side-by-side when split view mode is enabled on viewports wider than 900px.

#### Scenario: Enable split view mode
- **WHEN** user toggles split view button in the top navigation bar or triggers the split shortcut
- **THEN** UI renders left primary and right secondary chat panes divided by a vertical separator

#### Scenario: Disable split view mode
- **WHEN** user closes the secondary pane via close button or split toggle
- **THEN** secondary pane unmounts and primary chat pane expands to fill the full content width

### Requirement: Resizable pane divider
The split view divider SHALL support mouse drag to adjust left and right pane width ratios with a minimum pane width clamp of 320px.

#### Scenario: Drag divider to resize panes
- **WHEN** user drags the vertical split divider horizontally
- **THEN** left and right pane widths adjust smoothly according to pointer position while respecting minimum 320px boundaries

#### Scenario: Double click divider to reset ratio
- **WHEN** user double clicks the vertical split divider
- **THEN** left and right pane widths reset to an equal 50/50 ratio

### Requirement: Active pane focus indication
The UI SHALL highlight the currently active pane and route keyboard shortcuts and quick switcher commands to the focused pane.

#### Scenario: Click pane to switch focus
- **WHEN** user clicks inside the secondary pane
- **THEN** secondary pane receives active focus highlight and primary pane border becomes inactive

#### Scenario: Keyboard shortcut pane toggle
- **WHEN** user presses `Ctrl+Alt+Left` or `Ctrl+Alt+Right`
- **THEN** focus shifts between the primary and secondary panes respectively

### Requirement: Responsive narrow viewport fallback
The UI SHALL automatically collapse split view into a single pane with a tab switcher when viewport width drops below 900px or when running inside Telegram Mini App.

#### Scenario: Viewport resize below threshold
- **WHEN** window width resizes below 900px while split view is enabled
- **THEN** UI displays a single pane with top tabs to switch between the two sessions instead of side-by-side columns

#### Scenario: Viewport resize restores side-by-side
- **WHEN** window width resizes back above 900px
- **THEN** UI restores side-by-side dual pane layout automatically
