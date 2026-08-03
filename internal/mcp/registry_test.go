package mcp

import "testing"

func TestNormalizeState(t *testing.T) {
	cases := map[string]string{
		"IDLE":                    StateIdle,
		"SHELL_IDLE":              StateIdle,
		"AWAITING_CONFIRM":        StateAwaitingPermission,
		"AWAITING_SHELL_CONFIRM":  StateShellPending,
		"SHELL_AWAITING_APPROVAL": StateShellPending,
		"THINKING":                StateRunning,
		"STREAMING":               StateRunning,
		"SHELL_RUNNING":           StateRunning,
	}
	for in, want := range cases {
		if got := normalizeState(in); got != want {
			t.Errorf("normalizeState(%q) = %q, want %q", in, got, want)
		}
	}
}
