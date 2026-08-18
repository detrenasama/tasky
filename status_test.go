package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/server"
	"github.com/detrenasama/tasky/internal/status"
	"github.com/detrenasama/tasky/internal/store"
)

// startStatusServer поднимает gRPC-сервер на временном сокете и возвращает
// хранилище, путь сокета и функцию очистки.
func startStatusServer(t *testing.T) (*store.SQLite, string, func()) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "tasky.db"))
	if err != nil {
		t.Fatal(err)
	}
	sp := filepath.Join(t.TempDir(), "tasky.sock")
	lis, err := server.Listen(sp)
	if err != nil {
		t.Fatal(err)
	}
	gs := server.Serve(lis, store.NewSQLite(conn))
	return store.NewSQLite(conn), sp, func() {
		gs.GracefulStop()
		conn.Close()
	}
}

// captureStdout запускает f и возвращает всё, что f вывел в stdout.
func captureStdout(f func() int) (string, int) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := f()
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	return string(data), code
}

func TestRunStatus(t *testing.T) {
	st, sp, done := startStatusServer(t)
	defer done()

	p, err := st.CreateProject("Проект")
	if err != nil {
		t.Fatal(err)
	}
	tk, err := st.CreateTask(p.ID, "Задача")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := st.CreateSubtask(tk.ID, "Правки в README")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.StartSession(sub.ID, time.Now().Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}

	t.Setenv("TASKY_SOCKET", sp)
	t.Setenv("TASKY_HOME", "/unused")

	// с запущенной сессией
	out, code := captureStdout(func() int { return runStatus(nil) })
	if code != 0 {
		t.Fatalf("код %d, stdout: %q", code, out)
	}
	var got status.Out
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("JSON: %v, stdout: %q", err, out)
	}
	if got.Subtask == nil || got.Subtask.ID != sub.ID || got.Subtask.Title != "Правки в README" {
		t.Errorf("subtask: %+v", got.Subtask)
	}
	if got.TodaySeconds == 0 {
		t.Error("today_seconds == 0 при активной сессии")
	}

	// после остановки сессии подзадачи нет
	if err := st.StopSession(sub.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	out, code = captureStdout(func() int { return runStatus(nil) })
	if code != 0 {
		t.Fatalf("код %d, stdout: %q", code, out)
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if got.Subtask != nil {
		t.Errorf("subtask не nil после остановки: %+v", got.Subtask)
	}
}

func TestRunStatusServerDown(t *testing.T) {
	// сокета нет — ничего в stdout, код 1
	t.Setenv("TASKY_SOCKET", filepath.Join(t.TempDir(), "missing.sock"))
	t.Setenv("TASKY_HOME", "/unused")

	out, code := captureStdout(func() int { return runStatus(nil) })
	if code != 1 {
		t.Errorf("код = %d, ожидался 1", code)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("stdout не пуст при недоступном сервере: %q", out)
	}
}
