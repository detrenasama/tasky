package main

import (
	"fmt"
	"os"
	"time"

	"github.com/detrenasama/tasky/internal/client"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/status"
)

// runStatus — команда tasky status: данные для внешних индикаторов (например,
// GNOME Shell). При недоступном сервере ничего не печатает в stdout и
// возвращает 1 — индикатор просто ничего не отображает.
func runStatus(args []string) int {
	cl, err := client.Dial(socketPath(args))
	if err != nil {
		return 1
	}
	defer cl.Close()

	var today int64
	if d, err := cl.TodayTotal(time.Now()); err == nil {
		today = int64(d.Seconds())
	}
	var run *db.SubtaskWithTime
	if r, err := cl.RunningSession(); err == nil {
		run = r
	}
	data, err := status.Build(today, run)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tasky status: %v\n", err)
		return 1
	}
	fmt.Println(string(data))
	return 0
}
