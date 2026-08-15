package main

import "fmt"

// buildHelpText собирает текст справки по CLI-командам.
func buildHelpText() string {
	return fmt.Sprintf(`Tasky — TUI-менеджер задач с учётом времени

Использование:
  tasky               запустить интерфейс
  tasky help          показать эту справку (то же: --help, -h)
  tasky --version     показать версию (то же: -v)
  tasky upgrade       обновить до последней версии

Данные: ~/.local/share/tasky/ (переопределение — TASKY_HOME)
Версия: %s
`, version)
}

// runHelp печатает справку и возвращает код выхода.
func runHelp() int {
	fmt.Print(buildHelpText())
	return 0
}
