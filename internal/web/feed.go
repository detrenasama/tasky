package web

import (
	"net/http"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

func registerTaskDates(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("PUT /api/tasks/{id}/dates", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			DueDate      *string `json:"due_date"`
			ScheduleDate *string `json:"schedule_date"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		due, err := parseWebDate(b.DueDate)
		if err != nil {
			writeErr(w, err)
			return
		}
		sched, err := parseWebDate(b.ScheduleDate)
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.SetTaskDates(id, due, sched); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("PUT /api/subtasks/{id}/dates", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			DueDate      *string `json:"due_date"`
			ScheduleDate *string `json:"schedule_date"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		due, err := parseWebDate(b.DueDate)
		if err != nil {
			writeErr(w, err)
			return
		}
		sched, err := parseWebDate(b.ScheduleDate)
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.SetSubtaskDates(id, due, sched); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
}

func parseWebDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		// try DD.MM.YYYY
		t2, err2 := time.Parse("02.01.2006", *s)
		if err2 != nil {
			return nil, errBadRequest
		}
		t = t2
	}
	nd := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return &nd, nil
}

func registerFeed(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/feed", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		f := db.FeedFilter{
			Kind:            q.Get("kind"),
			UnreadOnly:      q.Get("unread_only") == "1" || q.Get("unread_only") == "true",
			IncludeArchived: q.Get("archived") == "1" || q.Get("include_archived") == "1",
		}
		items, err := st.FeedItems(f)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, items)
	})
	mux.HandleFunc("POST /api/feed/{id}/read", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.MarkFeedRead(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/feed/unread_count", func(w http.ResponseWriter, r *http.Request) {
		n, err := st.FeedUnreadCount()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]int{"count": n})
	})
	mux.HandleFunc("POST /api/feed/archive", func(w http.ResponseWriter, r *http.Request) {
		n, err := st.ArchiveFeed()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]int{"archived": n})
	})
	mux.HandleFunc("DELETE /api/feed/archived", func(w http.ResponseWriter, r *http.Request) {
		n, err := st.DeleteArchivedFeed()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]int{"deleted": n})
	})
}
