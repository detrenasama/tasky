package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/detrenasama/tasky/internal/update"
	"github.com/mattn/go-isatty"
)

// cliReporter выводит в консоль информативный лог этапов обновления и
// прогресс-бар загрузки (если доступен терминал).
type cliReporter struct {
	tty   bool
	onBar bool
}

func newCLIReporter() *cliReporter {
	return &cliReporter{tty: isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsTerminal(os.Stderr.Fd())}
}

// report обрабатывает уведомление от update.Upgrade.
func (c *cliReporter) report(step update.Step, msg string, frac float64) {
	if step == update.StepDownload {
		if msg == "" {
			// Чистый прогресс загрузки — рисуем прогресс-бар.
			if c.tty {
				c.drawBar(frac)
				c.onBar = true
			}
			return
		}
		// Текстовое сообщение этапа загрузки (URL / «Загружено.»).
		if c.onBar {
			fmt.Fprintln(os.Stderr)
			c.onBar = false
		}
		fmt.Println(msg)
		return
	}
	// Прочие этапы: сначала завершаем строку прогресс-бара, затем печатаем
	// сообщение.
	if c.onBar {
		fmt.Fprintln(os.Stderr)
		c.onBar = false
	}
	if msg != "" {
		fmt.Println(msg)
	}
}

// drawBar рисует прогресс-бар загрузки в одной строке.
func (c *cliReporter) drawBar(frac float64) {
	const width = 30
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * width)
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	fmt.Fprintf(os.Stderr, "\r[%s] %3.0f%%", bar, frac*100)
}

// isTerminal сообщает, является ли fd терминалом (для интерактивности).
func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd())
}

// confirm запрашивает у пользователя подтверждение (Y/n — по умолчанию да).
func confirm(prompt string) bool {
	fmt.Print(prompt)
	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "" || line == "y" || line == "yes"
}
