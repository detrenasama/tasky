package theme

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTheme(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, "themes", name+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseTheme(t *testing.T) {
	t.Run("full", func(t *testing.T) {
		th, ok := parseTheme([]byte(`{"name":"t","colors":{"accent":"#ff0000"}}`), "file")
		if !ok {
			t.Fatal("parse failed")
		}
		if th.Name != "t" {
			t.Errorf("Name = %q", th.Name)
		}
		if th.Colors.Accent != "#ff0000" {
			t.Errorf("Accent = %q", th.Colors.Accent)
		}
		// не заданные цвета падают на дефолт
		if th.Colors.Muted != defaultColors().Muted {
			t.Errorf("Muted = %q, want default %q", th.Colors.Muted, defaultColors().Muted)
		}
	})
	t.Run("name from file", func(t *testing.T) {
		th, ok := parseTheme([]byte(`{"colors":{"accent":"#00ff00"}}`), "myt")
		if !ok {
			t.Fatal("parse failed")
		}
		if th.Name != "myt" {
			t.Errorf("Name = %q, want myt", th.Name)
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		if _, ok := parseTheme([]byte(`{bad`), "x"); ok {
			t.Error("expected failure")
		}
	})
}

func TestLoadUserThemes(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "mine", `{"colors":{"accent":"#123456"}}`)
	writeTheme(t, dir, "broken", `{oops`)
	got := loadUserThemes(dir)
	if len(got) != 1 {
		t.Fatalf("got %d themes, want 1", len(got))
	}
	if got["mine"].Colors.Accent != "#123456" {
		t.Errorf("accent = %q", got["mine"].Colors.Accent)
	}
}

func TestThemesAndApply(t *testing.T) {
	// встроенные темы из embed
	themes := Themes()
	if len(themes) < 5 {
		t.Errorf("themes = %v, want >= 5 builtin", themes)
	}
	found := map[string]bool{}
	for _, n := range themes {
		found[n] = true
	}
	for _, want := range []string{"opencode", "classic", "nord", "dracula", "one-dark"} {
		if !found[want] {
			t.Errorf("theme %q not listed", want)
		}
	}
	prev := ActiveName()
	if err := Apply("nord"); err != nil {
		t.Fatal(err)
	}
	if ActiveName() != "nord" {
		t.Errorf("ActiveName = %q", ActiveName())
	}
	if active.Colors.Muted == defaultColors().Muted {
		t.Error("active palette not rebuilt")
	}
	if err := Apply(prev); err != nil {
		t.Fatal(err)
	}
	if err := Apply("no-such-theme"); err == nil {
		t.Error("expected error for unknown theme")
	}
}

func TestInit(t *testing.T) {
	dir := t.TempDir()
	writeTheme(t, dir, "mine", `{"colors":{"accent":"#123456"}}`)
	t.Run("user theme from configDir", func(t *testing.T) {
		t.Setenv("TASKY_THEME", "")
		Init(dir, "mine")
		if ActiveName() != "mine" {
			t.Errorf("ActiveName = %q", ActiveName())
		}
		Init(dir, "")
		if ActiveName() != DefaultName {
			t.Errorf("fallback ActiveName = %q", ActiveName())
		}
	})
	t.Run("env override", func(t *testing.T) {
		t.Setenv("TASKY_THEME", "dracula")
		Init(dir, "nord")
		if ActiveName() != "dracula" {
			t.Errorf("env override ActiveName = %q", ActiveName())
		}
	})
	t.Run("unknown falls back", func(t *testing.T) {
		t.Setenv("TASKY_THEME", "")
		Init(dir, "ghost")
		if ActiveName() != DefaultName {
			t.Errorf("ActiveName = %q", ActiveName())
		}
	})
}
