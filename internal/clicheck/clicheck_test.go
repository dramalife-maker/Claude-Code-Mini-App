package clicheck

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCheckResolution(t *testing.T) {
	t.Run("PATH 命中", func(t *testing.T) {
		dir := t.TempDir()
		isolateLookup(t, dir)
		for _, name := range []string{"claude", "codex", "cursor-agent", "kiro-cli", "agy"} {
			writeFake(t, dir, name)
		}
		tools := Checker{Run: noopRun}.Check(context.Background())
		assertOrder(t, tools)
		for _, tool := range tools {
			if !tool.Found || !pathIn(dir, tool.Path) {
				t.Fatalf("%s 未在 PATH 找到: %+v", tool.Command, tool)
			}
		}
		if tools[2].Note != cursorNote {
			t.Fatalf("note = %q", tools[2].Note)
		}
	})

	t.Run("CODEX_BIN", func(t *testing.T) {
		dir := t.TempDir()
		isolateLookup(t, dir)
		for _, name := range []string{"claude", "cursor-agent", "kiro-cli", "agy"} {
			writeFake(t, dir, name)
		}
		codexPath := filepath.Join(t.TempDir(), "codex-custom")
		if err := os.WriteFile(codexPath, nil, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("CODEX_BIN", codexPath)
		tools := Checker{Run: noopRun}.Check(context.Background())
		codex := toolByCommand(t, tools, "codex")
		if !codex.Found || codex.Path != codexPath {
			t.Fatalf("codex = %+v，預期 %s", codex, codexPath)
		}
	})

	t.Run("CC_AGY_BIN", func(t *testing.T) {
		dir := t.TempDir()
		isolateLookup(t, dir)
		agyPath := filepath.Join(t.TempDir(), "agy-custom")
		if err := os.WriteFile(agyPath, nil, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("CC_AGY_BIN", agyPath)
		tools := Checker{Run: noopRun}.Check(context.Background())
		agy := toolByCommand(t, tools, "agy")
		if !agy.Found || agy.Path != agyPath {
			t.Fatalf("agy = %+v，預期 %s", agy, agyPath)
		}
	})

	t.Run("全部缺失", func(t *testing.T) {
		isolateLookup(t, t.TempDir())
		tools := Checker{Run: noopRun}.Check(context.Background())
		assertOrder(t, tools)
		for _, tool := range tools {
			if tool.Found || tool.Path != "" {
				t.Fatalf("%s 不該被找到: %+v", tool.Command, tool)
			}
			if tool.InstallURL == "" {
				t.Fatalf("%s 缺少 install_url", tool.Command)
			}
		}
		if tools[2].Note != cursorNote {
			t.Fatalf("缺失時 note = %q", tools[2].Note)
		}
	})

	t.Run("單項缺失", func(t *testing.T) {
		dir := t.TempDir()
		isolateLookup(t, dir)
		for _, name := range []string{"claude", "codex", "kiro-cli", "agy"} {
			writeFake(t, dir, name)
		}
		tools := Checker{Run: noopRun}.Check(context.Background())
		for _, tool := range tools {
			want := tool.Command != "cursor-agent"
			if tool.Found != want {
				t.Fatalf("%s found=%v，預期 %v", tool.Command, tool.Found, want)
			}
		}
	})
}

func TestCheckVersion(t *testing.T) {
	timeout := 80 * time.Millisecond
	c := Checker{
		Timeout:      timeout,
		LookPath:     func(name string) (string, error) { return "bin-" + name, nil },
		ResolveCodex: func() (string, error) { return "bin-codex", nil },
		AgyBin:       func() string { return "agy" },
		Run: func(ctx context.Context, path string) (string, string, error) {
			switch path {
			case "bin-claude":
				return "\n  claude 1.2  \nextra\n", "", nil
			case "bin-codex":
				return " \n\t\n", " codex 9 \nmore\n", nil
			case "bin-cursor-agent":
				return "cursor 3", "", nil
			case "bin-kiro-cli":
				return "should-ignore", "", errors.New("boom")
			case "bin-agy":
				select {
				case <-ctx.Done():
					return "", "", ctx.Err()
				case <-time.After(2 * time.Second):
					return "late", "", nil
				}
			default:
				return "", "", errors.New("unexpected " + path)
			}
		},
	}
	start := time.Now()
	tools := c.Check(context.Background())
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("花了 %s，逾時 %s 應在略久於逾時後返回", elapsed, timeout)
	}
	got := map[string]string{}
	for _, tool := range tools {
		if !tool.Found {
			t.Fatalf("%s 不該缺失", tool.Command)
		}
		got[tool.Command] = tool.Version
	}
	if got["claude"] != "claude 1.2" || got["codex"] != "codex 9" || got["cursor-agent"] != "cursor 3" {
		t.Fatalf("版本 = %v", got)
	}
	if got["kiro-cli"] != "" || got["agy"] != "" {
		t.Fatalf("失敗或逾時仍有版本: %v", got)
	}
}

func TestCheckVersionParallel(t *testing.T) {
	const pause = 200 * time.Millisecond
	c := Checker{
		Timeout: time.Second,
		LookPath: func(name string) (string, error) {
			if name == "claude" || name == "cursor-agent" {
				return name, nil
			}
			return "", errors.New("missing")
		},
		ResolveCodex: func() (string, error) { return "", errors.New("missing") },
		AgyBin:       func() string { return "" },
		Run: func(context.Context, string) (string, string, error) {
			time.Sleep(pause)
			return "ok", "", nil
		},
	}
	start := time.Now()
	tools := c.Check(context.Background())
	if elapsed := time.Since(start); elapsed >= 2*pause {
		t.Fatalf("總時間 %s，並行應少於 %s", elapsed, 2*pause)
	}
	for _, tool := range tools {
		if (tool.Command == "claude" || tool.Command == "cursor-agent") && tool.Version != "ok" {
			t.Fatalf("%s version = %q", tool.Command, tool.Version)
		}
	}
}

func TestCheckVersionBoundedWithLingeringChild(t *testing.T) {
	dir := t.TempDir()
	var path string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "fake.cmd")
		// 孫行程繼承 stdout。父行程被殺後，管線仍開著，Wait 會一直等到孫行程結束。
		script := "@echo off\r\nstart /b ping -n 7 127.0.0.1\r\nping -n 7 127.0.0.1\r\n"
		if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
			t.Fatal(err)
		}
	} else {
		path = filepath.Join(dir, "fake.sh")
		script := "#!/bin/sh\nsleep 6 &\nsleep 6\n"
		if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	c := Checker{
		Timeout: 500 * time.Millisecond,
		LookPath: func(name string) (string, error) {
			if name == "claude" {
				return path, nil
			}
			return "", errors.New("missing")
		},
		ResolveCodex: func() (string, error) { return "", errors.New("missing") },
		AgyBin:       func() string { return "" },
	}
	const limit = 500*time.Millisecond + time.Second + 3*time.Second
	start := time.Now()
	tools := c.Check(context.Background())
	elapsed := time.Since(start)
	t.Logf("耗時 %s", elapsed)
	if elapsed > limit {
		t.Fatalf("花了 %s，上限 %s", elapsed, limit)
	}
	claude := toolByCommand(t, tools, "claude")
	if !claude.Found || claude.Version != "" {
		t.Fatalf("found=%v version=%q", claude.Found, claude.Version)
	}
}

func noopRun(context.Context, string) (string, string, error) { return "", "", nil }

func isolateLookup(t *testing.T, pathDir string) {
	t.Helper()
	t.Setenv("PATH", pathDir)
	t.Setenv("CODEX_BIN", "")
	t.Setenv("CC_AGY_BIN", "")
	// ResolveBin 在 Windows 會再看使用者目錄下的預設安裝路徑。改到空目錄，避免測到這台機器已安裝的 Codex。
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
}

func writeFake(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), nil, 0o755); err != nil {
		t.Fatal(err)
	}
}

func assertOrder(t *testing.T, tools []Tool) {
	t.Helper()
	want := []string{"claude", "codex", "cursor-agent", "kiro-cli", "agy"}
	if len(tools) != len(want) {
		t.Fatalf("len = %d", len(tools))
	}
	for i, name := range want {
		if tools[i].Command != name {
			t.Fatalf("順序[%d] = %s，預期 %s", i, tools[i].Command, name)
		}
	}
}

func toolByCommand(t *testing.T, tools []Tool, command string) Tool {
	t.Helper()
	for _, tool := range tools {
		if tool.Command == command {
			return tool
		}
	}
	t.Fatalf("沒有 %s", command)
	return Tool{}
}

func pathIn(dir, p string) bool {
	rel, err := filepath.Rel(dir, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
