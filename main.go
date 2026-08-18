package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/charmbracelet/bubbletea"

	"github.com/detrenasama/tasky/internal/client"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/server"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"github.com/detrenasama/tasky/internal/xdg"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Println(version)
			return
		case "upgrade":
			os.Exit(runUpgrade())
		case "serve":
			os.Exit(runServe(os.Args[2:]))
		case "attach":
			os.Exit(runAttach(os.Args[2:]))
		case "help", "--help", "-h":
			os.Exit(runHelp())
		default:
			fmt.Fprintf(os.Stderr, "неизвестная команда: %s\n\n", os.Args[1])
			fmt.Fprint(os.Stderr, buildHelpText())
			os.Exit(1)
		}
	}
	os.Exit(runDefault(os.Args[1:]))
}

// dataDirPrepare создаёт каталог данных и переносит старую базу из рабочего
// каталога (только серверные режимы: default и serve).
func dataDirPrepare() (string, error) {
	dir := xdg.DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("не удалось создать каталог данных %s: %w", dir, err)
	}
	if err := migrateDataDir(dir); err != nil {
		log.Printf("миграция данных: %v", err)
	}
	return dir, nil
}

// socketPath выбирает адрес сокета: --socket PATH, TASKY_SOCKET или
// каталог данных по умолчанию.
func socketPath(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--socket" && i+1 < len(args) {
			return args[i+1]
		}
	}
	if s := os.Getenv("TASKY_SOCKET"); s != "" {
		return s
	}
	return filepath.Join(xdg.DataDir(), "tasky.sock")
}

// openServerDB открывает базу данных сервера.
func openServerDB(dir string) (*store.SQLite, func()) {
	conn, err := db.Open(filepath.Join(dir, "tasky.db"))
	if err != nil {
		log.Fatalf("не удалось открыть базу данных: %v", err)
	}
	return store.NewSQLite(conn), func() { conn.Close() }
}

// runTUI запускает интерфейс поверх хранилища и возвращает код выхода.
func runTUI(st store.Store, dir, version string) int {
	name, _, _ := st.GetSetting("theme")
	theme.Init(xdg.ConfigDir(), name)

	p := tea.NewProgram(ui.New(st, dir, version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка запуска: ", err)
		return 1
	}
	return 0
}

// runDefault — основной режим: подключается к уже работающему серверу, если
// он есть, иначе поднимает встроенный сервер до выхода из интерфейса.
func runDefault(args []string) int {
	dir, err := dataDirPrepare()
	if err != nil {
		log.Fatal(err)
	}
	sp := socketPath(args)

	cl, err := client.Dial(sp)
	if err == nil {
		code := runTUI(cl, dir, version)
		cl.Close()
		return code
	}

	st, closeDB := openServerDB(dir)
	defer closeDB()
	lis, err := server.Listen(sp)
	if err != nil {
		log.Fatalf("не удалось открыть сокет %s: %v", sp, err)
	}
	gs := server.Serve(lis, st)
	defer func() {
		gs.GracefulStop()
		os.Remove(sp)
	}()

	cl, err = client.Dial(sp)
	if err != nil {
		log.Fatalf("не удалось подключиться к встроенному серверу: %v", err)
	}
	code := runTUI(cl, dir, version)
	cl.Close()
	return code
}

// runServe — сервер в foreground: слушает сокет до SIGINT/SIGTERM.
func runServe(args []string) int {
	dir, err := dataDirPrepare()
	if err != nil {
		log.Fatal(err)
	}
	sp := socketPath(args)

	st, closeDB := openServerDB(dir)
	defer closeDB()
	lis, err := server.Listen(sp)
	if err != nil {
		if errors.Is(err, server.ErrAlreadyRunning) {
			log.Fatalf("%v (продолжите работу: tasky attach)", err)
		}
		log.Fatalf("не удалось открыть сокет %s: %v", sp, err)
	}
	defer os.Remove(sp)
	gs := server.Serve(lis, st)

	fmt.Printf("Сервер Tasky слушает: %s\nНажмите Ctrl+C, чтобы остановить.\n", sp)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	gs.GracefulStop()
	return 0
}

// runAttach — подключение к уже запущенному серверу (tasky serve).
func runAttach(args []string) int {
	dir := xdg.DataDir()
	sp := socketPath(args)
	cl, err := client.Dial(sp)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Не удалось подключиться к серверу (%s).\nЗапустите «tasky serve» (или просто «tasky» — он поднимет сервер сам).\n",
			sp)
		return 1
	}
	defer cl.Close()
	return runTUI(cl, dir, version)
}

// migrateDataDir переносит старую базу из рабочего каталога в dataDir при
// первом запуске новой версии: копирует tasky.db и tasky.db-wal/-shm.
func migrateDataDir(dir string) error {
	dst := filepath.Join(dir, "tasky.db")
	if _, err := os.Stat(dst); err == nil {
		return nil // новая база уже на месте
	}
	for _, name := range []string{"tasky.db", "tasky.db-wal", "tasky.db-shm"} {
		src := filepath.Join(".", name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := copyFile(src, filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	log.Printf("перенесена база из рабочего каталога в %s", dir)
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
