package server

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/status"
	"github.com/detrenasama/tasky/internal/store"
)

// startHTTP поднимает HTTP-сервер на случайном порту и возвращает хранилище,
// адрес и функцию очистки.
func startHTTP(t *testing.T) (*store.SQLite, string, func()) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "tasky.db"))
	if err != nil {
		t.Fatal(err)
	}
	st := store.NewSQLite(conn)
	lis, err := ListenHTTP("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	hs := ServeHTTP(lis, st, nil)
	return st, lis.Addr().String(), func() {
		hs.Close()
		conn.Close()
	}
}

func getStatus(t *testing.T, addr string) (*status.Out, int) {
	t.Helper()
	resp, err := http.Get("http://" + addr + "/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out status.Out
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("JSON: %v, body: %q", err, body)
	}
	return &out, resp.StatusCode
}

func TestHTTPStatus(t *testing.T) {
	st, addr, done := startHTTP(t)
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

	// с запущенной сессией
	out, code := getStatus(t, addr)
	if code != http.StatusOK {
		t.Errorf("code = %d, ожидался 200", code)
	}
	if out.Subtask == nil || out.Subtask.ID != sub.ID || out.Subtask.Title != "Правки в README" {
		t.Errorf("subtask: %+v", out.Subtask)
	}
	if out.TodaySeconds == 0 {
		t.Error("today_seconds == 0 при активной сессии")
	}

	// после остановки сессии подзадачи нет
	if err := st.StopSession(sub.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	out, _ = getStatus(t, addr)
	if out.Subtask != nil {
		t.Errorf("subtask после остановки: %+v", out.Subtask)
	}
}

func TestHTTPStatusMethodNotAllowed(t *testing.T) {
	_, addr, done := startHTTP(t)
	defer done()

	resp, err := http.Post("http://"+addr+"/status", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST code = %d, ожидался 405", resp.StatusCode)
	}
}

func TestHTTPStatusNotFound(t *testing.T) {
	_, addr, done := startHTTP(t)
	defer done()

	// Неизвестный путь без расширения отдаёт SPA-оболочку (заглушку, т.к.
	// фронтенд в тестах не собран) — 200, а не 404.
	resp, err := http.Get("http://" + addr + "/nope")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("code = %d, ожидался 200 (SPA-фоллбэк)", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Веб-интерфейс не собран") {
		t.Errorf("ожидалась заглушка фронтенда, получено: %q", string(body))
	}
}
