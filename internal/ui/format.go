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

// ruWeekdayShort — русское сокращённое название дня недели (Пн/Вт/Ср/Чт/Пт/Сб/Вс).
func ruWeekdayShort(t time.Time) string {
	switch t.Weekday() {
	case time.Monday:
		return "Пн"
	case time.Tuesday:
		return "Вт"
	case time.Wednesday:
		return "Ср"
	case time.Thursday:
		return "Чт"
	case time.Friday:
		return "Пт"
	case time.Saturday:
		return "Сб"
	default:
		return "Вс"
	}
}

// fmtTimeEntry форматирует момент времени для записей учёта:
// «[2006-01-02 Чт 15:04]» (с русским днём недели).
func fmtTimeEntry(t time.Time) string {
	return "[" + t.Format("2006-01-02") + " " + ruWeekdayShort(t) + " " +
		t.Format("15:04") + "]"
}

// fmtDurationHM форматирует длительность как «ЧЧ:ММ» (часы:минуты, минуты с
// ведущим нулём): 9 минут → "0:09", 1ч5м → "1:05", 25ч30м → "25:30".
func fmtDurationHM(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	h := int(d / time.Hour)
	m := int(d % time.Hour / time.Minute)
	return fmt.Sprintf("%d:%02d", h, m)
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

// styleHints форматирует строку подсказок в единый вид: сочетание клавиш —
// белым (theme.Text), название — серым (theme.Muted), между клавишей и
// названием — пробел, между пунктами — маленький bullet «·» (серым).
// Тире «—» между клавишей и названием (если есть в исходной строке)
// убирается. Формат элемента: «КЛАВИША название».
func styleHints(s string) string {
	if s == "" {
		return ""
	}
	// убираем тире между клавишей и названием, оставляя один пробел
	norm := strings.ReplaceAll(s, " — ", " ")
	norm = strings.ReplaceAll(norm, "— ", " ")
	norm = strings.ReplaceAll(norm, " —", " ")
	parts := strings.Split(norm, " · ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		i := strings.Index(p, " ")
		if i < 0 {
			out = append(out, theme.Faint(p))
			continue
		}
		out = append(out, theme.Text(p[:i])+" "+theme.Muted(p[i+1:]))
	}
	return strings.Join(out, theme.Muted(" · "))
}
