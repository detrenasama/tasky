package ui

import (
	"strings"
	"testing"
)

// TestStyleHints — единый формат подсказок: клавиша белым, название серым,
// между клавишей и названием пробел, между пунктами — bullet «•». Тире «—»
// между клавишей и названием убирается.
func TestStyleHints(t *testing.T) {
	got := styleHints("Enter — открыть · e — изменить · d — удалить")
	plain := stripANSI(got)
	// тире не должно остаться
	if strings.Contains(plain, "—") {
		t.Errorf("в подсказках осталось тире: %q", plain)
	}
	// пункты разделены маленьким bullet'ом
	if !strings.Contains(plain, "·") {
		t.Errorf("между пунктами нет bullet'а: %q", plain)
	}
	// клавиша и название разделены пробелом (не тире, не bullet)
	if !strings.Contains(plain, "Enter открыть") {
		t.Errorf("ожидался формат «Enter открыть»: %q", plain)
	}
	if !strings.Contains(plain, "e изменить") {
		t.Errorf("ожидался формат «e изменить»: %q", plain)
	}
	// исходная строка без тире тоже корректна
	got2 := styleHints("ctrl+p команды")
	if strings.Contains(stripANSI(got2), "—") {
		t.Errorf("в одиночной подсказке осталось тире: %q", stripANSI(got2))
	}
}
