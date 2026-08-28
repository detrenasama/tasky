package web

import (
	"net/http"

	"github.com/detrenasama/tasky/internal/store"
)

// registerSettings — простые настройки «ключ → значение».
func registerSettings(mux *http.ServeMux, st store.Store) {
	mux.HandleFunc("GET /api/settings/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		v, _, err := st.GetSetting(key)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, map[string]string{"value": v})
	})
	mux.HandleFunc("PUT /api/settings/{key}", func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		var b struct {
			Value string `json:"value"`
		}
		if err := decodeJSON(r, &b); err != nil {
			writeErr(w, errBadRequest)
			return
		}
		if err := st.SetSetting(key, b.Value); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, okTrue())
	})
	// Кураторский набор настроек, используемых веб-UI (тема, скрытие и т.п.).
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		keys := []string{"theme", "hide_days"}
		out := map[string]string{}
		for _, k := range keys {
			if v, ok, err := st.GetSetting(k); err == nil && ok {
				out[k] = v
			}
		}
		writeJSON(w, out)
	})
}
