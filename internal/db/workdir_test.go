package db

import (
	"testing"
)

func TestWorkDirCatalog_SurvivesSessionDelete(t *testing.T) {
	database, err := Open(t.TempDir() + "/workdir.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	s, err := database.CreateSession("t", "", "/tmp/proj-a", "default", "claude", nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	if s.WorkDir != "/tmp/proj-a" {
		t.Fatalf("WorkDir=%q", s.WorkDir)
	}

	dirs, err := database.ListWorkDirs()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 1 || dirs[0] != "/tmp/proj-a" {
		t.Fatalf("建立後清單=%+v", dirs)
	}

	if err := database.DeleteSession(s.ID); err != nil {
		t.Fatal(err)
	}
	dirs, err = database.ListWorkDirs()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 1 || dirs[0] != "/tmp/proj-a" {
		t.Fatalf("刪 session 後清單應仍在，got=%+v", dirs)
	}
}

func TestWorkDirCatalog_EmptyNotInserted(t *testing.T) {
	database, err := Open(t.TempDir() + "/workdir_empty.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	if _, err := database.CreateSession("t", "", "   ", "default", "claude", nil, "agent"); err != nil {
		t.Fatal(err)
	}
	dirs, err := database.ListWorkDirs()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 0 {
		t.Fatalf("空白 work_dir 不應寫入清單，got=%+v", dirs)
	}
}

func TestWorkDirCatalog_TrimAndDedupe(t *testing.T) {
	database, err := Open(t.TempDir() + "/workdir_trim.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	s, err := database.CreateSession("t", "", "  /tmp/proj-b  ", "default", "claude", nil, "agent")
	if err != nil {
		t.Fatal(err)
	}
	if s.WorkDir != "/tmp/proj-b" {
		t.Fatalf("應 trim，WorkDir=%q", s.WorkDir)
	}
	if _, err := database.CreateSession("t2", "", "/tmp/proj-b", "default", "claude", nil, "agent"); err != nil {
		t.Fatal(err)
	}
	dirs, err := database.ListWorkDirs()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 1 || dirs[0] != "/tmp/proj-b" {
		t.Fatalf("同一路徑只應一列，got=%+v", dirs)
	}
}

func TestWorkDirCatalog_SeedOnReopen(t *testing.T) {
	path := t.TempDir() + "/workdir_seed.db"
	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := database.CreateSession("t", "", "/tmp/seed-me", "default", "claude", nil, "agent"); err != nil {
		database.Close()
		t.Fatal(err)
	}
	if _, err := database.Exec(`DELETE FROM work_dirs`); err != nil {
		database.Close()
		t.Fatal(err)
	}
	database.Close()

	database, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	dirs, err := database.ListWorkDirs()
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) != 1 || dirs[0] != "/tmp/seed-me" {
		t.Fatalf("重開應從 session 補回清單，got=%+v", dirs)
	}
}
