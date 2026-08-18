package client

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/server"
	"github.com/detrenasama/tasky/internal/store"
)

// startServer поднимает gRPC-сервер на unix-сокете во временном каталоге и
// возвращает клиента и функцию очистки.
func startServer(t *testing.T) (*Client, func()) {
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

	cl, err := Dial(sp)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	return cl, func() {
		cl.Close()
		gs.GracefulStop()
		conn.Close()
	}
}

func TestClientFullFlow(t *testing.T) {
	cl, done := startServer(t)
	defer done()

	// проект
	p, err := cl.CreateProject("Проект")
	if err != nil {
		t.Fatal(err)
	}
	ps, err := cl.Projects()
	if err != nil || len(ps) != 1 || ps[0].Name != "Проект" {
		t.Fatalf("Projects: %v %+v", err, ps)
	}

	// задача
	tk, err := cl.CreateTask(p.ID, "Задача")
	if err != nil {
		t.Fatal(err)
	}
	if err := cl.UpdateTaskTitle(tk.ID, "Задача 2"); err != nil {
		t.Fatal(err)
	}
	if err := cl.UpdateTaskDescription(tk.ID, "описание задачи"); err != nil {
		t.Fatal(err)
	}
	desc, err := cl.TaskDescription(tk.ID)
	if err != nil || desc != "описание задачи" {
		t.Fatalf("TaskDescription: %v %q", err, desc)
	}
	ts, err := cl.TasksByProject(p.ID)
	if err != nil || len(ts) != 1 || ts[0].Title != "Задача 2" {
		t.Fatalf("TasksByProject: %v %+v", err, ts)
	}

	// подзадача + журнал
	st, err := cl.CreateSubtask(tk.ID, "Подзадача")
	if err != nil {
		t.Fatal(err)
	}
	if err := cl.UpdateSubtaskDescription(st.ID, "под-описание"); err != nil {
		t.Fatal(err)
	}
	je, err := cl.CreateJournalEntry(st.ID, "запись журнала")
	if err != nil {
		t.Fatal(err)
	}
	jes, err := cl.JournalEntries(st.ID)
	if err != nil || len(jes) != 1 || jes[0].ID != je.ID {
		t.Fatalf("JournalEntries: %v %+v", err, jes)
	}
	ss, err := cl.SubtasksWithTime(tk.ID)
	if err != nil || len(ss) != 1 || ss[0].Title != "Подзадача" {
		t.Fatalf("SubtasksWithTime: %v %+v", err, ss)
	}

	// учёт времени
	now := time.Now()
	if err := cl.StartSession(st.ID, now); err != nil {
		t.Fatal(err)
	}
	run, err := cl.RunningSession()
	if err != nil || run == nil || run.ID != st.ID {
		t.Fatalf("RunningSession: %v %+v", err, run)
	}
	if err := cl.StopSession(st.ID, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	entries, err := cl.TimeEntriesBySubtask(st.ID)
	if err != nil || len(entries) != 1 {
		t.Fatalf("TimeEntriesBySubtask: %v %+v", err, entries)
	}
	tt, err := cl.TodayTotal(now)
	if err != nil || tt < 2*time.Minute {
		t.Fatalf("TodayTotal: %v %v", err, tt)
	}

	// статусы
	if err := cl.SetStatus(db.OwnerSubtask, st.ID, "В работе", "взялся", now); err != nil {
		t.Fatal(err)
	}
	hist, err := cl.StatusHistory(db.OwnerSubtask, st.ID)
	if err != nil || len(hist) != 1 || hist[0].To != "В работе" {
		t.Fatalf("StatusHistory: %v %+v", err, hist)
	}

	// теги
	tts, err := cl.TagTypes()
	if err != nil || len(tts) == 0 {
		t.Fatalf("TagTypes (сид): %v %+v", err, tts)
	}
	tag, err := cl.CreateTag(tk.ID, tts[0].ID, "TASK-42", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	tags, err := cl.TaskTags(tk.ID)
	if err != nil || len(tags) != 1 || tags[0].ID != tag.ID {
		t.Fatalf("TaskTags: %v %+v", err, tags)
	}
	byProj, err := cl.TagsByProject(p.ID)
	if err != nil || len(byProj[tk.ID]) != 1 {
		t.Fatalf("TagsByProject: %v %+v", err, byProj)
	}

	// отчёты
	rep, err := cl.ReportEntries(now.Add(-time.Hour), now.Add(time.Hour), 0)
	if err != nil || len(rep) != 1 || rep[0].Seconds < 120 {
		t.Fatalf("ReportEntries: %v %+v", err, rep)
	}

	// ссылки
	l, err := cl.CreateProjectLink(p.ID, "сайт", "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	links, err := cl.ProjectLinks(p.ID)
	if err != nil || len(links) != 1 || links[0].ID != l.ID {
		t.Fatalf("ProjectLinks: %v %+v", err, links)
	}

	// настройки
	if err := cl.SetSetting("theme", "classic"); err != nil {
		t.Fatal(err)
	}
	v, ok, err := cl.GetSetting("theme")
	if err != nil || !ok || v != "classic" {
		t.Fatalf("GetSetting: %v %v %q", err, ok, v)
	}

	// удаление каскадом
	if err := cl.DeleteProject(p.ID); err != nil {
		t.Fatal(err)
	}
	ps, err = cl.Projects()
	if err != nil || len(ps) != 0 {
		t.Fatalf("Projects after delete: %v %+v", err, ps)
	}
}

func TestDialNoServer(t *testing.T) {
	sp := filepath.Join(t.TempDir(), "nope.sock")
	if _, err := Dial(sp); err == nil {
		t.Fatal("Dial без сервера должен падать")
	}
}

func TestListenAlreadyRunning(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "tasky.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	sp := filepath.Join(t.TempDir(), "tasky.sock")
	lis, err := server.Listen(sp)
	if err != nil {
		t.Fatal(err)
	}
	gs := server.Serve(lis, store.NewSQLite(conn))
	defer gs.GracefulStop()

	// второй Listen на живом сокете — ErrAlreadyRunning
	if _, err := server.Listen(sp); err == nil {
		t.Fatal("второй Listen должен возвращать ErrAlreadyRunning")
	}
}

func TestStatusInUseThroughRPC(t *testing.T) {
	cl, done := startServer(t)
	defer done()

	// задача со статусом по умолчанию — «Новая» становится используемой
	p, err := cl.CreateProject("П")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cl.CreateTask(p.ID, "З"); err != nil {
		t.Fatal(err)
	}
	sts, err := cl.Statuses()
	if err != nil || len(sts) == 0 {
		t.Fatalf("Statuses: %v %+v", err, sts)
	}
	// попытка удалить используемый статус должна вернуть db.ErrStatusInUse
	if err := cl.DeleteStatus(sts[0].ID); err != db.ErrStatusInUse {
		t.Fatalf("DeleteStatus: %v (ожидался ErrStatusInUse)", err)
	}
}
