package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateDataDir(t *testing.T) {
	dir := t.TempDir()
	cwd := t.TempDir()
	old := filepath.Join(cwd, "tasky.db")
	if err := os.WriteFile(old, []byte("tasky-db-data"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	if err := migrateDataDir(dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "tasky.db"))
	if err != nil {
		t.Fatalf("база не перенесена: %v", err)
	}
	if string(got) != "tasky-db-data" {
		t.Errorf("содержимое базы: %q", got)
	}
	if _, err := os.Stat(old); err != nil {
		t.Error("исходная база в рабочем каталоге удалена")
	}

	// повторный вызов: база уже на месте, ошибки нет
	if err := migrateDataDir(dir); err != nil {
		t.Errorf("повторный вызов: %v", err)
	}
}

func TestMigrateDataDirSkipsWhenTargetExists(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "tasky.db")
	os.WriteFile(dst, []byte("new"), 0o644)
	cwd := t.TempDir()
	os.WriteFile(filepath.Join(cwd, "tasky.db"), []byte("old"), 0o644)
	t.Chdir(cwd)

	if err := migrateDataDir(dir); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "new" {
		t.Errorf("существующая база перезаписана: %q", got)
	}
}
