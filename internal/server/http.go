package server

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/status"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/web"
)

// ListenHTTP открывает TCP-слушатель для HTTP-эндпоинтов интеграций
// (например, GET /status для GNOME Shell-индикатора). Адрес — host:port;
// по умолчанию 127.0.0.1:9110 (только локально).
func ListenHTTP(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

// ServeHTTP запускает HTTP-сервер интеграций в фоновой горутине. Возвращает
// *http.Server для остановки через Close. webFS — встроенный фронтенд
// (web/dist), отдаваемый по корню; version — версия бинаря для шапки веба.
func ServeHTTP(lis net.Listener, st store.Store, webFS fs.FS, version string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", handleStatus(st))
	// JSON API для веб-интерфейса (зеркало Store).
	web.Register(mux, st)
	// Версия бинаря — для отображения в шапке веб-интерфейса.
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":` + strconv.Quote(version) + `}`))
	})
	// Статика фронтенда (SPA) по корню — регистрируется последней.
	web.RegisterStatic(mux, webFS)
	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(lis); err != nil && err != http.ErrServerClosed {
			// падение HTTP-сервера печатается, но процесс живёт (как и gRPC)
			fmt.Fprintln(os.Stderr, "HTTP-сервер:", err)
		}
	}()
	return srv
}

// handleStatus отдаёт JSON статуса (время за сегодня и запущенная подзадача) —
// тот же формат, что и у команды `tasky status`.
func handleStatus(st store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var today int64
		if d, err := st.TodayTotal(time.Now()); err == nil {
			today = int64(d / time.Second)
		}
		var run *db.SubtaskWithTime
		if x, err := st.RunningSession(); err == nil {
			run = x
		}
		data, err := status.Build(today, run)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}
}
