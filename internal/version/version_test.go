package version

import (
	"runtime/debug"
	"testing"
)

func TestRevision(t *testing.T) {
	const sha = "7fcc9a16665e4b710d5244fe5e62b07cc5082e3b"
	for _, tt := range []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{"no build info", nil, "unknown"},
		{"no VCS info", &debug.BuildInfo{}, "unknown"},
		{"empty revision", &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.revision"}}}, "unknown"},
		{"revision", &debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "vcs.time", Value: "2026-10-14"}, {Key: "vcs.revision", Value: sha}}}, sha},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := revision(tt.info); got != tt.want {
				t.Fatalf("revision() = %q, want %q", got, tt.want)
			}
		})
	}
}
