package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

// newTestServer поднимает httptest-сервер поверх временной БД.
func newTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()
	dir := t.TempDir()
	conn, err := db.Open(filepath.Join(dir, "tasky.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	st := store.NewSQLite(conn)
	mux := http.NewServeMux()
	Register(mux, st)
	srv := httptest.NewServer(mux)
	return srv, func() { srv.Close(); conn.Close(); os.RemoveAll(dir) }
}

func doJSON(t *testing.T, srv *httptest.Server, method, path string, body any, out any) int {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, srv.URL+path, rdr)
	if err != nil {
		t.Fatalf("newreq: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if out != nil && resp.StatusCode < 300 {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode %s %s: %v", method, path, err)
		}
	}
	return resp.StatusCode
}

func TestAPIProjectsTasksSubtasks(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	var proj db.Project
	if code := doJSON(t, srv, "POST", "/api/projects", map[string]string{"name": "Проект"}, &proj); code != 200 {
		t.Fatalf("create project: %d", code)
	}
	if proj.ID == 0 || proj.Name != "Проект" {
		t.Fatalf("bad project: %+v", proj)
	}

	var list []db.Project
	if code := doJSON(t, srv, "GET", "/api/projects", nil, &list); code != 200 || len(list) != 1 {
		t.Fatalf("list projects: %d %v", code, list)
	}

	var task db.Task
	if code := doJSON(t, srv, "POST", "/api/projects/"+itoa(proj.ID)+"/tasks", map[string]string{"title": "Задача"}, &task); code != 200 {
		t.Fatalf("create task: %d", code)
	}

	var sub db.SubtaskWithTime
	if code := doJSON(t, srv, "POST", "/api/tasks/"+itoa(task.ID)+"/subtasks", map[string]string{"title": "Подзадача"}, &sub); code != 200 {
		t.Fatalf("create subtask: %d", code)
	}

	// запуск/остановка сессии
	if code := doJSON(t, srv, "POST", "/api/subtasks/"+itoa(sub.ID)+"/start", nil, nil); code != 200 {
		t.Fatalf("start: %d", code)
	}
	var running *db.SubtaskWithTime
	if code := doJSON(t, srv, "GET", "/api/running", nil, &running); code != 200 || running == nil {
		t.Fatalf("running: %d %v", code, running)
	}
	if code := doJSON(t, srv, "POST", "/api/subtasks/"+itoa(sub.ID)+"/stop", nil, nil); code != 200 {
		t.Fatalf("stop: %d", code)
	}

	// смена статуса
	var sts []db.StatusDef
	doJSON(t, srv, "GET", "/api/statuses", nil, &sts)
	if code := doJSON(t, srv, "POST", "/api/status", map[string]any{"owner": "subtask", "id": sub.ID, "to": sts[0].Name}, nil); code != 200 {
		t.Fatalf("set status: %d", code)
	}

	// отчёт за сегодня
	var rep []db.ReportEntry
	if code := doJSON(t, srv, "GET", "/api/reports?from=2000-01-01&to=2100-01-01", nil, &rep); code != 200 {
		t.Fatalf("reports: %d", code)
	}
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

// rawBody выполняет запрос и возвращает тело ответа как строку (без декода).
func rawBody(t *testing.T, srv *httptest.Server, method, path string) string {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("newreq: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	return buf.String()
}

// TestEmptyCollectionsNotNull проверяет, что пустые коллекции кодируются как
// [] / {} (а не null), иначе фронтенд падает на .map/.filter.
func TestEmptyCollectionsNotNull(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	if b := rawBody(t, srv, "GET", "/api/projects"); b != "[]" {
		t.Fatalf("empty projects: got %q, want []", b)
	}

	var proj db.Project
	doJSON(t, srv, "POST", "/api/projects", map[string]string{"name": "Проект"}, &proj)
	var task db.Task
	doJSON(t, srv, "POST", "/api/projects/"+itoa(proj.ID)+"/tasks", map[string]string{"title": "Задача"}, &task)
	var sub db.SubtaskWithTime
	doJSON(t, srv, "POST", "/api/tasks/"+itoa(task.ID)+"/subtasks", map[string]string{"title": "Подзадача"}, &sub)

	checks := []struct {
		path string
		want string
	}{
		{"/api/subtasks/" + itoa(sub.ID) + "/journal", "[]"},
		{"/api/subtasks/" + itoa(sub.ID) + "/time", "[]"},
		{"/api/subtasks/" + itoa(sub.ID) + "/checklist", "[]"},
		{"/api/subtasks/" + itoa(sub.ID) + "/links", "[]"},
		{"/api/tasks/" + itoa(task.ID) + "/tags", "[]"},
		{"/api/reports?from=2100-01-01&to=2100-01-02", "[]"},
	}
	for _, c := range checks {
		if b := rawBody(t, srv, "GET", c.path); b != c.want {
			t.Fatalf("%s: got %q, want %q", c.path, b, c.want)
		}
	}
}
