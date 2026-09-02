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
	Command  string            `json:"command" yaml:"command"`
	Args     []string          `json:"args" yaml:"args"`
	URL      string            `json:"url" yaml:"url"`
	Headers  map[string]string `json:"headers" yaml:"headers"`
	Env      map[string]string `json:"env" yaml:"env"`
	Disabled *bool             `json:"disabled" yaml:"disabled"`
}

type mcpLoadMeta struct {
	CWD         string
	Agent       string
	AgentSource string
	JSONSources []string
	ServerNames []string
}

// buildACPMcpServers 依 Kiro 優先順序合併 MCP：Agent > Workspace mcp.json > Global mcp.json。
// 無 agent 時等同目錄開 Kiro：只合併兩層 mcp.json。無任何設定時回傳空陣列（P3：不再硬塞 windows-mcp）。
func buildACPMcpServers(cwd, agentName string) ([]any, mcpLoadMeta, error) {
	meta := mcpLoadMeta{CWD: cwd, Agent: strings.TrimSpace(agentName)}

	agentCfg, agentPath, err := loadAgentConfig(cwd, agentName)
	if err != nil {
		return nil, meta, err
	}
	if agentPath != "" {
		meta.AgentSource = agentPath
	}

	merged := make(map[string]kiroMcpServer)
	includeJSON := shouldIncludeMcpJSON(agentName, agentCfg)
	if includeJSON {
		globalPath := kiroGlobalMcpPath()
		workspacePath := kiroWorkspaceMcpPath(cwd)
		for name, srv := range mergeKiroMcpConfigs(globalPath, workspacePath) {
			merged[name] = srv
		}
		if _, err := os.Stat(globalPath); err == nil {
			meta.JSONSources = append(meta.JSONSources, globalPath)
		}
		if _, err := os.Stat(workspacePath); err == nil {
			meta.JSONSources = append(meta.JSONSources, workspacePath)
		}
	}

	if agentCfg != nil {
		for name, srv := range agentCfg.McpServers {
			merged[name] = srv
		}
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
			meta.ServerNames = append(meta.ServerNames, name)
		}
	}
	return out, meta, nil
}

// shouldIncludeMcpJSON：無 agent 時永遠載入 mcp.json；有 agent 時依 includeMcpJson（省略視為 true）。
func shouldIncludeMcpJSON(agentName string, agentCfg *kiroAgentConfig) bool {
	if strings.TrimSpace(agentName) == "" {
		return true
	}
	if agentCfg == nil {
		return true
	}
	if agentCfg.IncludeMcpJson == nil {
		return true
	}
	return *agentCfg.IncludeMcpJson
}

func kiroHomeDir() string {
	if home := strings.TrimSpace(os.Getenv("USERPROFILE")); home != "" {
		return home
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func kiroGlobalMcpPath() string {
	home := kiroHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".kiro", "settings", "mcp.json")
}

func kiroWorkspaceMcpPath(cwd string) string {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return ""
	}
	return filepath.Join(cwd, ".kiro", "settings", "mcp.json")
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
			"name":    name,
			"type":    "http",
			"url":     url,
			"headers": acpEnvEntries(srv.Headers),
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

func logMcpLoad(meta mcpLoadMeta) {
	log.Printf("[kiroacp] MCP cwd=%s agent=%q agentSrc=%q jsonSrc=%v servers=%v count=%d",
		meta.CWD, meta.Agent, meta.AgentSource, meta.JSONSources, meta.ServerNames, len(meta.ServerNames))
}
