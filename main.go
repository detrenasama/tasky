package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
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
			os.Exit(runUpgrade(os.Args[2:]))
		case "serve":
			os.Exit(runServe(os.Args[2:]))
		case "attach":
			os.Exit(runAttach(os.Args[2:]))
		case "status":
			os.Exit(runStatus(os.Args[2:]))
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

// dataDirPrepare создаёт каталог данных (только серверные режимы: default и
// serve). TASKY_HOME просто переключает используемый каталог, никакого
// копирования данных не выполняется.
func dataDirPrepare() (string, error) {
	dir := xdg.DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("не удалось создать каталог данных %s: %w", dir, err)
	}
	return dir, nil
}

// httpAddr выбирает адрес HTTP-эндпоинтов интеграций: --http-addr ADDR,
// TASKY_HTTP_ADDR или 127.0.0.1:9110 по умолчанию.
func httpAddr(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--http-addr" && i+1 < len(args) {
			return args[i+1]
		}
	}
	if a := os.Getenv("TASKY_HTTP_ADDR"); a != "" {
		return a
	}
	return "127.0.0.1:9110"
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

// startHTTP поднимает HTTP-сервер интеграций на httpAddr(args). Если порт
// занят — печатает предупреждение и продолжает без HTTP (gRPC работает
// независимо); возвращает nil в этом случае.
func startHTTP(args []string, st store.Store) *http.Server {
	addr := httpAddr(args)
	lis, err := server.ListenHTTP(addr)
	if err != nil {
		log.Printf("HTTP-сервер (%s) не запущен: %v", addr, err)
		return nil
	}
	return server.ServeHTTP(lis, st)
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

	// HTTP-эндпоинты для внешних интеграций (GET /status и подобные).
	if hs := startHTTP(args, st); hs != nil {
		defer hs.Close()
	}

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

	hs := startHTTP(args, st)
	fmt.Printf("Сервер Tasky слушает: %s\n", sp)
	fmt.Printf("HTTP (интеграции): http://%s\n", httpAddr(args))
	fmt.Println("Нажмите Ctrl+C, чтобы остановить.")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	gs.GracefulStop()
	if hs != nil {
		hs.Close()
	}
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
