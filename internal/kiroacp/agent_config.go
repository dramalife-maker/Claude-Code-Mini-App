package kiroacp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

type kiroAgentConfig struct {
	Name           string                   `json:"name" yaml:"name"`
	McpServers     map[string]kiroMcpServer `json:"mcpServers" yaml:"mcpServers"`
	IncludeMcpJson *bool                    `json:"includeMcpJson" yaml:"includeMcpJson"`
}

// loadAgentConfig 依名稱載入 agent（workspace 優先於 global；支援 .json / .md frontmatter）。
func loadAgentConfig(cwd, agentName string) (*kiroAgentConfig, string, error) {
	agentName = strings.TrimSpace(agentName)
	if agentName == "" {
		return nil, "", nil
	}
	for _, path := range agentConfigCandidates(cwd, agentName) {
		cfg, err := readAgentConfigFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, "", err
		}
		return cfg, path, nil
	}
	return nil, "", fmt.Errorf("kiroacp: 找不到 agent %q（已查 .kiro/agents）", agentName)
}

func agentConfigCandidates(cwd, agentName string) []string {
	names := []string{agentName}
	if ext := filepath.Ext(agentName); ext != "" {
		names = append(names, strings.TrimSuffix(agentName, ext))
	}
	var paths []string
	add := func(base string) {
		for _, n := range names {
			paths = append(paths,
				filepath.Join(base, n+".json"),
				filepath.Join(base, n+".md"),
			)
		}
	}
	add(filepath.Join(cwd, ".kiro", "agents"))
	if home := kiroHomeDir(); home != "" {
		add(filepath.Join(home, ".kiro", "agents"))
	}
	return paths
}

func readAgentConfigFile(path string) (*kiroAgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(filepath.Ext(path), ".md") {
		return parseAgentMarkdown(data)
	}
	var cfg kiroAgentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.McpServers == nil {
		cfg.McpServers = map[string]kiroMcpServer{}
	}
	return &cfg, nil
}

func parseAgentMarkdown(data []byte) (*kiroAgentConfig, error) {
	fm, err := splitMarkdownFrontmatter(data)
	if err != nil {
		return nil, err
	}
	if len(fm) == 0 {
		return &kiroAgentConfig{McpServers: map[string]kiroMcpServer{}}, nil
	}
	var cfg kiroAgentConfig
	if err := yaml.Unmarshal(fm, &cfg); err != nil {
		return nil, fmt.Errorf("解析 agent frontmatter: %w", err)
	}
	if cfg.McpServers == nil {
		cfg.McpServers = map[string]kiroMcpServer{}
	}
	return &cfg, nil
}

func splitMarkdownFrontmatter(data []byte) ([]byte, error) {
	text := string(data)
	if !strings.HasPrefix(text, "---") {
		return nil, nil
	}
	rest := text[3:]
	if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	} else {
		return nil, nil
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, fmt.Errorf("agent markdown 缺少 frontmatter 結尾 ---")
	}
	return []byte(rest[:end]), nil
}
