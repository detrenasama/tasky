package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
)

// TestPaletteScroll — плавная прокрутка палитры с отступом listScrollOff:
// курсор не прилипает к краю видимого окна (кроме начала/конца списка).
func TestPaletteScroll(t *testing.T) {
	m := model{screen: screenTasks, height: 30}
	m.paletteInput = textinput.New()
	m.openPalette()
	if m.paletteSel < 0 {
		t.Fatal("палитра пуста после открытия")
	}
	visible := m.paletteVisible()
	off := listScrollOff
	if visible <= 2*off {
		off = 0
	}
	total := len(m.paletteRows)
	maxScroll := total - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	for i := 0; i < total-1; i++ {
		mm, _ := m.updatePalette(tea.KeyMsg{Type: tea.KeyDown})
		m = mm.(model)
		sel := m.paletteSel
		scroll := m.paletteScroll
		if scroll > 0 && sel-scroll < off-1 {
			t.Fatalf("шаг %d: курсор слишком близко к верху sel=%d scroll=%d off=%d", i, sel, scroll, off)
		}
		if scroll < maxScroll && (scroll+visible-1)-sel < off-1 {
			t.Fatalf("шаг %d: курсор слишком близко к низу sel=%d scroll=%d off=%d", i, sel, scroll, off)
		}
		// курсор всегда в пределах окна
		if sel < scroll || sel > scroll+visible-1 {
			t.Fatalf("шаг %d: курсор вне окна sel=%d scroll=%d visible=%d", i, sel, scroll, visible)
		}
	}
}
