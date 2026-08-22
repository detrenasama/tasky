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

func TestDataDirPrepareNoMigrate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TASKY_HOME", dir)

	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "tasky.db"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	got, err := dataDirPrepare()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("возвращённая директория: %q, хотели %q", got, dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "tasky.db")); err == nil {
		t.Error("база из рабочего каталога скопирована в переопределённую директорию")
	}
}
