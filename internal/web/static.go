package web

import (
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

// placeholderHTML — страница-заглушка, когда фронтенд не собран (web/dist
// пуст). Собирается в бинарь вместе с embed, чтобы не ломать компиляцию.
const placeholderHTML = `<!doctype html><html lang="ru"><head><meta charset="utf-8">` +
	`<title>Tasky</title></head><body style="font-family:sans-serif;background:#f4f6fa;color:#232733;max-width:40rem;margin:3rem auto;padding:0 1rem">` +
	`<h1 style="color:#3b6fd4">Tasky</h1>` +
	`<p>Веб-интерфейс не собран. Выполните <code>just web-build</code> ` +
	`(или <code>npm run build</code> в каталоге <code>web/</code>) и пересоберите бинарник.</p>` +
	`</body></html>`

// RegisterStatic отдаёт собранный фронтенд (web/dist) по корню «/».
// SPA-фоллбэк: маршруты без расширения, для которых нет файла, отдают
// index.html. Не перехватывает /api/* и /status — те зарегистрированы ранее
// и имеют приоритет в http.ServeMux.
func RegisterStatic(mux *http.ServeMux, webFS fs.FS) {
	if webFS == nil {
		mux.Handle("/", placeholder())
		return
	}
	sub, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		mux.Handle("/", placeholder())
		return
	}
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Читаем файл по пути; при ошибке и отсутствии расширения —
		// SPA-фоллбэк на index.html; иначе 404.
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		data, err := fs.ReadFile(sub, name)
		if err != nil {
			if !strings.Contains(r.URL.Path, ".") {
				if idx, e2 := fs.ReadFile(sub, "index.html"); e2 == nil {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					_, _ = w.Write(idx)
					return
				}
				placeholder().ServeHTTP(w, r)
				return
			}
			http.NotFound(w, r)
			return
		}
		ct := mime.TypeByExtension(path.Ext(name))
		if ct == "" {
			ct = "application/octet-stream"
		}
		w.Header().Set("Content-Type", ct)
		_, _ = w.Write(data)
	}))
}

func placeholder() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(placeholderHTML))
	})
}
