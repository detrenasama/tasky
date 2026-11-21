// Package changelog содержит встроенный список изменений Tasky по версиям.
//
// Канонический файл — changelog/CHANGELOG.md; он встраивается в бинарник,
// поэтому список доступен без сети и в любой момент. Из этого же файла
// scripts/release.sh берёт заметки для GitHub-релиза.
package changelog

import (
	_ "embed"
	"regexp"
	"sort"
	"strings"

	"github.com/detrenasama/tasky/internal/update"
)

//go:embed CHANGELOG.md
var markdown string

// Entry — изменения одной версии.
type Entry struct {
	Version string
	Date    string
	Items   []string
}

// headingRe разбирает заголовки вида «## v0.5.4 — 2026-10-20» и
// «## v0.5.4 - 2026-10-20» (дата необязательна).
var headingRe = regexp.MustCompile(`^##\s+(v?\d+\.\d+\.\d+)(?:\s+[—–-]\s+(.+?))?\s*$`)

// All возвращает все версии из встроенного changelog (новые сверху).
func All() []Entry { return Parse(markdown) }

// Parse разбирает Markdown в список версий. Секции без пунктов пропускаются;
// заголовки нерелизных версий (например, «## [Unreleased]») игнорируются.
func Parse(md string) []Entry {
	var out []Entry
	var cur *Entry

	flush := func() {
		if cur != nil && len(cur.Items) > 0 {
			out = append(out, *cur)
		}
		cur = nil
	}

	for _, raw := range strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if m := headingRe.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Entry{Version: update.Canonical(m[1]), Date: strings.TrimSpace(m[2])}
			continue
		}
		// Любой другой заголовок второго уровня завершает текущую секцию.
		if strings.HasPrefix(line, "## ") {
			flush()
			continue
		}
		if cur == nil {
			continue
		}
		// Пункты верхнего уровня — только без отступа; вложенные списки
		// намеренно не разбираем.
		if strings.HasPrefix(raw, "- ") || strings.HasPrefix(raw, "* ") {
			if item := strings.TrimSpace(raw[2:]); item != "" {
				cur.Items = append(cur.Items, item)
			}
		}
	}
	flush()

	sort.SliceStable(out, func(i, j int) bool {
		return update.Compare(out[i].Version, out[j].Version) > 0
	})
	return out
}

// Current возвращает секцию указанной версии (nil, если её нет).
func Current(v string) []Entry {
	v = update.Canonical(v)
	if v == "" {
		return nil
	}
	for _, e := range All() {
		if update.Compare(e.Version, v) == 0 {
			return []Entry{e}
		}
	}
	return nil
}

// Between возвращает версии в диапазоне (from, to]; from == "" — все версии
// не новее to.
func Between(from, to string) []Entry {
	to = update.Canonical(to)
	if to == "" {
		return nil
	}
	from = update.Canonical(from)
	var out []Entry
	for _, e := range All() {
		if update.Compare(e.Version, to) > 0 {
			continue
		}
		if from != "" && update.Compare(e.Version, from) <= 0 {
			continue
		}
		out = append(out, e)
	}
	return out
}

// Render формирует plain-text тело для модалки «Что нового».
func Render(entries []Entry) string {
	var b strings.Builder
	for i, e := range entries {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(e.Version)
		if e.Date != "" {
			b.WriteString(" — " + e.Date)
		}
		for _, item := range e.Items {
			b.WriteString("\n• " + item)
		}
	}
	return b.String()
}
