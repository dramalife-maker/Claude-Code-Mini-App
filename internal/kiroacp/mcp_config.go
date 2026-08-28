package kiroacp

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// kiroMcpFile 對應 ~/.kiro/settings/mcp.json 與 {cwd}/.kiro/settings/mcp.json。
type kiroMcpFile struct {
	McpServers map[string]kiroMcpServer `json:"mcpServers"`
}

type kiroMcpServer struct {
	Command  string            `json:"command"`
	Args     []string          `json:"args"`
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers"`
	Env      map[string]string `json:"env"`
	Disabled *bool             `json:"disabled"`
}

// mcpServersForACP 依 Kiro 慣例合併 global + workspace MCP，轉成 ACP session/new|load 格式。
// 優先順序：workspace 覆蓋 global；皆無時退回內建 windows-mcp。
func mcpServersForACP(cwd string) []any {
	merged := mergeKiroMcpConfigs(kiroGlobalMcpPath(), kiroWorkspaceMcpPath(cwd))
	if len(merged) == 0 {
		return fallbackMcpServers()
	}
	out := make([]any, 0, len(merged))
	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if srv, ok := toACPMcpServer(name, merged[name]); ok {
			out = append(out, srv)
		}
	}
	if len(out) == 0 {
		return fallbackMcpServers()
	}
	return out
}

func kiroGlobalMcpPath() string {
	if home := strings.TrimSpace(os.Getenv("USERPROFILE")); home != "" {
		return filepath.Join(home, ".kiro", "settings", "mcp.json")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".kiro", "settings", "mcp.json")
	}
	return ""
}

func kiroWorkspaceMcpPath(cwd string) string {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" || cwd == "." {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}
	return filepath.Join(abs, ".kiro", "settings", "mcp.json")
}

func mergeKiroMcpConfigs(paths ...string) map[string]kiroMcpServer {
	merged := make(map[string]kiroMcpServer)
	for _, path := range paths {
		if path == "" {
			continue
		}
		cfg, err := readKiroMcpFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("[kiroacp] 讀取 MCP 設定 %s: %v", path, err)
			}
			continue
		}
		for name, srv := range cfg.McpServers {
			merged[name] = srv
		}
	}
	return merged
}

func readKiroMcpFile(path string) (*kiroMcpFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg kiroMcpFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.McpServers == nil {
		cfg.McpServers = map[string]kiroMcpServer{}
	}
	return &cfg, nil
}

func toACPMcpServer(name string, srv kiroMcpServer) (map[string]any, bool) {
	if srv.Disabled != nil && *srv.Disabled {
		return nil, false
	}

	url := strings.TrimSpace(srv.URL)
	if url != "" {
		out := map[string]any{
			"name": name,
			"type": "http",
			"url":  url,
		}
		if len(srv.Headers) > 0 {
			out["headers"] = srv.Headers
		}
		return out, true
	}

	command := strings.TrimSpace(srv.Command)
	if command == "" {
		log.Printf("[kiroacp] 略過 MCP %q：缺少 command 或 url", name)
		return nil, false
	}

	args := srv.Args
	if args == nil {
		args = []string{}
	}
	return map[string]any{
		"name":    name,
		"command": command,
		"args":    args,
		"env":     acpEnvEntries(srv.Env),
	}, true
}

func acpEnvEntries(env map[string]string) []any {
	if len(env) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{
			"name":  k,
			"value": env[k],
		})
	}
	return out
}

// fallbackMcpServers 在找不到任何 Kiro MCP 設定時的保底清單。
func fallbackMcpServers() []any {
	uvx := filepath.Join(os.Getenv("USERPROFILE"), `.local`, `bin`, `uvx.exe`)
	if _, err := os.Stat(uvx); err != nil {
		uvx = "uvx"
	}
	return []any{
		map[string]any{
			"name":    "windows-mcp",
			"command": uvx,
			"args":    []string{"windows-mcp", "serve"},
			"env":     []any{},
		},
	}
}
