package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/detrenasama/tasky/internal/store"
)

// ErrAlreadyRunning — на сокете уже слушает живой сервер.
var ErrAlreadyRunning = errors.New("сервер уже запущен")

// Listen открывает unix-сокет. Если на пути остался файл сокета от упавшего
// процесса — он удаляется (dial не удаётся); если сокет живой — ErrAlreadyRunning.
func Listen(socketPath string) (net.Listener, error) {
	if fi, err := os.Stat(socketPath); err == nil && fi.Mode()&os.ModeSocket != 0 {
		conn, err := net.DialTimeout("unix", socketPath, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil, fmt.Errorf("%w: %s", ErrAlreadyRunning, socketPath)
		}
		if err := os.Remove(socketPath); err != nil {
			return nil, fmt.Errorf("удаление битого сокета: %w", err)
		}
	}
	return net.Listen("unix", socketPath)
}

// Serve создаёт gRPC-сервер с сервисом Tasky и запускает его на слушателе
// в фоновой горутине. Возвращает *grpc.Server для GracefulStop.
func Serve(lis net.Listener, st store.Store) *grpc.Server {
	gs := grpc.NewServer()
	Register(gs, st)
	go func() {
		if err := gs.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			// падение serve печатается, но процесс живёт (TUI продолжает работу)
			fmt.Fprintln(os.Stderr, "gRPC-сервер:", err)
		}
	}()
	return gs
}
