package web

import (
	"net/http"

	"github.com/detrenasama/tasky/internal/store"
)

// registerJournal — записи журнала подзадач.
func registerJournal(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/subtasks/{id}/journal", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.JournalEntries(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/subtasks/{id}/journal", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Text string `json:"text"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		e, err := st.CreateJournalEntry(id, b.Text)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, e)
	})
	mux.HandleFunc("PUT /api/journal/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Text string `json:"text"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.UpdateJournalEntry(id, b.Text); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/projects/{id}/journaltexts", func(w http.ResponseWriter, r *http.Request) {
		pid, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		m, err := st.JournalTexts(pid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, m)
	})
}
