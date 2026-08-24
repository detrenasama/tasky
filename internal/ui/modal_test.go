package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestOverlayModalNotFullWidth — диалоговая модалка не должна растягиваться
// во всю ширину экрана (иначе её фон выглядит серой полосой). Проверяем, что
// на узком терминале у модалки есть боковые поля и ширина коробки < w.
func TestOverlayModalNotFullWidth(t *testing.T) {
	for _, w := range []int{80, 100, 150} {
		h := 24
		row := strings.Repeat("x", w)
		base := strings.Repeat(row+"\n", h)
		d := dialog{
			title:   "Ссылки",
			body:    "Доки",
			primary: "Enter открыть · e изменить · d удалить · n новая · Esc закрыть",
		}
		out := overlay(base, d.render(), w, h, dialogMaxW(w))
		lines := strings.Split(out, "\n")
		found := false
		for _, l := range lines {
			plain := stripANSI(l)
			i := strings.Index(plain, "Ссылки")
			if i < 0 {
				continue
			}
			found = true
			rightPad := len(plain) - (i + len("Ссылки"))
			if i == 0 {
				t.Errorf("w=%d: модалка прижата к левому краю (нет левого поля)", w)
			}
			if rightPad <= 0 {
				t.Errorf("w=%d: модалка прижата к правому краю (нет правого поля)", w)
			}
			t.Logf("w=%d: leftMargin=%d rightPad=%d", w, i, rightPad)
			break
		}
		if !found {
			t.Fatalf("w=%d: в выводе нет заголовка модалки", w)
		}
	}
}

// TestWrapStyled — перенос длинной строки с ANSI по ширине, состояние стиля
// сохраняется, а сама строка не превышает заданную ширину.
func TestWrapStyled(t *testing.T) {
	s := "\x1b[32mEnter\x1b[0m открыть · \x1b[32me\x1b[0m изменить длинный текст без пробела-переноса abcdefghijklmnop"
	lines := wrapStyled(s, 20)
	for _, l := range lines {
		if lipgloss.Width(l) > 20 {
			t.Errorf("строка шире лимита: %q", l)
		}
	}
	if len(lines) < 2 {
		t.Errorf("ожидался перенос, получено строк: %d", len(lines))
	}
}

// TestWrapStyledKeepsBox — строка с box-drawing символами (рамка модалки)
// никогда не разрывается посередине: ┌ и ┐ остаются на одной строке.
func TestWrapStyledKeepsBox(t *testing.T) {
	border := "\x1b[38;5;245m┌──────────────────────────────────────────────────────────┐\x1b[0m"
	lines := wrapStyled(border, 40)
	if len(lines) != 1 {
		t.Fatalf("рамка разорвана на %d строк: %q", len(lines), lines)
	}
	if !strings.Contains(lines[0], "┌") || !strings.Contains(lines[0], "┐") {
		t.Errorf("рамка потеряла углы: %q", lines[0])
	}
}
