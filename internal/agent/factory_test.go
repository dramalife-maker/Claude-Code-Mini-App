package agent

import "testing"

func TestIsEnabledAntigravityEnabled(t *testing.T) {
	for _, typ := range []string{TypeAntigravity, "antigravity"} {
		if !IsEnabled(typ) {
			t.Fatalf("%q should be enabled", typ)
		}
		if !CanCreate(typ) {
			t.Fatalf("%q should be creatable", typ)
		}
		if reason := DisabledReason(typ); reason != "" {
			t.Fatalf("%q should have no disabled reason, got %q", typ, reason)
		}
	}
}

func TestGeminiAliasAndCreateDisabled(t *testing.T) {
	if !IsEnabled(TypeGemini) {
		t.Fatal("gemini should be enabled via antigravity alias")
	}
	if CanCreate(TypeGemini) {
		t.Fatal("gemini should not be creatable")
	}
	if reason := CreateDisabledReason(TypeGemini); reason == "" {
		t.Fatal("gemini should have create-disabled reason")
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
