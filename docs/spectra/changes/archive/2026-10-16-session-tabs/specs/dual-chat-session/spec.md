## MODIFIED Requirements

### Requirement: Independent session selection per pane
Each group SHALL maintain its own tab list and active tab, and show the independent conversation view of its active tab.

#### Scenario: Select different session in secondary pane
- **WHEN** user chooses session B in the second group while the first group displays session A
- **THEN** the second group loads session B while the first group continues displaying session A

#### Scenario: Quick switcher target focused pane
- **WHEN** user triggers Ctrl+P quick switcher while the second group has active focus
- **THEN** the selected session opens as a tab in the second group without changing the first group's active tab

### Requirement: Split view persistence
The application SHALL persist the tab groups, each group's active tab, the focused group, pane ratio, and orientation in local storage across browser reloads.

#### Scenario: Page reload restores split state
- **WHEN** user reloads the browser while two groups are open
- **THEN** both groups restore with their previous tabs and active tabs and the saved pane ratio

#### Scenario: Auto clear invalid secondary session
- **WHEN** a persisted tab's session was deleted on server
- **THEN** application removes that tab, and if the second group is left empty it reverts to single group mode gracefully
