package web

import (
	"net/http"

	"github.com/detrenasama/tasky/internal/store"
)

// registerProjects — проекты и их ссылки.
func registerProjects(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/projects", func(w http.ResponseWriter, r *http.Request) {
		list, err := st.Projects()
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/projects", func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			Name string `json:"name"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		p, err := st.CreateProject(b.Name)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, p)
	})
	mux.HandleFunc("DELETE /api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteProject(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/projects/{id}/description", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		txt, err := st.ProjectDescription(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]string{"description": txt})
	})
	mux.HandleFunc("PUT /api/projects/{id}/description", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Description string `json:"description"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.UpdateProjectDescription(id, b.Description); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/projects/{id}/links", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.ProjectLinks(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/projects/{id}/links", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		link, err := st.CreateProjectLink(id, b.Name, b.URL)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, link)
	})
	mux.HandleFunc("PUT /api/projectlinks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.UpdateProjectLink(id, b.Name, b.URL); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("DELETE /api/projectlinks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteProjectLink(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
}
