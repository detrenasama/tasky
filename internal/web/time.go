package web

import (
	"net/http"
	"time"

	"github.com/detrenasama/tasky/internal/store"
)

// registerTime — учёт времени и сессии.
func registerTime(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/subtasks/{id}/time", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.TimeEntriesBySubtask(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/subtasks/{id}/start", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.StartSession(id, timeNow()); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("POST /api/subtasks/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.StopSession(id, timeNow()); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/today", func(w http.ResponseWriter, r *http.Request) {
		d, err := st.TodayTotal(timeNow())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]int64{"seconds": int64(d / 1e9)})
	})
	mux.HandleFunc("GET /api/weekly", func(w http.ResponseWriter, r *http.Request) {
		d, err := st.WeeklyTotal(timeNow())
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]int64{"seconds": int64(d / 1e9)})
	})
	mux.HandleFunc("GET /api/running", func(w http.ResponseWriter, r *http.Request) {
		run, err := st.RunningSession()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, run)
	})
	mux.HandleFunc("PUT /api/timeentries/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			StartedAt string  `json:"started_at"`
			EndedAt   *string `json:"ended_at"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		started, err := time.Parse(time.RFC3339, b.StartedAt)
		if err != nil {
			writeErr(w, errBadRequest)
			return
		}
		var ended *time.Time
		if b.EndedAt != nil {
			e, err := time.Parse(time.RFC3339, *b.EndedAt)
			if err != nil {
				writeErr(w, errBadRequest)
				return
			}
			ended = &e
		}
		if err := st.UpdateTimeEntry(id, started, ended); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("DELETE /api/timeentries/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteTimeEntry(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
}
