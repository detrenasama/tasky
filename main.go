package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbletea"

	"github.com/kalpamer/tasky/internal/db"
	"github.com/kalpamer/tasky/internal/ui"
)

const dbPath = "tasky.db"

func main() {
	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("не удалось открыть базу данных: %v", err)
	}
	defer conn.Close()

	p := tea.NewProgram(ui.New(conn), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка запуска: ", err)
		os.Exit(1)
	}
}
