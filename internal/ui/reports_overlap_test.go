package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/detrenasama/tasky/internal/db"
)

func TestReportsShowOverlaps(t *testing.T) {
	conn, _, st := reportsSeedProject(t)
	now := time.Now()
	// reportsSeedProject уже создал запись 2ч назад – 1ч назад.
	// Добавляем пересекающуюся: 1ч30м назад – 30м назад.
	start2 := now.Add(-90 * time.Minute)
	end2 := now.Add(-30 * time.Minute)
	db.StartSession(conn, st.ID, start2)
	db.StopSession(conn, st.ID, end2)

	m := newReportsModel(conn)
	m.reports.load()
	if len(m.reports.overlaps) != 1 {
		t.Fatalf("пересечений: %d, ожидалось 1", len(m.reports.overlaps))
	}
	view := m.reports.view(150, 27)
	if !strings.Contains(view, "Пересечения") {
		t.Error("в отчёте нет секции пересечений")
	}
}

func TestReportsNoOverlapsWhenSeparate(t *testing.T) {
	conn, _, st := reportsSeedProject(t)
	now := time.Now()
	// непересекающаяся запись: 3ч назад – 2ч10м назад (до существующей 2ч–1ч)
	start2 := now.Add(-3 * time.Hour)
	end2 := now.Add(-130 * time.Minute)
	db.StartSession(conn, st.ID, start2)
	db.StopSession(conn, st.ID, end2)

	m := newReportsModel(conn)
	m.reports.load()
	if len(m.reports.overlaps) != 0 {
		t.Errorf("пересечений: %d, ожидалось 0", len(m.reports.overlaps))
	}
}
