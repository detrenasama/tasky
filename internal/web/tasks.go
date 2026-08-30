package web

import (
	"net/http"

	"github.com/detrenasama/tasky/internal/store"
)

// registerTasks — задачи, подзадачи и их ссылки.
func registerTasks(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/projects/{id}/tasks", func(w http.ResponseWriter, r *http.Request) {
		pid, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.TasksByProject(pid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("GET /api/projects/{id}/subtasks", func(w http.ResponseWriter, r *http.Request) {
		pid, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.SubtasksByProject(pid)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/projects/{id}/tasks", func(w http.ResponseWriter, r *http.Request) {
		pid, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Title string `json:"title"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		t, err := st.CreateTask(pid, b.Title)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, t)
	})
	mux.HandleFunc("DELETE /api/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteTask(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("PUT /api/tasks/{id}/title", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Title string `json:"title"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.UpdateTaskTitle(id, b.Title); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/tasks/{id}/description", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		txt, err := st.TaskDescription(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]string{"description": txt})
	})
	mux.HandleFunc("PUT /api/tasks/{id}/description", func(w http.ResponseWriter, r *http.Request) {
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
		if err := st.UpdateTaskDescription(id, b.Description); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("POST /api/tasks/{id}/move", func(w http.ResponseWriter, r *http.Request) {
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
		if err := st.MoveTask(id, b.Dir); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("POST /api/tasks/{id}/reorder", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			To int `json:"to"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.ReorderTask(id, b.To); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/tasks/{id}/subtasks", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.SubtasksWithTime(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/tasks/{id}/subtasks", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Title string `json:"title"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		sub, err := st.CreateSubtask(id, b.Title)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, sub)
	})
	mux.HandleFunc("GET /api/tasks/{id}/links", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.TaskLinks(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/tasks/{id}/links", func(w http.ResponseWriter, r *http.Request) {
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
		link, err := st.CreateTaskLink(id, b.Name, b.URL)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, link)
	})
	mux.HandleFunc("PUT /api/tasklinks/{id}", func(w http.ResponseWriter, r *http.Request) {
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
		if err := st.UpdateTaskLink(id, b.Name, b.URL); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("DELETE /api/tasklinks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteTaskLink(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})

	// Подзадачи.
	mux.HandleFunc("DELETE /api/subtasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteSubtask(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("PUT /api/subtasks/{id}/title", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			Title string `json:"title"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.UpdateSubtaskTitle(id, b.Title); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/subtasks/{id}/description", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		txt, err := st.SubtaskDescription(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]string{"description": txt})
	})
	mux.HandleFunc("PUT /api/subtasks/{id}/description", func(w http.ResponseWriter, r *http.Request) {
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
		if err := st.UpdateSubtaskDescription(id, b.Description); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("POST /api/subtasks/{id}/move", func(w http.ResponseWriter, r *http.Request) {
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
		if err := st.MoveSubtask(id, b.Dir); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("POST /api/subtasks/{id}/reorder", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		var b struct {
			To       int   `json:"to"`
			TaskID   int64 `json:"task_id"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.ReorderSubtask(id, b.TaskID, b.To); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("GET /api/subtasks/{id}/links", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		list, err := st.SubtaskLinks(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, list)
	})
	mux.HandleFunc("POST /api/subtasks/{id}/links", func(w http.ResponseWriter, r *http.Request) {
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
		link, err := st.CreateSubtaskLink(id, b.Name, b.URL)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, link)
	})
	mux.HandleFunc("PUT /api/sublinks/{id}", func(w http.ResponseWriter, r *http.Request) {
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
		if err := st.UpdateSubtaskLink(id, b.Name, b.URL); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	mux.HandleFunc("DELETE /api/sublinks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := idParam(r, "id")
		if err != nil {
			writeErr(w, err)
			return
		}
		if err := st.DeleteSubtaskLink(id); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
}
