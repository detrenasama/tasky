package web

import (
	"net/http"

	"github.com/detrenasama/tasky/internal/store"
)

// registerTags — типы тегов и теги задач.
func registerTags(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/tagtypes", func(w http.ResponseWriter, r *http.Request) {
		list, err := st.TagTypes()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/tagtypes", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Name  string `json:"name"`
			Kind  string `json:"kind"`
			Color string `json:"color"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		t, err := st.CreateTagType(b.Name, b.Kind, b.Color)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, t)
	})
	mux.HandleFunc("PUT /api/tagtypes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Name  string `json:"name"`
			Kind  string `json:"kind"`
			Color string `json:"color"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.UpdateTagType(id, b.Name, b.Kind, b.Color); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("DELETE /api/tagtypes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteTagType(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/tasks/{id}/tags", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.TaskTags(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("GET /api/projects/{id}/tags", func(w http.ResponseWriter, r *http.Request) {
		pid, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		m, err := st.TagsByProject(pid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, m)
	})
	mux.HandleFunc("POST /api/tasks/{id}/tags", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			TypeID int64  `json:"type_id"`
			Text   string `json:"text"`
			URL    string `json:"url"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		tag, err := st.CreateTag(id, b.TypeID, b.Text, b.URL)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, tag)
	})
	mux.HandleFunc("PUT /api/tags/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			TypeID int64  `json:"type_id"`
			Text   string `json:"text"`
			URL    string `json:"url"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.UpdateTag(id, b.TypeID, b.Text, b.URL); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("DELETE /api/tags/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteTag(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
}
