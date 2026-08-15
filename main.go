package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbletea"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/ui"
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
		case "help", "--help", "-h":
			os.Exit(runHelp())
		default:
			fmt.Fprintf(os.Stderr, "неизвестная команда: %s\n\n", os.Args[1])
			fmt.Fprint(os.Stderr, buildHelpText())
			os.Exit(1)
		}
	}

	dir := xdg.DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("не удалось создать каталог данных %s: %v", dir, err)
	}
	if err := migrateDataDir(dir); err != nil {
		log.Printf("миграция данных: %v", err)
	}

	conn, err := db.Open(filepath.Join(dir, "tasky.db"))
	if err != nil {
		log.Fatalf("не удалось открыть базу данных: %v", err)
	}
	defer conn.Close()

	p := tea.NewProgram(ui.New(conn, dir, version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка запуска: ", err)
		os.Exit(1)
	}
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
