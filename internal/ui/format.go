package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

// fmtDur форматирует длительность в «Чч Мм» / «Мм Сс» / «Сс».
func fmtDur(d time.Duration) string {
	d = d.Round(time.Second)
	if d < 0 {
		d = 0
	}
	h := int(d / time.Hour)
	m := int(d%time.Hour) / int(time.Minute)
	sec := int(d%time.Minute) / int(time.Second)
	if h > 0 {
		return fmt.Sprintf("%dч %dм", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dм %dс", m, sec)
	}
	return fmt.Sprintf("%dс", sec)
}

// fmtElapsed форматирует идущую длительность в «Мм Сс» / «Сс».
func fmtElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	if d < 0 {
		d = 0
	}
	m := int(d / time.Minute)
	sec := int(d%time.Minute) / int(time.Second)
	if m > 0 {
		return fmt.Sprintf("%dм %dс", m, sec)
	}
	return fmt.Sprintf("%dс", sec)
}

// openURL открывает ссылку в браузере по умолчанию.
func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// wrapText переносит текст по словам на видимую ширину w, разрывая
// длинные слова.
func wrapText(s string, w int) string {
	if w <= 0 {
		return ""
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		var cur []rune
		curW := 0
		for _, word := range words {
			wordRunes := []rune(word)
			wordW := runewidth.StringWidth(word)
			if wordW > w {
				if len(cur) > 0 {
					out = append(out, string(cur))
					cur, curW = nil, 0
				}
				for len(wordRunes) > 0 {
					n, aw := 0, 0
					for _, r := range wordRunes {
						rw := runewidth.RuneWidth(r)
						if aw+rw > w {
							break
						}
						aw += rw
						n++
					}
					out = append(out, string(wordRunes[:n]))
					wordRunes = wordRunes[n:]
				}
				continue
			}
			if curW > 0 && curW+1+wordW > w {
				out = append(out, string(cur))
				cur, curW = nil, 0
			}
			if curW > 0 {
				cur = append(cur, ' ')
				curW++
			}
			cur = append(cur, wordRunes...)
			curW += wordW
		}
		if len(cur) > 0 {
			out = append(out, string(cur))
		}
	}
	return strings.Join(out, "\n")
}

// styleHints форматирует строку подсказок: сочетание клавиш — белым
// (основной цвет терминала), название — серым (faint, как раньше),
// разделитель « · » — серым. Каждый фрагмент имеет вид «КЛАВИША название»;
// граница — первый пробел.
func styleHints(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, " · ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		i := strings.Index(p, " ")
		if i < 0 {
			out = append(out, theme.Faint(p))
			continue
		}
		out = append(out, theme.Text(p[:i])+theme.Faint(p[i:]))
	}
	return strings.Join(out, theme.Faint(" · "))
}
