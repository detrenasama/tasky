package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildHelpText(t *testing.T) {
	text := buildHelpText()
	for _, want := range []string{"tasky upgrade", "tasky --version", "tasky help", "TASKY_HOME", "tasky serve", "tasky attach", "tasky status", "TASKY_SOCKET", "--socket"} {
		if !strings.Contains(text, want) {
			t.Errorf("справка не содержит %q", want)
		}
	}
}

func TestSocketPath(t *testing.T) {
	t.Setenv("TASKY_HOME", "/tmp/th")
	t.Setenv("TASKY_SOCKET", "")

	if got := socketPath(nil); got != "/tmp/th/tasky.sock" {
		t.Errorf("дефолт: %q", got)
	}
	t.Setenv("TASKY_SOCKET", "/env/sock")
	if got := socketPath(nil); got != "/env/sock" {
		t.Errorf("env: %q", got)
	}
	if got := socketPath([]string{"--socket", "/arg/sock"}); got != "/arg/sock" {
		t.Errorf("флаг: %q", got)
	}
	if got := socketPath([]string{"--socket", "/arg/sock", "TASKY_SOCKET=/env/sock"}); got != "/arg/sock" {
		t.Errorf("флаг приоритетнее env: %q", got)
	}
}

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
