package web

import (
	"net/http"

	"github.com/detrenasama/tasky/internal/store"
)

// registerChecklist — чек-листы подзадач.
func registerChecklist(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/subtasks/{id}/checklist", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.ChecklistItems(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("GET /api/projects/{id}/checklistcounts", func(w http.ResponseWriter, r *http.Request) {
		pid, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		m, err := st.ChecklistCounts(pid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, m)
	})
	mux.HandleFunc("POST /api/subtasks/{id}/checklist", func(w http.ResponseWriter, r *http.Request) {
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
		it, err := st.CreateChecklistItem(id, b.Text)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, it)
	})
	mux.HandleFunc("PUT /api/checklist/{id}/text", func(w http.ResponseWriter, r *http.Request) {
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
		if err := st.UpdateChecklistItemText(id, b.Text); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("PUT /api/checklist/{id}/status", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Status string `json:"status"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.SetChecklistItemStatus(id, b.Status); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("POST /api/checklist/{id}/move", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Dir int `json:"dir"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.MoveChecklistItem(id, b.Dir); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("DELETE /api/checklist/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteChecklistItem(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
}
