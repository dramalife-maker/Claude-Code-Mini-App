package kiroacp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAgentConfig_MarkdownFrontmatter(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".kiro", "agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(agentsDir, "golang-pro.md"), `---
name: golang-pro
includeMcpJson: true
---
You are a Go expert.
`)
	cfg, path, err := loadAgentConfig(dir, "golang-pro")
	if err != nil {
		t.Fatal(err)
	}
	if path == "" || cfg.Name != "golang-pro" {
		t.Fatalf("cfg=%+v path=%q", cfg, path)
	}
	if cfg.IncludeMcpJson == nil || !*cfg.IncludeMcpJson {
		t.Fatalf("includeMcpJson=%v", cfg.IncludeMcpJson)
	}
}

func TestShouldIncludeMcpJSON_Defaults(t *testing.T) {
	if !shouldIncludeMcpJSON("", nil) {
		t.Fatal("no agent should include json")
	}
	if !shouldIncludeMcpJSON("solo", &kiroAgentConfig{}) {
		t.Fatal("omitted includeMcpJson should default true")
	}
	falseVal := false
	if shouldIncludeMcpJSON("solo", &kiroAgentConfig{IncludeMcpJson: &falseVal}) {
		t.Fatal("explicit false should exclude json")
	}
}
