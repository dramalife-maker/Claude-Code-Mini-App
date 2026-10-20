package antigravity

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/proc"
)

type ModelEntry struct {
	ModelID string
	Label   string
}

// FetchModelOptions 解析 agy models 的 TSV 輸出；失敗或空清單不應覆寫既有選項。
func FetchModelOptions(ctx context.Context) ([]ModelEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, Binary(), "models")
	cmd.SysProcAttr = proc.SysProcAttr()
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return nil, fmt.Errorf("agy models: %w", err)
	}

	var entries []ModelEntry
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		id, label, ok := strings.Cut(scanner.Text(), "\t")
		if !ok { // CLI 的進度訊息不是 TSV。
			continue
		}
		id, label = strings.TrimSpace(id), strings.TrimSpace(label)
		if id == "" || label == "" || strings.ContainsAny(id, " \t\r\n") || strings.Contains(label, "\t") {
			return nil, fmt.Errorf("agy models: invalid model row")
		}
		entries = append(entries, ModelEntry{ModelID: id, Label: label})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("agy models: no models found")
	}
	return entries, nil
}
