package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/agent"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
)

const questionFixture = `{"questions":[{"question":"Colors?","header":"Colors","multiSelect":true,"options":[{"label":"Blue","description":"Cool"},{"label":"Green","description":"Nature"}]},{"question":"Size?","header":"Size","multiSelect":false,"options":[{"label":"Small"},{"label":"Large"}]}],"metadata":{"keep":true}}`

func TestQuestionValidationAndDuplicate(t *testing.T) {
	q, err := newQuestion(json.RawMessage(questionFixture))
	if err != nil {
		t.Fatal(err)
	}
	pendingQuestions.Lock()
	pendingQuestions.m[t.Name()] = q
	pendingQuestions.Unlock()
	t.Cleanup(func() { pendingQuestions.Lock(); delete(pendingQuestions.m, t.Name()); pendingQuestions.Unlock() })
	valid := []questionAnswer{{Selected: []string{"Blue", "Green"}}, {Text: "Medium"}}
	for _, bad := range [][]questionAnswer{
		nil,
		{{Selected: []string{"Unknown"}}, {Text: "Medium"}},
		{{Selected: []string{"Blue", "Blue"}}, {Text: "Medium"}},
		{{Selected: []string{"Blue"}}, {Selected: []string{"Small", "Large"}}},
		{{Selected: []string{"Blue"}}, {Selected: []string{"Small"}, Text: "Medium"}},
	} {
		if questionResolve(t.Name(), q.RequestID, bad) == nil {
			t.Fatalf("accepted invalid answers: %+v", bad)
		}
	}
	if questionResolve("another-session", q.RequestID, valid) == nil || questionResolve(t.Name(), "stale-id", valid) == nil {
		t.Fatal("accepted wrong session or request")
	}
	if err := questionResolve(t.Name(), q.RequestID, valid); err != nil {
		t.Fatal(err)
	}
	if questionResolve(t.Name(), q.RequestID, valid) == nil {
		t.Fatal("accepted duplicate")
	}
	var out struct {
		Answers   map[string]string
		Metadata  map[string]bool
		Questions []userQuestion
	}
	if err := json.Unmarshal(<-q.ch, &out); err != nil {
		t.Fatal(err)
	}
	if out.Answers["Colors?"] != "Blue, Green" || out.Answers["Size?"] != "Medium" || !out.Metadata["keep"] || len(out.Questions) != 2 {
		t.Fatalf("lost answers/original input: %+v", out)
	}
	for _, input := range []string{`null`, `{}`, `{"questions":[]}`, `{"questions":[{"question":"Bad","options":[{"label":"A"},{"label":"A"}]}]}`} {
		if _, err := newQuestion(json.RawMessage(input)); err == nil {
			t.Fatalf("accepted malformed question: %s", input)
		}
	}
}

type questionRunner struct{ sid string }

func (questionRunner) Name() string { return "claude" }
func (r questionRunner) Run(ctx context.Context, opts agent.RunOptions, cb agent.EventCallback) error {
	out, err := AskClaudeQuestion(ctx, r.sid, json.RawMessage(questionFixture), "question-tool-1")
	if err != nil {
		return err
	}
	cb(agent.Event{Type: agent.EventDelta, Text: string(out)})
	cb(agent.Event{Type: agent.EventDone})
	return nil
}

func readQuestionEvent(t *testing.T, c *websocket.Conn, kind string) serverMsg {
	t.Helper()
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		var event serverMsg
		if err := c.ReadJSON(&event); err != nil {
			t.Fatalf("waiting for %s: %v", kind, err)
		}
		if event.Type == "permission_request" {
			t.Fatal("question misclassified as permission")
		}
		if event.Type == kind {
			return event
		}
	}
}

func TestQuestionReconnectAnswerAndInterrupt(t *testing.T) {
	database, wsURL := startTestServer(t)
	s, err := database.CreateSession("questions", "", t.TempDir(), "default", agent.TypeClaude, nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	agent.Register(agent.TypeClaude, func() agent.Runner { return questionRunner{s.ID} })
	t.Cleanup(func() { agent.Register(agent.TypeClaude, func() agent.Runner { return fakeClaude{} }) })
	SetClaudePermMCP("http://127.0.0.1:1/mcp/perm", "test")
	t.Cleanup(func() { SetClaudePermMCP("", "") })
	dial := func() *websocket.Conn {
		c, _, err := websocket.DefaultDialer.Dial(wsURL(s.ID), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.Close() })
		return c
	}
	a := dial()
	if err := a.WriteJSON(clientMsg{Type: "input", Data: "ask"}); err != nil {
		t.Fatal(err)
	}
	first := readQuestionEvent(t, a, "question_request")
	a.Close()
	b := dial()
	sync := readQuestionEvent(t, b, "sync")
	if sync.Value != StateAwaitingInput {
		t.Fatalf("state=%s", sync.Value)
	}
	q := readQuestionEvent(t, b, "question_request").Question
	if q.RequestID != first.Question.RequestID {
		t.Fatal("reconnect replaced request ID")
	}
	if err := b.WriteJSON(clientMsg{Type: "question_answer", RequestID: "old"}); err != nil {
		t.Fatal(err)
	}
	readQuestionEvent(t, b, "question_error")
	if err := b.WriteJSON(clientMsg{Type: "question_answer", RequestID: q.RequestID, Answers: []questionAnswer{{Selected: []string{"Blue", "Green"}}, {Text: "Medium"}}}); err != nil {
		t.Fatal(err)
	}
	readQuestionEvent(t, b, "question_resolved")
	waitFor(t, "answer delivered and task complete", func() bool {
		x, _ := database.GetSession(s.ID)
		return x.Status == db.SessionStatusIdle && !taskIsActive(s.ID)
	})
	var out struct{ Answers map[string]string }
	if err := json.Unmarshal([]byte(lastReply(database, s.ID)), &out); err != nil || out.Answers["Colors?"] != "Blue, Green" {
		t.Fatalf("answer lost: %+v %v", out, err)
	}
	if err := b.WriteJSON(clientMsg{Type: "input", Data: "ask again"}); err != nil {
		t.Fatal(err)
	}
	readQuestionEvent(t, b, "question_request")
	if err := b.WriteJSON(clientMsg{Type: "interrupt"}); err != nil {
		t.Fatal(err)
	}
	readQuestionEvent(t, b, "question_resolved")
	waitFor(t, "cancel cleared pending question", func() bool {
		x, _ := database.GetSession(s.ID)
		return x.Status == db.SessionStatusIdle && !taskIsActive(s.ID) && questionPending(s.ID) == nil
	})
}
