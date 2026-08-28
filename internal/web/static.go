package web

import (
	"io/fs"
	"net/http"
	"strings"
)

// placeholderHTML — страница-заглушка, когда фронтенд не собран (web/dist
// пуст). Собирается в бинарь вместе с embed, чтобы не ломать компиляцию.
const placeholderHTML = `<!doctype html><html lang="ru"><head><meta charset="utf-8">` +
	`<title>Tasky</title></head><body style="font-family:sans-serif;max-width:40rem;margin:3rem auto;padding:0 1rem">` +
	`<h1>Tasky</h1>` +
	`<p>Веб-интерфейс не собран. Выполните <code>just web-build</code> ` +
	`(или <code>npm run build</code> в каталоге <code>web/</code>) и пересоберите бинарник.</p>` +
	`</body></html>`

// RegisterStatic отдаёт собранный фронтенд (web/dist) по корню «/».
// SPA-фоллбэк: маршруты без расширения отдают index.html (или заглушку,
// если фронтенд не собран). Не перехватывает /api/* и /status — те
// зарегистрированы ранее и имеют приоритет в http.ServeMux.
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
	fileServer := http.FileServer(http.FS(sub))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Запрос конкретного файла (есть расширение) — отдаём как есть.
		if strings.Contains(r.URL.Path, ".") {
			fileServer.ServeHTTP(w, r)
			return
		}
		// Фронтенд не собран — заглушка.
		if _, err := fs.Stat(sub, "index.html"); err != nil {
			placeholder().ServeHTTP(w, r)
			return
		}
		// SPA-маршрут: если файл по пути не существует — отдаём index.html.
		f, err := sub.Open(strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil {
			r.URL.Path = "/index.html"
		} else {
			_ = f.Close()
		}
		fileServer.ServeHTTP(w, r)
	}))
}

func placeholder() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(placeholderHTML))
	})
}
