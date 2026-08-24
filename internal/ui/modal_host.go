package ui

import (
	"github.com/charmbracelet/bubbletea"
)

// ModalHost — держатель активной модалки (Form / Confirm и др.),
// перехватывающий клавиши в model.Update. Активная модалка возвращает
// (nil, _) из Update при закрытии — тогда хост сбрасывает active.
//
// Это ключевой камень компонентного подхода: добавление новой модалки
// больше не требует правок в цепочке роутинга model.Update и в switch'ах
// dialog()/updateX каждого экрана — достаточно вызвать m.modal.Show(...).
type ModalHost struct {
	active tea.Model
}

// Show открывает модалку (любой tea.Model, у которого View(w,h) и
// Update возвращают (nil, _) при закрытии).
func (h *ModalHost) Show(c tea.Model) {
	h.active = c
}

// Active — открыта ли сейчас какая-либо модалка.
func (h *ModalHost) Active() bool {
	return h.active != nil
}

// Update прокидывает сообщение активной модалке; при её закрытии сбрасывает
// active и возвращает cmd модалки.
func (h *ModalHost) Update(msg tea.Msg) tea.Cmd {
	if h.active == nil {
		return nil
	}
	newM, cmd := h.active.Update(msg)
	if newM == nil {
		h.active = nil
	} else {
		h.active = newM
	}
	return cmd
}

// View возвращает отрендеренную модалку (пустая строка, если неактивна).
func (h *ModalHost) View() string {
	if h.active == nil {
		return ""
	}
	return h.active.View()
}
