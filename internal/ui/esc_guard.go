package ui

// escHintInput — подсказка при первой попытке отменить ввод по Esc.
const escHintInput = "Нажмите Esc ещё раз, чтобы отменить ввод."

// escHintChanges — подсказка при первой попытке отменить изменения по Esc.
const escHintChanges = "Нажмите Esc ещё раз, чтобы отменить изменения."

// escGuard — двухшаговая отмена по Esc: первый Esc при несохранённых
// изменениях только предупреждает, второй — подтверждает отмену.
type escGuard struct {
	armed bool
}

// Reset снимает предупреждение (любая клавиша, кроме Esc, или смена режима).
func (g *escGuard) Reset() {
	g.armed = false
}

// Press обрабатывает нажатие Esc. dirty — есть ли несохранённые изменения.
// Возвращает true, если отмену нужно выполнить сейчас.
func (g *escGuard) Press(dirty bool) bool {
	if !dirty {
		g.armed = false
		return true
	}
	if g.armed {
		g.armed = false
		return true
	}
	g.armed = true
	return false
}

// Armed — показано ли предупреждение о повторном Esc.
func (g *escGuard) Armed() bool {
	return g.armed
}
