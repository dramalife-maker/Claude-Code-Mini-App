package cursor

import (
	"encoding/json"
	"strings"
)

// 參考 docs/cursor-agent-cli.md 與官方 output-format 文件。
// stream-json 採 NDJSON，每行一個 JSON 物件。

// StreamEvent 為 cursor-agent stream-json 的頂層事件。
// 欄位採鬆綁定義，未知欄位會被忽略以保持前向相容。
type StreamEvent struct {
	Type    string `json:"type"`              // system | user | assistant | tool_call | result
	Subtype string `json:"subtype,omitempty"` // init | started | completed | success | error

	SessionID string `json:"session_id,omitempty"`
	CWD       string `json:"cwd,omitempty"`
	Model     string `json:"model,omitempty"`

	// assistant / user
	Message *Message `json:"message,omitempty"`

	// assistant streaming delta 專屬（--stream-partial-output）
	// 用來區分三種 assistant 事件：
	//   streaming delta: 有 timestamp_ms、無 model_call_id → 附加
	//   buffered flush : 兩者皆有 → 略過
	//   final flush    : 兩者皆無 → 略過
	TimestampMS *int64  `json:"timestamp_ms,omitempty"`
	ModelCallID *string `json:"model_call_id,omitempty"`

	// tool_call
	CallID   string          `json:"call_id,omitempty"`
	ToolCall json.RawMessage `json:"tool_call,omitempty"`

	// result
	IsError       bool   `json:"is_error,omitempty"`
	Result        string `json:"result,omitempty"`
	DurationMS    int64  `json:"duration_ms,omitempty"`
	DurationAPIMS int64  `json:"duration_api_ms,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
}

// Message 對應 assistant / user 事件中的 message 欄位。
type Message struct {
	Role    string           `json:"role"`
	Content []MessageContent `json:"content"`
}

// MessageContent 是 message.content 的項目。
type MessageContent struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

// Text 串接所有 text content 後回傳。
func (m *Message) Text() string {
	if m == nil {
		return ""
	}
	var out string
	for _, c := range m.Content {
		if c.Type == "" || c.Type == "text" {
			out += c.Text
		}
	}
	return out
}

// ParseEvent 解析單行 NDJSON 為 StreamEvent。
func ParseEvent(line []byte) (*StreamEvent, error) {
	var e StreamEvent
	if err := json.Unmarshal(line, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// ToolLabel 從多型 tool_call payload 取出前端活動提示文字。
// 已知 schema：readToolCall / writeToolCall；fallback：function.name。
// ponytail: tool_call 物件預期只有單一工具鍵，故 map 首個相符鍵即可（順序不定但只有一個）。
func (e *StreamEvent) ToolLabel() string {
	if len(e.ToolCall) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(e.ToolCall, &m) != nil {
		return ""
	}
	if raw, ok := m["function"]; ok {
		var fn struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(raw, &fn) == nil && strings.TrimSpace(fn.Name) != "" {
			return "呼叫工具 " + fn.Name + "…"
		}
	}
	for k := range m {
		switch k {
		case "readToolCall":
			return "讀取檔案中…"
		case "writeToolCall":
			return "修改檔案中…"
		default:
			if name := strings.TrimSuffix(k, "ToolCall"); name != k && name != "" {
				return "使用工具 " + name + "…"
			}
		}
	}
	return "使用工具中…"
}
