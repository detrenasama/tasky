package main

import (
	"fmt"
	"os"

	"github.com/detrenasama/tasky/internal/update"
)

// runUpgrade — команда tasky upgrade: самообновление до последнего релиза.
func runUpgrade() int {
	if version == "dev" || version == "" {
		fmt.Println("Обновление недоступно: сборка из исходников (нет версии релиза).")
		return 1
	}
	next, replaced, err := update.Upgrade(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка обновления: %v\n", err)
		return 1
	}
	if !replaced {
		fmt.Printf("Установлена последняя версия %s.\n", update.TrimV(next))
		return 0
	}
	fmt.Printf("Обновлено до %s. Перезапустите tasky.\n", update.TrimV(next))
	return 0
}
