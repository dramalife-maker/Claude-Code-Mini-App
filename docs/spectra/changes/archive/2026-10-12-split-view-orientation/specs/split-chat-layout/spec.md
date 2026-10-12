## MODIFIED Requirements

### Requirement: Split pane container layout
The UI SHALL render two chat panes either side-by-side (horizontal orientation) or stacked top-and-bottom (vertical orientation) according to the active split orientation setting.

#### Scenario: Enable horizontal split view
- **WHEN** user enables split view with horizontal orientation
- **THEN** UI renders left primary and right secondary chat panes divided by a vertical separator

#### Scenario: Enable vertical split view
- **WHEN** user enables split view with vertical orientation
- **THEN** UI renders top primary and bottom secondary chat panes divided by a horizontal separator

#### Scenario: Disable split view mode
- **WHEN** user closes the secondary pane via close button or split toggle
- **THEN** secondary pane unmounts and primary chat pane expands to fill the full content area

### Requirement: Resizable pane divider
The split view divider SHALL support mouse and touch drag to adjust pane dimensions according to the active orientation, enforcing a minimum width of 320px in horizontal mode and a minimum height of 200px in vertical mode.

#### Scenario: Drag divider in horizontal orientation
- **WHEN** user drags the vertical split divider horizontally
- **THEN** left and right pane widths adjust smoothly according to pointer position while respecting minimum 320px width boundaries

#### Scenario: Drag divider in vertical orientation
- **WHEN** user drags the horizontal split divider vertically
- **THEN** top and bottom pane heights adjust smoothly according to pointer position while respecting minimum 200px height boundaries

#### Scenario: Double click divider to reset ratio
- **WHEN** user double clicks the split divider in either orientation
- **THEN** pane dimensions reset to an equal 50/50 ratio

## ADDED Requirements

### Requirement: Split orientation switching
The UI SHALL provide intuitive orientation toggle controls to switch between horizontal (side-by-side) and vertical (top-bottom) split view modes.

#### Scenario: Toggle orientation via header button
- **WHEN** user clicks the orientation toggle action in the split view controls
- **THEN** split layout switches between side-by-side and top-bottom while preserving the active sessions in each pane

#### Scenario: Toggle orientation via keyboard shortcut
- **WHEN** user presses `Ctrl+Alt+O`
- **THEN** split layout toggles between horizontal and vertical orientations immediately

### Requirement: Split orientation persistence
The UI SHALL persist the split orientation setting in local storage and restore it upon page reload.

#### Scenario: Restore orientation on page load
- **WHEN** user reloads the application after configuring vertical split view
- **THEN** UI restores vertical split layout with top and bottom panes without reverting to horizontal default
