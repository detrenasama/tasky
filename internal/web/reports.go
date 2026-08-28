package web

import (
	"net/http"
	"time"

	"github.com/detrenasama/tasky/internal/store"
)

// registerReports — отчёты и связанные выборки.
func registerReports(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/reports", func(w http.ResponseWriter, r *http.Request) {
		from, to, err := rangeParams(r)
		if err != nil {
			writeErr(w, err)
			return
		}
		pid := intQuery(r, "project_id")
		list, err := st.ReportEntries(from, to, pid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("GET /api/reports/journal", func(w http.ResponseWriter, r *http.Request) {
		from, to, err := rangeParams(r)
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.JournalEntriesByRange(from, to)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/reports/tags", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			TaskIDs []int64 `json:"task_ids"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		m, err := st.TagsByTasks(b.TaskIDs)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, m)
	})
}

// rangeParams разбирает from/to из query (RFC3339 или дата 2006-01-02).
func rangeParams(r *http.Request) (time.Time, time.Time, error) {
	parse := func(s string) (time.Time, error) {
		if s == "" {
			return time.Time{}, errBadRequest
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t, nil
		}
		return time.Parse("2006-01-02", s)
	}
	from, err := parse(r.URL.Query().Get("from"))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	to, err := parse(r.URL.Query().Get("to"))
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return from, to, nil
}
