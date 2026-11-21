package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

func newChangelogTestModel(t *testing.T, version, seen string) (*model, store.Store) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	st := store.NewSQLite(conn)
	if seen != "" {
		if err := st.SetSetting(changelogSeenKey, seen); err != nil {
			t.Fatal(err)
		}
	}
	return New(st, t.TempDir(), version), st
}

func resizeModel(t *testing.T, m *model, w, h int) model {
	t.Helper()
	mm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	got, ok := mm.(model)
	if !ok {
		t.Fatalf("Update вернул %T", mm)
	}
	return got
}

func TestChangelogShowsBetweenVersions(t *testing.T) {
	m, st := newChangelogTestModel(t, "v0.5.4", "v0.5.3")
	got := resizeModel(t, m, 120, 40)
	if !got.modal.Active() {
		t.Fatal("модалка «Что нового» не открылась")
	}
	view := stripANSI(got.modal.View())
	for _, want := range []string{"Что нового в Tasky v0.5.4", "v0.5.4 — 2026-10-20", "v0.5.4"} {
		if !strings.Contains(view, want) {
			t.Fatalf("в модалке нет %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "v0.5.3 —") {
		t.Fatalf("показана уже просмотренная версия:\n%s", view)
	}

	mm, _ := got.Update(tea.KeyMsg{Type: tea.KeyEnter})
	closed := mm.(model)
	if closed.modal.Active() {
		t.Fatal("модалка не закрылась по Enter")
	}
	if seen, ok, _ := st.GetSetting(changelogSeenKey); !ok || seen != "v0.5.4" {
		t.Fatalf("seen_version = %q, %v", seen, ok)
	}
}

func TestChangelogFirstRunShowsCurrentOnly(t *testing.T) {
	m, _ := newChangelogTestModel(t, "v0.5.4", "")
	got := resizeModel(t, m, 120, 40)
	if !got.modal.Active() {
		t.Fatal("при первом запуске модалка не открылась")
	}
	view := stripANSI(got.modal.View())
	if !strings.Contains(view, "v0.5.4") {
		t.Fatalf("нет текущей версии:\n%s", view)
	}
	if strings.Contains(view, "v0.5.3 —") || strings.Contains(view, "v0.5.0 —") {
		t.Fatalf("при первом запуске показана история:\n%s", view)
	}
}

func TestChangelogSameVersionNoModal(t *testing.T) {
	m, _ := newChangelogTestModel(t, "v0.5.4", "v0.5.4")
	got := resizeModel(t, m, 120, 40)
	if got.modal.Active() {
		t.Fatal("модалка открылась для уже просмотренной версии")
	}
}

func TestChangelogDevNoModal(t *testing.T) {
	m, st := newChangelogTestModel(t, "dev", "")
	got := resizeModel(t, m, 120, 40)
	if got.modal.Active() {
		t.Fatal("модалка открылась для dev-сборки")
	}
	if _, ok, _ := st.GetSetting(changelogSeenKey); ok {
		t.Fatal("dev-сборка записала seen_version")
	}
}

func TestChangelogDowngradeSilent(t *testing.T) {
	m, st := newChangelogTestModel(t, "v0.5.4", "v0.6.0")
	got := resizeModel(t, m, 120, 40)
	if got.modal.Active() {
		t.Fatal("модалка открылась при откате версии")
	}
	if seen, ok, _ := st.GetSetting(changelogSeenKey); !ok || seen != "v0.5.4" {
		t.Fatalf("seen_version = %q, %v", seen, ok)
	}
}

func TestChangelogMissingSectionSilent(t *testing.T) {
	m, st := newChangelogTestModel(t, "v9.9.9", "")
	got := resizeModel(t, m, 120, 40)
	if got.modal.Active() {
		t.Fatal("модалка открылась без секции changelog")
	}
	if seen, ok, _ := st.GetSetting(changelogSeenKey); !ok || seen != "v9.9.9" {
		t.Fatalf("seen_version = %q, %v", seen, ok)
	}
}

func TestChangelogPaletteShowsAllVersions(t *testing.T) {
	m, st := newChangelogTestModel(t, "v0.5.4", "v0.5.4")
	m.width, m.height = 120, 40

	found := false
	for _, g := range m.paletteGroups() {
		for _, c := range g.cmds {
			if c.title == "Что нового" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("в палитре нет команды «Что нового»")
	}

	m.openCurrentChangelog()
	if !m.modal.Active() {
		t.Fatal("команда «Что нового» не открыла модалку")
	}
	view := stripANSI(m.modal.View())
	if !strings.Contains(view, "Что нового в Tasky") {
		t.Fatalf("нет заголовка истории:\n%s", view)
	}
	for _, want := range []string{"v0.5.4", "v0.5.3"} {
		if !strings.Contains(view, want) {
			t.Fatalf("в ручном changelog нет %q:\n%s", want, view)
		}
	}
	// Версия новее текущей не должна попадать даже при ручном просмотре.
	if strings.Contains(view, "v0.6.0") {
		t.Fatalf("показана будущая версия:\n%s", view)
	}
	if seen, ok, _ := st.GetSetting(changelogSeenKey); !ok || seen != "v0.5.4" {
		t.Fatalf("ручной просмотр изменил seen_version: %q, %v", seen, ok)
	}
}

func TestChangelogPaletteDevShowsAll(t *testing.T) {
	m, _ := newChangelogTestModel(t, "dev", "")
	m.width, m.height = 120, 40

	m.openCurrentChangelog()
	if !m.modal.Active() {
		t.Fatal("команда «Что нового» не открыла модалку")
	}
	view := stripANSI(m.modal.View())
	if !strings.Contains(view, "v0.6.0") {
		t.Fatalf("dev-сборка не показала весь changelog:\n%s", view)
	}
}
