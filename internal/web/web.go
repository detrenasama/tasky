// Package web реализует JSON API поверх store.Store для веб-интерфейса.
// Роутинг использует нативный http.ServeMux Go 1.22+ (метод + {id}-паттерны).
// Эндпоинты зеркалят методы Store; единая точка доступа — тот же Store, что у
// TUI (gRPC), поэтому поведение (в т.ч. слияние истории статусов) идентично.
package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

// Register добавляет JSON API-эндпоинты в переданный mux. Вызывается из
// internal/server рядом с существующим /status.
func Register(mux *http.ServeMux, st store.Store) {
	registerProjects(mux, st)
	registerTasks(mux, st)
	registerTime(mux, st)
	registerStatus(mux, st)
	registerTags(mux, st)
	registerJournal(mux, st)
	registerChecklist(mux, st)
	registerReports(mux, st)
	registerSettings(mux, st)

	// Поллинг статуса (время за сегодня + активная сессия) — аналог /status,
	// но в том же формате, что и остальной API.
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		now := timeNow()
		var today int64
		if d, err := st.TodayTotal(now); err == nil {
			today = int64(d / 1e9)
		}
		run, _ := st.RunningSession()
		writeJSON(w, map[string]any{
			"today_seconds": today,
			"running":       run,
		})
	})
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
	})
}

// timeNow — текущее время (вынесено для переопределения в тестах).
var timeNow = time.Now

// writeJSON отправляет 200 + JSON.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// decodeJSON читает тело запроса в dst (неизвестные поля игнорируются).
func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// writeErr маппит ошибку в HTTP-статус и пишет {"error": msg}.
func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, db.ErrStatusInUse), errors.Is(err, db.ErrTagTypeInUse):
		status = http.StatusConflict
	case errors.Is(err, errBadRequest):
		status = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
}

// errBadRequest — маркер ошибки ввода (плохой JSON/ID/параметр).
var errBadRequest = errors.New("некорректный запрос")

// okTrue — стандартный успешный ответ для мутаций без тела.
func okTrue() map[string]bool { return map[string]bool{"ok": true} }

// ownerFromStr преобразует "task"/"subtask" в db.StatusOwner.
func ownerFromStr(s string) (db.StatusOwner, error) {
	switch s {
	case "task":
		return db.OwnerTask, nil
	case "subtask":
		return db.OwnerSubtask, nil
	default:
		return 0, errBadRequest
	}
}

// idParam читает целочисленный параметр пути {name}.
func idParam(r *http.Request, name string) (int64, error) {
	s := r.PathValue(name)
	if s == "" {
		return 0, errBadRequest
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, errBadRequest
	}
	return n, nil
}

// intQuery читает целочисленный query-параметр (0, если пусто/некорректно).
func intQuery(r *http.Request, name string) int64 {
	n, _ := strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
	return n
}
