package ui

import "testing"

func TestEscGuardRequiresTwoPressesWhenDirty(t *testing.T) {
	var g escGuard
	if g.Press(true) {
		t.Fatal("первый Esc при изменениях не должен отменять")
	}
	if !g.Armed() {
		t.Fatal("первый Esc должен взвести предупреждение")
	}
	if !g.Press(true) {
		t.Fatal("второй Esc должен отменить")
	}
	if g.Armed() {
		t.Fatal("после отмены предупреждение снимается")
	}
}

func TestEscGuardClosesCleanInputImmediately(t *testing.T) {
	var g escGuard
	g.Press(true) // взведён
	if !g.Press(false) {
		t.Fatal("Esc без изменений должен закрывать сразу")
	}
	if g.Armed() {
		t.Fatal("предупреждение должно сниматься")
	}
}

func TestEscGuardResetDropsWarning(t *testing.T) {
	var g escGuard
	g.Press(true)
	g.Reset()
	if g.Armed() {
		t.Fatal("Reset должен снять предупреждение")
	}
	if g.Press(true) {
		t.Fatal("после Reset снова нужен первый Esc-предупреждение")
	}
}
