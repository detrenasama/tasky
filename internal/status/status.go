// Package status — JSON-представление статуса Tasky для внешних интеграций
// (GNOME Shell-индикатор и подобные). Общий источник правды для команды
// `tasky status` и HTTP-эндпоинта GET /status.
package status

import (
	"encoding/json"

	"github.com/detrenasama/tasky/internal/db"
)

// Subtask — запущенная подзадача.
type Subtask struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// Out — время за сегодня и запущенная подзадача (nil, если учёт не идёт).
type Out struct {
	TodaySeconds int64    `json:"today_seconds"`
	Subtask      *Subtask `json:"subtask"`
}

// Build собирает JSON статуса: todaySeconds — суммарное время за сегодня,
// run — активная сессия (nil, если учёт времени не идёт).
func Build(todaySeconds int64, run *db.SubtaskWithTime) ([]byte, error) {
	out := Out{TodaySeconds: todaySeconds}
	if run != nil {
		out.Subtask = &Subtask{ID: run.ID, Title: run.Title}
	}
	return json.Marshal(out)
}
