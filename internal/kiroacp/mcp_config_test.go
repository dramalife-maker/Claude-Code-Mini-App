package kiroacp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildACPMcpServers_WorkspaceOverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	globalDir := filepath.Join(dir, "home", ".kiro", "settings")
	workspaceDir := filepath.Join(dir, "proj", ".kiro", "settings")
	mustMkdirAll(t, globalDir, workspaceDir)

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
	servers, meta, err := buildACPMcpServers(filepath.Join(dir, "proj"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 3 {
		t.Fatalf("want 3 servers, got %d: %+v", len(servers), servers)
	}
	if len(meta.JSONSources) != 2 {
		t.Fatalf("json sources=%v", meta.JSONSources)
	}

	byName := indexServers(t, servers)
	if byName["miniapp"]["url"] != "http://workspace.example/mcp" {
		t.Fatalf("workspace should override global miniapp url: %+v", byName["miniapp"])
	}
	if byName["windows-mcp"]["command"] != "uvx-global" {
		t.Fatalf("global windows-mcp should remain: %+v", byName["windows-mcp"])
	}
	// kiro-cli acp 的 http mcpServers.headers 需為 [{name,value}] 陣列；
	// 傳物件會讓子進程在 session/new 時無聲崩潰（stdout 直接關閉，無 stderr）。
	headers, ok := byName["miniapp"]["headers"].([]any)
	if !ok {
		t.Fatalf("miniapp headers 應為陣列，got %T: %+v", byName["miniapp"]["headers"], byName["miniapp"])
	}
	if len(headers) != 0 {
		t.Fatalf("workspace miniapp 未設定 headers，應為空陣列: %+v", headers)
	}
}

func TestBuildACPMcpServers_SkipsDisabled(t *testing.T) {
	dir := t.TempDir()
	workspaceDir := filepath.Join(dir, "proj", ".kiro", "settings")
	mustMkdirAll(t, workspaceDir)
	writeFile(t, filepath.Join(workspaceDir, "mcp.json"), `{
  "mcpServers": {
    "off": {"command": "echo", "args": ["off"], "disabled": true},
    "on": {"command": "echo", "args": ["on"]}
  }
}`)
	t.Setenv("USERPROFILE", filepath.Join(dir, "missing-home"))

	servers, _, err := buildACPMcpServers(filepath.Join(dir, "proj"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 {
		t.Fatalf("want 1 server, got %d: %+v", len(servers), servers)
	}
	if servers[0].(map[string]any)["name"] != "on" {
		t.Fatalf("got %+v", servers[0])
	}
}

func TestBuildACPMcpServers_EmptyWhenNoConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", filepath.Join(dir, "missing-home"))

	servers, meta, err := buildACPMcpServers(filepath.Join(dir, "proj"), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 0 {
		t.Fatalf("want no fallback servers, got %+v", servers)
	}
	if len(meta.ServerNames) != 0 {
		t.Fatalf("meta=%+v", meta)
	}
}

func TestBuildACPMcpServers_AgentOverridesJSON(t *testing.T) {
	dir := t.TempDir()
	workspaceDir := filepath.Join(dir, "proj", ".kiro")
	mustMkdirAll(t, filepath.Join(workspaceDir, "settings"), filepath.Join(workspaceDir, "agents"))
	writeFile(t, filepath.Join(workspaceDir, "settings", "mcp.json"), `{
  "mcpServers": {
    "shared": {"command": "from-json", "args": []}
  }
}`)
	writeFile(t, filepath.Join(workspaceDir, "agents", "myagent.json"), `{
  "name": "myagent",
  "includeMcpJson": true,
  "mcpServers": {
    "shared": {"command": "from-agent", "args": []},
    "agent-only": {"command": "agent", "args": []}
  }
}`)
	t.Setenv("USERPROFILE", filepath.Join(dir, "missing-home"))

	servers, meta, err := buildACPMcpServers(filepath.Join(dir, "proj"), "myagent")
	if err != nil {
		t.Fatal(err)
	}
	if meta.AgentSource == "" {
		t.Fatal("expected agent source path")
	}
	byName := indexServers(t, servers)
	if byName["shared"]["command"] != "from-agent" {
		t.Fatalf("agent should override json: %+v", byName["shared"])
	}
	if _, ok := byName["agent-only"]; !ok {
		t.Fatalf("missing agent-only: %+v", byName)
	}
}

func TestBuildACPMcpServers_AgentExcludeMcpJSON(t *testing.T) {
	dir := t.TempDir()
	workspaceDir := filepath.Join(dir, "proj", ".kiro")
	mustMkdirAll(t, filepath.Join(workspaceDir, "settings"), filepath.Join(workspaceDir, "agents"))
	writeFile(t, filepath.Join(workspaceDir, "settings", "mcp.json"), `{
  "mcpServers": {
    "json-only": {"command": "json", "args": []}
  }
}`)
	writeFile(t, filepath.Join(workspaceDir, "agents", "solo.json"), `{
  "name": "solo",
  "includeMcpJson": false,
  "mcpServers": {
    "agent-only": {"command": "agent", "args": []}
  }
}`)
	t.Setenv("USERPROFILE", filepath.Join(dir, "missing-home"))

	servers, _, err := buildACPMcpServers(filepath.Join(dir, "proj"), "solo")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 1 || servers[0].(map[string]any)["name"] != "agent-only" {
		t.Fatalf("got %+v", servers)
	}
}

func TestResolveWorkDir(t *testing.T) {
	dir := t.TempDir()
	res, err := resolveWorkDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != dir || res.Fallback {
		t.Fatalf("got %+v want path=%q fallback=false", res, dir)
	}

	t.Chdir(dir)
	res, err = resolveWorkDir("")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Fallback || res.Path != dir {
		t.Fatalf("empty work_dir should fallback to server cwd: %+v", res)
	}

	if _, err := resolveWorkDir(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing dir should fail")
	}
}

func indexServers(t *testing.T, servers []any) map[string]map[string]any {
	t.Helper()
	out := make(map[string]map[string]any, len(servers))
	for _, raw := range servers {
		srv, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected type %T", raw)
		}
		out[srv["name"].(string)] = srv
	}
	return out
}

func mustMkdirAll(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
