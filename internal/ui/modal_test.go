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

// TestOverlayBackdropKeepsBackground — бэкдроп вокруг модалки должен
// сохранять цвета фона экрана (а не превращаться в плоский блок дефолтного
// фона). Проверяем, что слева/справа от диалога остаются SGR-коды фона, причём
// затемнённые (dimFactor), а сама модалка использует свой цвет (ModalStyle).
func TestOverlayBackdropKeepsBackground(t *testing.T) {
	const w, h = 80, 5
	// Строка экрана с фоном #1e1414 (30,20,20) и сбросом в конце.
	bg := "\x1b[48;2;30;20;20m"
	plain := strings.Repeat(" ", w)
	baseLine := bg + plain + "\x1b[0m"
	base := strings.Repeat(baseLine+"\n", h)

	d := dialog{title: "Заголовок", body: "Тело модалки", primary: "Enter — ок"}
	out := overlay(base, d.render(), w, h, dialogMaxW(w))
	lines := strings.Split(out, "\n")

	var dialogLine string
	for _, l := range lines {
		if strings.Contains(stripANSI(l), "Заголовок") {
			dialogLine = l
			break
		}
	}
	if dialogLine == "" {
		t.Fatal("в выводе нет строки с модалкой")
	}
	// Затемнённый фон бэкдропа: 30/20/20 * 0.4 = 12/8/8.
	if !strings.Contains(dialogLine, "48;2;12;8;8") {
		t.Errorf("бэкдроп потерял/не затемнил фон: %q", dialogLine)
	}
	// Фон самой модалки (ModalStyle = #161616) сохраняется: в truecolor
	// профиле это 48;2;22;22;22, в 256 — 48;5;232.
	if !strings.Contains(dialogLine, "48;2;22;22;22") && !strings.Contains(dialogLine, "48;5;232") {
		t.Errorf("модалка не использует свой цвет фона: %q", dialogLine)
	}
}

// TestSplitStyledHidesCrossingWord — слово, пересекающее левый/правый край
// модалки, должно полностью прятаться, а не разрываться пополам (часть слева,
// часть справа). Слова целиком в полях сохраняются.
func TestSplitStyledHidesCrossingWord(t *testing.T) {
	s := "aaa bbb ccc ddd"
	// col0=4: "aaa " занимает 0..3, "bbb" начинается в 4 (под левым краем).
	left, right := splitStyledAtWidth(s, 4, 2)
	lv, rv := stripANSI(left), stripANSI(right)
	if strings.Contains(lv, "bbb") || strings.Contains(rv, "bbb") {
		t.Errorf("слово на краю не спрятано: left=%q right=%q", lv, rv)
	}
	if !strings.Contains(lv, "aaa") {
		t.Errorf("слово слева потеряно: %q", lv)
	}
	if !strings.Contains(rv, "ccc") || !strings.Contains(rv, "ddd") {
		t.Errorf("слова справа потеряны: %q", rv)
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
