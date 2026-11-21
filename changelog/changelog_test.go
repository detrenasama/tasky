package changelog

import (
	"strings"
	"testing"
)

const fixture = `# Changelog

## [Unreleased]
- should be ignored

## v0.6.0 — 2026-10-20
- Третье изменение
- Ещё одно

## v0.5.4 - 2026-09-14
- Второе изменение
	- вложенный список игнорируется

## v0.5.3 — 2026-09-01
- Первое изменение
`

func TestParse(t *testing.T) {
	entries := Parse(fixture)
	if len(entries) != 3 {
		t.Fatalf("версий: %d, хочу 3: %+v", len(entries), entries)
	}
	if entries[0].Version != "v0.6.0" || entries[0].Date != "2026-10-20" {
		t.Fatalf("первая запись: %+v", entries[0])
	}
	if len(entries[0].Items) != 2 || entries[0].Items[0] != "Третье изменение" {
		t.Fatalf("пункты v0.6.0: %+v", entries[0].Items)
	}
	if entries[1].Version != "v0.5.4" || entries[1].Date != "2026-09-14" {
		t.Fatalf("вторая запись: %+v", entries[1])
	}
	// Нерелизный заголовок и вложенный пункт не должны попадать в список.
	for _, e := range entries {
		for _, it := range e.Items {
			if strings.Contains(it, "ignored") || strings.Contains(it, "вложенный") {
				t.Fatalf("лишний пункт %q в %s", it, e.Version)
			}
		}
	}
}

func TestParseSortsVersions(t *testing.T) {
	entries := Parse(`## v0.5.0 — 2026-01-01
- a

## v0.10.0 — 2026-02-01
- b

## v0.6.0 — 2026-03-01
- c
`)
	want := []string{"v0.10.0", "v0.6.0", "v0.5.0"}
	for i, w := range want {
		if entries[i].Version != w {
			t.Fatalf("порядок: %v", entries)
		}
	}
}

func TestBetween(t *testing.T) {
	// Between использует встроенный CHANGELOG.md.
	got := Between("v0.5.3", "v0.5.4")
	if len(got) == 0 || got[0].Version != "v0.5.4" {
		t.Fatalf("Between(v0.5.3, v0.5.4) = %+v", got)
	}
	for _, e := range got {
		if e.Version == "v0.5.3" || e.Version == "v0.5.0" {
			t.Fatalf("в диапазон попала старая версия: %+v", got)
		}
	}
	if got := Between("v0.5.4", "v0.5.4"); len(got) != 0 {
		t.Fatalf("пустой диапазон: %+v", got)
	}
	if got := Between("dev", "v0.5.4"); len(got) == 0 {
		t.Fatal("невалидный from должен считаться пустым")
	}
}

func TestCurrent(t *testing.T) {
	got := Current("0.5.4")
	if len(got) != 1 || got[0].Version != "v0.5.4" {
		t.Fatalf("Current(0.5.4) = %+v", got)
	}
	if got := Current("v9.9.9"); len(got) != 0 {
		t.Fatalf("несуществующая версия: %+v", got)
	}
	if got := Current("dev"); len(got) != 0 {
		t.Fatalf("dev: %+v", got)
	}
}

func TestRender(t *testing.T) {
	got := Render([]Entry{
		{Version: "v0.6.0", Date: "2026-10-20", Items: []string{"a", "b"}},
		{Version: "v0.5.4", Items: []string{"c"}},
	})
	for _, want := range []string{"v0.6.0 — 2026-10-20", "• a", "• b", "v0.5.4", "• c"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Render: нет %q в %q", want, got)
		}
	}
	if Render(nil) != "" {
		t.Fatal("Render(nil) должен быть пустым")
	}
}
