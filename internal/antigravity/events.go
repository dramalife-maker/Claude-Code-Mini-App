package antigravity

import (
	"encoding/json"
	"strings"
)

// agy CLI (v1.1.8+) 的 stream-json 事件格式。
// 頂層 event 區分：init | step_update | result

// StreamTopEvent 為 stream-json 頂層事件。
type StreamTopEvent struct {
	Event          string             `json:"event"`
	ConversationID string             `json:"conversation_id,omitempty"`
	Init           *InitPayload       `json:"init,omitempty"`
	StepUpdate     *StepUpdatePayload `json:"step_update,omitempty"`
	Result         *ResultPayload     `json:"result,omitempty"`
}

// InitPayload 對應 event: "init" 的內容。
type InitPayload struct {
	Cwd            string   `json:"cwd,omitempty"`
	Tools          []string `json:"tools,omitempty"`
	PermissionMode string   `json:"permission_mode,omitempty"`
}

// StepUpdatePayload 對應 event: "step_update" 的內容。
type StepUpdatePayload struct {
	ConversationID string           `json:"conversation_id,omitempty"`
	StepIndex      int              `json:"step_index"`
	State          string           `json:"state"`     // ACTIVE | DONE | ERROR
	StepType       string           `json:"step_type"` // agent_response | tool | user_input | system_message
	TextDelta      string           `json:"text_delta,omitempty"`
	DurationSec    float64          `json:"duration_seconds,omitempty"`
	Usage          *UsagePayload    `json:"usage,omitempty"`
	ToolName       string           `json:"tool_name,omitempty"`
	ToolInfo       *ToolInfoPayload `json:"tool_info,omitempty"`
}

// ToolInfoPayload 對應 step_type: "tool" 的 tool_info 物件。
type ToolInfoPayload struct {
	Name       string          `json:"name,omitempty"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
	Output     json.RawMessage `json:"output,omitempty"`
	Error      *ToolErrorInfo  `json:"error,omitempty"`
}

// ToolErrorInfo 對應 tool_info.error。
type ToolErrorInfo struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

// ResultPayload 對應 event: "result" 的內容。
type ResultPayload struct {
	ConversationID string         `json:"conversation_id,omitempty"`
	Status         string         `json:"status"` // SUCCESS | ERROR
	Response       string         `json:"response,omitempty"`
	DurationSec    float64        `json:"duration_seconds,omitempty"`
	NumTurns       int            `json:"num_turns,omitempty"`
	Usage          *UsagePayload  `json:"usage,omitempty"`
	DeniedActions  []DeniedAction `json:"denied_actions,omitempty"`
	Error          string         `json:"error,omitempty"`
}

// DeniedAction 記錄被安全政策自動拒絕的動作。
type DeniedAction struct {
	Action      string `json:"action,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// UsagePayload 統計 token 消耗。
type UsagePayload struct {
	InputTokens     int64 `json:"input_tokens,omitempty"`
	OutputTokens    int64 `json:"output_tokens,omitempty"`
	ThinkingTokens  int64 `json:"thinking_tokens,omitempty"`
	CacheReadTokens int64 `json:"cache_read_tokens,omitempty"`
	TotalTokens     int64 `json:"total_tokens,omitempty"`
}

// OutputText 取出 tool_info.output 文字。
func (t *ToolInfoPayload) OutputText() string {
	if t == nil || len(t.Output) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(t.Output, &s); err == nil {
		return s
	}
	return strings.TrimSpace(string(t.Output))
}

// ParseEvent 解析單行 NDJSON 為 StreamTopEvent。
func ParseEvent(line []byte) (*StreamTopEvent, error) {
	var e StreamTopEvent
	if err := json.Unmarshal(line, &e); err != nil {
		return nil, err
	}
	return &e, nil
}
