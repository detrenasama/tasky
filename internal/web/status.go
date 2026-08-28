package web

import (
	"net/http"

	"github.com/detrenasama/tasky/internal/store"
)

// registerStatus — каталог статусов, смена и история.
func registerStatus(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/statuses", func(w http.ResponseWriter, r *http.Request) {
		list, err := st.Statuses()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/statuses", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Name  string `json:"name"`
			Type  string `json:"type"`
			Color string `json:"color"`
			Note  string `json:"note_prompt"`
			Quick bool   `json:"is_quick"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		s, err := st.CreateStatus(b.Name, b.Type, b.Color, b.Note, b.Quick)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, s)
	})
	mux.HandleFunc("PUT /api/statuses/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Name  string `json:"name"`
			Type  string `json:"type"`
			Color string `json:"color"`
			Note  string `json:"note_prompt"`
			Quick bool   `json:"is_quick"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.UpdateStatus(id, b.Name, b.Type, b.Color, b.Note, b.Quick); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("DELETE /api/statuses/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteStatus(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("POST /api/status", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Owner string `json:"owner"`
			ID    int64  `json:"id"`
			To    string `json:"to"`
			Note  string `json:"note"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		owner, err := ownerFromStr(b.Owner)
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.SetStatus(owner, b.ID, b.To, b.Note, timeNow()); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/status/history/{owner}/{id}", func(w http.ResponseWriter, r *http.Request) {
		owner, err := ownerFromStr(r.PathValue("owner"))
		if err != nil {
			writeErr(w, err)
			return
		}
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.StatusHistory(owner, id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
}
