package xdg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDir(t *testing.T) {
	t.Run("TASKY_HOME", func(t *testing.T) {
		t.Setenv("TASKY_HOME", "/tmp/tasky-home")
		t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
		if got := DataDir(); got != "/tmp/tasky-home" {
			t.Errorf("DataDir() = %q", got)
		}
	})
	t.Run("XDG_DATA_HOME", func(t *testing.T) {
		t.Setenv("TASKY_HOME", "")
		t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
		if got := DataDir(); got != filepath.Join("/tmp", "xdg", "tasky") {
			t.Errorf("DataDir() = %q", got)
		}
	})
	t.Run("default", func(t *testing.T) {
		t.Setenv("TASKY_HOME", "")
		t.Setenv("XDG_DATA_HOME", "")
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join(home, ".local", "share", "tasky")
		if got := DataDir(); got != want {
			t.Errorf("DataDir() = %q, want %q", got, want)
		}
	})
}
