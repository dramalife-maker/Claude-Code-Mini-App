package agent

import "testing"

func TestIsEnabledGeminiAntigravityDisabled(t *testing.T) {
	for _, typ := range []string{TypeGemini, TypeAntigravity, "gemini", "antigravity"} {
		if IsEnabled(typ) {
			t.Fatalf("%q should be disabled", typ)
		}
		if reason := DisabledReason(typ); reason == "" {
			t.Fatalf("%q should have disabled reason", typ)
		}
	}
}

func TestNewRunnerGeminiRejected(t *testing.T) {
	for _, typ := range []string{TypeGemini, TypeAntigravity} {
		if _, err := NewRunner(typ); err == nil {
			t.Fatalf("NewRunner(%q) should fail", typ)
		}
	}
}

func TestIsEnabledClaude(t *testing.T) {
	if !IsEnabled(TypeClaude) {
		t.Fatal("claude should be enabled")
	}
}

func TestCanCreateKiroDisabledKeepACP(t *testing.T) {
	if !IsEnabled(TypeKiro) {
		t.Fatal("既有 kiro session 仍應可執行")
	}
	if CanCreate(TypeKiro) {
		t.Fatal("kiro 不可新建")
	}
	if CreateDisabledReason(TypeKiro) == "" {
		t.Fatal("kiro 應有不可新建原因")
	}
	if !CanCreate(TypeKiroACP) {
		t.Fatal("kiroacp 應可新建")
	}
	if !CanCreate(TypeClaude) {
		t.Fatal("claude 應可新建")
	}
}
