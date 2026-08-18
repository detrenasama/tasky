package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/detrenasama/tasky/internal/client"
)

// statusSubtask — данные запущенной подзадачи для внешних индикаторов.
type statusSubtask struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// statusOut — JSON для команды tasky status: время за сегодня и запущенная
// подзадача (nil, если учёт времени не идёт).
type statusOut struct {
	TodaySeconds int64          `json:"today_seconds"`
	Subtask      *statusSubtask `json:"subtask"`
}

// runStatus — команда tasky status: данные для внешних индикаторов (например,
// GNOME Shell). При недоступном сервере ничего не печатает в stdout и
// возвращает 1 — индикатор просто ничего не отображает.
func runStatus(args []string) int {
	cl, err := client.Dial(socketPath(args))
	if err != nil {
		return 1
	}
	defer cl.Close()

	out := statusOut{}
	if today, err := cl.TodayTotal(time.Now()); err == nil {
		out.TodaySeconds = int64(today.Seconds())
	}
	if run, err := cl.RunningSession(); err == nil && run != nil {
		out.Subtask = &statusSubtask{ID: run.ID, Title: run.Title}
	}
	data, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tasky status: %v\n", err)
		return 1
	}
	fmt.Println(string(data))
	return 0
}
