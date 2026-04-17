package claude

import (
	"slices"
	"testing"

	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
)

func TestBuildClaudeArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts agent.RunOptions
		want []string
	}{
		{
			name: "僅內建旗標與預設 permission_mode",
			opts: agent.RunOptions{},
			want: []string{
				"-p", "--output-format", "stream-json", "--verbose",
				"--permission-mode", "default",
			},
		},
		{
			name: "resume 與自訂 permission_mode",
			opts: agent.RunOptions{
				SessionID: "sess-uuid",
				ExtraArgs: map[string]string{
					agent.ArgPermissionMode: "acceptEdits",
				},
			},
			want: []string{
				"-p", "--output-format", "stream-json", "--verbose",
				"--resume", "sess-uuid",
				"--permission-mode", "acceptEdits",
			},
		},
		{
			name: "多個 CliExtraArgs 置於 -p 之前（例如多個 --plugin-dir）",
			opts: agent.RunOptions{
				CliExtraArgs: []string{
					"--plugin-dir", "./.claude/plugins/crm",
					"--plugin-dir", "./.claude/plugins/crm2",
				},
				SessionID: "abc",
				ExtraArgs: map[string]string{
					agent.ArgPermissionMode: "default",
				},
			},
			want: []string{
				"--plugin-dir", "./.claude/plugins/crm",
				"--plugin-dir", "./.claude/plugins/crm2",
				"-p", "--output-format", "stream-json", "--verbose",
				"--resume", "abc",
				"--permission-mode", "default",
			},
		},
		{
			name: "路徑含空格之單一 argv 保持為一個元素",
			opts: agent.RunOptions{
				CliExtraArgs: []string{"--plugin-dir", "./.claude/plugins/my project"},
			},
			want: []string{
				"--plugin-dir", "./.claude/plugins/my project",
				"-p", "--output-format", "stream-json", "--verbose",
				"--permission-mode", "default",
			},
		},
		{
			name: "allowedTools 拆成多個 --allowedTools",
			opts: agent.RunOptions{
				ExtraArgs: map[string]string{
					agent.ArgPermissionMode: "default",
					agent.ArgAllowedTools:   " Write , Edit ",
				},
			},
			want: []string{
				"-p", "--output-format", "stream-json", "--verbose",
				"--permission-mode", "default",
				"--allowedTools", "Write",
				"--allowedTools", "Edit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildClaudeArgs(tt.opts)
			if !slices.Equal(got, tt.want) {
				t.Errorf("buildClaudeArgs() = %#v\nwant %#v", got, tt.want)
			}
		})
	}
}
