package ui

import (
	"os"
	"os/exec"

	"github.com/charmbracelet/bubbletea"
)

// editReturnMsg посылается после завершения внешнего редактора ($EDITOR).
type editReturnMsg struct {
	path string
	err  error
}

// editorCmd возвращает команду запуска $EDITOR (или $VISUAL, иначе vi) с
// указанным файлом.
func editorCmd(path string) (*exec.Cmd, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	return exec.Command(editor, path), nil
}

// openInEditor пишет initial во временный файл и возвращает tea.Cmd, который
// приостанавливает TUI, открывает редактор и по выходе шлёт editReturnMsg.
// Возвращает путь к временному файлу (его нужно сохранить на экране) и ошибку.
func openInEditor(initial string) (string, tea.Cmd, error) {
	f, err := os.CreateTemp("", "tasky-desc-*.md")
	if err != nil {
		return "", nil, err
	}
	if _, err := f.WriteString(initial); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, err
	}
	f.Close()
	path := f.Name()
	cmd, err := editorCmd(path)
	if err != nil {
		os.Remove(path)
		return "", nil, err
	}
	run := func(err error) tea.Msg { return editReturnMsg{path: path, err: err} }
	return path, tea.ExecProcess(cmd, run), nil
}
