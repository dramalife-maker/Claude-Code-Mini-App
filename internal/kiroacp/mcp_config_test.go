package kiroacp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMcpServersForACP_WorkspaceOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	globalDir := filepath.Join(dir, "home", ".kiro", "settings")
	workspaceDir := filepath.Join(dir, "proj", ".kiro", "settings")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(globalDir, "mcp.json"), `{
  "mcpServers": {
    "miniapp": {
      "url": "http://global.example/mcp",
      "headers": {"Authorization": "Bearer global"}
    },
    "windows-mcp": {
      "command": "uvx-global",
      "args": ["windows-mcp", "serve"]
    }
  }
}`)
	writeFile(t, filepath.Join(workspaceDir, "mcp.json"), `{
  "mcpServers": {
    "miniapp": {
      "url": "http://workspace.example/mcp"
    },
    "chrome-devtools-mcp": {
      "command": "npx",
      "args": ["-y", "@google/chrome-devtools-mcp@latest"]
    }
  }
}`)

	t.Setenv("USERPROFILE", filepath.Join(dir, "home"))
	servers := mcpServersForACP(filepath.Join(dir, "proj"))
	if len(servers) != 3 {
		t.Fatalf("want 3 servers, got %d: %+v", len(servers), servers)
	}

	byName := map[string]map[string]any{}
	for _, raw := range servers {
		srv, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected type %T", raw)
		}
		byName[srv["name"].(string)] = srv
	}

	if byName["miniapp"]["url"] != "http://workspace.example/mcp" {
		t.Fatalf("workspace should override global miniapp url: %+v", byName["miniapp"])
	}
	if byName["windows-mcp"]["command"] != "uvx-global" {
		t.Fatalf("global windows-mcp should remain: %+v", byName["windows-mcp"])
	}
	if byName["chrome-devtools-mcp"]["command"] != "npx" {
		t.Fatalf("workspace chrome-devtools missing: %+v", byName["chrome-devtools-mcp"])
	}
}

func TestMcpServersForACP_SkipsDisabled(t *testing.T) {
	dir := t.TempDir()
	workspaceDir := filepath.Join(dir, "proj", ".kiro", "settings")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(workspaceDir, "mcp.json"), `{
  "mcpServers": {
    "off": {"command": "echo", "args": ["off"], "disabled": true},
    "on": {"command": "echo", "args": ["on"]}
  }
}`)
	t.Setenv("USERPROFILE", filepath.Join(dir, "missing-home"))

	servers := mcpServersForACP(filepath.Join(dir, "proj"))
	if len(servers) != 1 {
		t.Fatalf("want 1 server, got %d: %+v", len(servers), servers)
	}
	srv := servers[0].(map[string]any)
	if srv["name"] != "on" {
		t.Fatalf("got %+v", srv)
	}
}

func TestMcpServersForACP_FallbackWhenMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", filepath.Join(dir, "missing-home"))

	servers := mcpServersForACP(filepath.Join(dir, "proj"))
	if len(servers) != 1 {
		t.Fatalf("want fallback server, got %+v", servers)
	}
	srv := servers[0].(map[string]any)
	if srv["name"] != "windows-mcp" {
		t.Fatalf("got %+v", srv)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
