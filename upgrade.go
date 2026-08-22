package main

import (
	"fmt"
	"os"

	"github.com/detrenasama/tasky/internal/update"
)

// runUpgrade — команда tasky upgrade [-y]: самообновление до последнего
// релиза. Без флага -y запрашивает подтверждение перед загрузкой.
func runUpgrade(args []string) int {
	yes := false
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			yes = true
		}
	}

	if version == "dev" || version == "" {
		fmt.Println("Обновление недоступно: сборка из исходников (нет версии релиза).")
		return 1
	}

	latest, needed, err := update.CheckUpdate(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка проверки обновления: %v\n", err)
		return 1
	}
	if !needed {
		fmt.Printf("Установлена последняя версия %s.\n", update.TrimV(latest))
		return 0
	}

	fmt.Printf("Доступно обновление: %s → %s\n", update.TrimV(version), update.TrimV(latest))

	if !yes {
		if !isTerminal(os.Stdin) {
			fmt.Println("Неинтерактивный режим: добавьте -y для автоматического обновления.")
			return 0
		}
		if !confirm("Загрузить и установить обновление? [Y/n] ") {
			fmt.Println("Обновление отменено.")
			return 0
		}
	}

	rep := newCLIReporter()
	next, replaced, err := update.Upgrade(version, rep.report)
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
