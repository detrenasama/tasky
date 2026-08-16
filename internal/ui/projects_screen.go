package ui

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/ui/theme"
)

type projMode int

const (
	projBrowse projMode = iota
	projInput
	projConfirm
	projDescEdit
	projLinkInput
	projLinks
	projLinkConfirm
	projSearch
)

type projFocus int

const (
	projFocusList projFocus = iota
	projFocusDesc
)

type projectItem struct{ p db.Project }

func (i projectItem) FilterValue() string { return i.p.Name }

func (i projectItem) Title() string { return i.p.Name }

func (i projectItem) Description() string {
	return "создан " + i.p.CreatedAt.Format("02.01.2006")
}

type projectsScreen struct {
	db            *sql.DB
	projects      []db.Project
	list          list.Model
	mode          projMode
	focus         projFocus
	input         textinput.Model
	linkName      textinput.Model
	linkInput     textinput.Model
	descText      textarea.Model
	linkList      list.Model
	confirmID     int64
	confirmLinkID int64
	lastErr       error

	desc  string
	links []db.Link
	descV viewport.Model

	searchInput textinput.Model
	searchQuery string
	linkTexts   map[int64]string
	items       []list.Item

	midH  int
	listW int
	descW int
	infoW int

	listDelegate *list.DefaultDelegate
	linkDelegate *list.DefaultDelegate
}

func newProjectsScreen(conn *sql.DB) *projectsScreen {
	s := &projectsScreen{db: conn, mode: projBrowse, focus: projFocusList}
	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	theme.ApplyToDelegate(&d)
	s.listDelegate = &d
	s.list = list.New(nil, &d, 40, 20)
	s.list.Title = "Проекты"
	s.list.SetShowHelp(false)
	s.list.SetShowPagination(false)
	s.list.SetShowStatusBar(false)
	// встроенный фильтр списка не используется — свой projSearch
	s.list.SetFilteringEnabled(false)

	s.input = textinput.New()
	s.input.Placeholder = "Имя проекта"
	s.input.Prompt = "> "
	s.input.CharLimit = 64
	s.input.Width = 40

	s.searchInput = textinput.New()
	s.searchInput.Placeholder = "название, описание, ссылки…"
	s.searchInput.Prompt = "> "
	s.searchInput.CharLimit = 64
	s.searchInput.Width = 40

	s.linkName = textinput.New()
	s.linkName.Placeholder = "Название (необязательно)"
	s.linkName.Prompt = "> "
	s.linkName.CharLimit = 64
	s.linkName.Width = 60

	s.linkInput = textinput.New()
	s.linkInput.Placeholder = "https://…"
	s.linkInput.Prompt = "> "
	s.linkInput.CharLimit = 512
	s.linkInput.Width = 60

	s.descText = textarea.New()
	s.descText.Placeholder = "Описание проекта"
	s.descText.ShowLineNumbers = false
	s.descText.SetWidth(60)
	s.descText.SetHeight(10)

	ld := list.NewDefaultDelegate()
	ld.ShowDescription = true
	theme.ApplyToDelegate(&ld)
	s.linkDelegate = &ld
	s.linkList = list.New(nil, &ld, 50, 8)
	s.linkList.Title = "Ссылки"
	s.linkList.SetShowHelp(false)
	s.linkList.SetShowPagination(false)
	s.linkList.SetShowStatusBar(false)
	s.linkList.DisableQuitKeybindings()

	s.list.DisableQuitKeybindings()

	s.descV = viewport.New(1, 1)
	return s
}

func (s *projectsScreen) load() {
	s.projects, _ = db.Projects(s.db)
	s.linkTexts, _ = db.ProjectLinksTexts(s.db)
	s.buildItems()
	s.lastErr = nil
	s.loadDesc()
}

// buildItems собирает список: при активном searchQuery — только совпавшие
// по названию, описанию или ссылкам проекты.
func (s *projectsScreen) buildItems() {
	selID := s.selectedProjectID()
	s.items = nil
	if q := strings.ToLower(strings.TrimSpace(s.searchQuery)); q != "" {
		for _, p := range s.projects {
			if strings.Contains(strings.ToLower(p.Name), q) ||
				strings.Contains(strings.ToLower(p.Desc), q) ||
				strings.Contains(strings.ToLower(s.linkTexts[p.ID]), q) {
				s.items = append(s.items, projectItem{p})
			}
		}
	} else {
		for _, p := range s.projects {
			s.items = append(s.items, projectItem{p})
		}
	}
	s.list.SetItems(s.items)
	if len(s.items) > 0 {
		idx := -1
		for i, item := range s.items {
			if it, ok := item.(projectItem); ok && it.p.ID == selID {
				idx = i
				break
			}
		}
		if idx < 0 {
			idx = 0
		}
		s.list.Select(idx)
	}
}

// loadDesc подгружает описание и ссылки выбранного проекта и пересобирает
// контент колонки описания.
func (s *projectsScreen) loadDesc() {
	pid := s.selectedProjectID()
	s.desc, _ = db.ProjectDescription(s.db, pid)
	s.links, _ = db.ProjectLinks(s.db, pid)
	items := make([]list.Item, len(s.links))
	for i, l := range s.links {
		items[i] = linkItem{l}
	}
	s.linkList.SetItems(items)
	s.refreshDesc()
}

// clearSearch сбрасывает активный поиск (используется из main.go при esc).
func (s *projectsScreen) clearSearch() {
	s.searchQuery = ""
	s.buildItems()
	s.loadDesc()
}

func (s *projectsScreen) selectedProjectID() int64 {
	if item, ok := s.list.SelectedItem().(projectItem); ok {
		return item.p.ID
	}
	return 0
}

// refreshDesc собирает контент viewport колонки описания.
func (s *projectsScreen) refreshDesc() {
	w := max(s.descV.Width, 1)
	var body []string
	if s.selectedProjectID() == 0 {
		body = append(body, theme.Faint("Выберите проект."))
	} else {
		desc := strings.TrimSpace(s.desc)
		if desc == "" {
			body = append(body, theme.Faint("Описание пустое. Нажмите e (в колонке описания), чтобы добавить."))
		} else {
			body = append(body, wrapText(desc, w))
		}
		if len(s.links) > 0 {
			body = append(body, "", theme.Faint("Ссылки:"))
			for _, l := range s.links {
				label := l.Name
				if label == "" {
					label = l.URL
				}
				body = append(body, truncateWEnd(linkStyle.Render("• "+label), w))
			}
		}
	}
	s.descV.SetContent(strings.Join(body, "\n"))
}

func (s *projectsScreen) resize(w, h int) {
	s.midH = max(h, 3)
	listW, descW, infoW := 0, 0, 0
	if w >= 110 {
		// три колонки 1:3:1 на всю ширину
		u := (w - 4) / 5
		listW, descW, infoW = u, 3*u, u
		listW += (w - 4) - 5*u
	} else if w >= 60 {
		// list + описание (1:1)
		listW = (w - 2) / 2
		descW = w - 2 - listW
	} else {
		listW = w
	}
	s.listW, s.descW, s.infoW = listW, descW, infoW
	frame := theme.Pane(false).GetHorizontalFrameSize()
	s.list.SetWidth(listW - frame)
	s.list.SetHeight(s.midH)
	s.descV.Width = max(descW-frame, 1)
	s.descV.Height = max(s.midH, 1)
	s.descText.SetWidth(max(descW-frame, 1))
	s.descText.SetHeight(max(s.midH, 1))
	s.refreshDesc()
}

// retheme пересобирает стили делегатов списков после смены темы.
func (s *projectsScreen) retheme() {
	theme.ApplyToDelegate(s.listDelegate)
	s.list.SetDelegate(s.listDelegate)
	theme.ApplyToDelegate(s.linkDelegate)
	s.linkList.SetDelegate(s.linkDelegate)
}

func (s *projectsScreen) header(w int) string {
	h := theme.HeaderStyle.Render("Tasky") + "  " + theme.Faint("Проекты")
	if s.searchQuery != "" {
		h += "  " + theme.Faint("поиск: ") + s.searchQuery
	}
	return padW(h, w)
}

func (s *projectsScreen) footer(w int) string {
	if s.mode == projDescEdit {
		return padW(theme.Faint("Ctrl+S — сохранить · Esc — отмена"), w)
	}
	if s.focus == projFocusDesc {
		return padW(theme.Faint("↑/↓ скролл · e — редактировать · l — ссылка · o — ссылки · / — поиск · Tab — список"), w)
	}
	if s.searchQuery != "" {
		return padW(theme.Faint("Поиск: «"+s.searchQuery+"» — / — изменить · Esc — сбросить"), w)
	}
	return padW(theme.Faint("↑/↓ выбор · n — создать · d — удалить · / — поиск · Tab — описание · esc — назад · q — выход"), w)
}

func (s *projectsScreen) view(w, h int) string {
	leftStyle := theme.Pane(s.focus == projFocusList)
	var left string
	if len(s.projects) == 0 && s.mode == projBrowse {
		left = fixedBox(theme.Pane(false), "Проектов нет.\nНажмите n для создания.", s.listW, s.midH)
	} else if s.searchQuery != "" && len(s.items) == 0 {
		left = fixedBox(theme.Pane(false), "Ничего не найдено по запросу\n«"+s.searchQuery+"».", s.listW, s.midH)
	} else {
		// bubbles/list не дополняет строки до ширины — паддинг вручную
		left = renderPane(leftStyle, padLines(s.list.View(), max(s.listW-leftStyle.GetHorizontalFrameSize(), 0), max(s.midH-leftStyle.GetVerticalFrameSize(), 0)))
	}

	cols := []string{left}
	if s.descW > 0 {
		cols = append(cols, "  ", s.descBox())
	}
	if s.infoW > 0 {
		cols = append(cols, "  ", s.infoBox())
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	return padH(body, w, h)
}

// descBox — средняя колонка: описание проекта (переносимое, прокручиваемое)
// и блок «Ссылки»; при редактировании вместо контента — textarea.
func (s *projectsScreen) descBox() string {
	style := theme.Pane(s.focus == projFocusDesc)
	if s.mode == projDescEdit {
		return renderPane(style, padLines(s.descText.View(), max(s.descW-style.GetHorizontalFrameSize(), 0), max(s.midH-style.GetVerticalFrameSize(), 0)))
	}
	return renderPane(style, s.descV.View())
}

// infoBox — правая колонка: сводка и настройки проекта (коэффициент времени
// и т.п.) — dim-заглушка.
func (s *projectsScreen) infoBox() string {
	return fixedBox(theme.Pane(false), "Сводка и настройки\n\nКоэффициент времени:\n1.0", s.infoW, s.midH)
}

func (s *projectsScreen) dialog() (string, bool) {
	switch s.mode {
	case projInput:
		body := s.input.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Новый проект", body: body,
			primary: "Enter — создать", esc: "Esc — отмена"}
		return d.render(), true
	case projConfirm:
		name := ""
		for _, p := range s.projects {
			if p.ID == s.confirmID {
				name = p.Name
			}
		}
		d := dialog{title: "Удаление проекта",
			body:    fmt.Sprintf("Удалить проект «%s» и всё его время?", name),
			primary: "y — удалить", esc: "n — нет"}
		return d.render(), true
	case projDescEdit:
		return "", false
	case projLinkInput:
		body := s.linkName.View() + "\n" + s.linkInput.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Добавить ссылку", body: body,
			primary: "Enter — добавить · Tab — поле", esc: "Esc — отмена"}
		return d.render(), true
	case projLinks:
		body := s.linkList.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Не удалось открыть: "+s.lastErr.Error())
		}
		d := dialog{title: "Ссылки проекта",
			body:    body,
			primary: "Enter — открыть · d — удалить", esc: "Esc — закрыть"}
		return d.render(), true
	case projLinkConfirm:
		label := ""
		for _, l := range s.links {
			if l.ID == s.confirmLinkID {
				label = l.Name
				if label == "" {
					label = l.URL
				}
			}
		}
		d := dialog{title: "Удаление ссылки",
			body:    fmt.Sprintf("Удалить ссылку «%s»?", label),
			primary: "y — удалить", esc: "n — нет"}
		return d.render(), true
	case projSearch:
		body := s.searchInput.View()
		if s.searchQuery != "" {
			body += "\n\n" + theme.Faint(fmt.Sprintf("Найдено: %d элементов", len(s.items)))
		}
		d := dialog{title: "Поиск", body: body,
			primary: "Enter — применить", esc: "Esc — отмена"}
		return d.render(), true
	}
	return "", false
}

func (m *model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.proj
	switch s.mode {
	case projInput:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(s.input.Value())
			if name != "" {
				_, err := db.CreateProject(s.db, name)
				s.lastErr = err
				if err == nil {
					s.load()
				}
			}
			s.input.SetValue("")
			s.input.Blur()
			s.mode = projBrowse
		case "esc":
			s.input.SetValue("")
			s.input.Blur()
			s.mode = projBrowse
		}
		return m, cmd
	case projConfirm:
		switch msg.String() {
		case "y", "enter":
			db.DeleteProject(s.db, s.confirmID)
			s.mode = projBrowse
			s.load()
		case "n", "esc":
			s.mode = projBrowse
		}
		return m, nil
	case projDescEdit:
		var cmd tea.Cmd
		s.descText, cmd = s.descText.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			db.UpdateProjectDescription(s.db, s.selectedProjectID(), s.descText.Value())
			s.descText.Blur()
			s.mode = projBrowse
			s.loadDesc()
		case "esc":
			s.descText.Blur()
			s.mode = projBrowse
		}
		return m, cmd
	case projLinkInput:
		var cmd tea.Cmd
		if s.linkName.Focused() {
			s.linkName, cmd = s.linkName.Update(msg)
		} else {
			s.linkInput, cmd = s.linkInput.Update(msg)
		}
		switch msg.String() {
		case "tab":
			if s.linkName.Focused() {
				s.linkName.Blur()
				s.linkInput.Focus()
			} else {
				s.linkInput.Blur()
				s.linkName.Focus()
			}
		case "enter":
			if s.linkInput.Focused() {
				url := strings.TrimSpace(s.linkInput.Value())
				if url != "" {
					_, err := db.CreateProjectLink(s.db, s.selectedProjectID(),
						strings.TrimSpace(s.linkName.Value()), url)
					s.lastErr = err
					if err == nil {
						s.loadDesc()
					}
				}
				s.linkName.SetValue("")
				s.linkName.Blur()
				s.linkInput.SetValue("")
				s.linkInput.Blur()
				s.mode = projBrowse
			} else {
				// Enter на названии — перейти к адресу
				s.linkName.Blur()
				s.linkInput.Focus()
			}
		case "esc":
			s.linkName.SetValue("")
			s.linkName.Blur()
			s.linkInput.SetValue("")
			s.linkInput.Blur()
			s.mode = projBrowse
		}
		return m, cmd
	case projLinks:
		var cmd tea.Cmd
		s.linkList, cmd = s.linkList.Update(msg)
		switch msg.String() {
		case "enter":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				if err := openURL(item.l.URL); err != nil {
					s.lastErr = err
				}
			}
			return m, nil
		case "d":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.confirmLinkID = item.l.ID
				s.mode = projLinkConfirm
			}
			return m, nil
		case "esc":
			s.mode = projBrowse
			s.lastErr = nil
		}
		return m, cmd
	case projLinkConfirm:
		switch msg.String() {
		case "y", "enter":
			db.DeleteProjectLink(s.db, s.confirmLinkID)
			s.mode = projLinks
			s.loadDesc()
		case "n", "esc":
			s.mode = projLinks
		}
		return m, nil
	case projSearch:
		var cmd tea.Cmd
		s.searchInput, cmd = s.searchInput.Update(msg)
		switch msg.String() {
		case "enter":
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.searchInput.Blur()
			s.mode = projBrowse
			s.buildItems()
			s.loadDesc()
		case "esc":
			s.searchQuery = ""
			s.searchInput.Blur()
			s.mode = projBrowse
			s.buildItems()
			s.loadDesc()
		default:
			// живой фильтр по мере ввода
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.buildItems()
			s.loadDesc()
		}
		return m, cmd
	}

	switch msg.String() {
	case "tab":
		if s.descW > 0 {
			if s.focus == projFocusList {
				s.focus = projFocusDesc
			} else {
				s.focus = projFocusList
			}
		}
		return m, nil
	case "e":
		if s.focus == projFocusDesc {
			s.descText.SetValue(s.desc)
			s.mode = projDescEdit
			s.descText.Focus()
			return m, nil
		}
	case "l":
		if s.focus == projFocusDesc {
			s.lastErr = nil
			s.linkName.SetValue("")
			s.linkInput.SetValue("")
			s.mode = projLinkInput
			s.linkName.Focus()
			s.linkInput.Blur()
			return m, nil
		}
	case "o":
		if s.focus == projFocusDesc {
			s.lastErr = nil
			s.mode = projLinks
			return m, nil
		}
	case "/":
		s.lastErr = nil
		s.searchInput.SetValue(s.searchQuery)
		s.searchInput.Focus()
		s.mode = projSearch
		return m, nil
	case "n":
		if s.focus == projFocusList {
			s.mode = projInput
			s.input.Focus()
			return m, nil
		}
	case "d":
		if s.focus == projFocusList {
			item, ok := s.list.SelectedItem().(projectItem)
			if ok {
				s.confirmID = item.p.ID
				s.mode = projConfirm
			}
			return m, nil
		}
	}

	if s.focus == projFocusDesc {
		switch msg.String() {
		case "up", "down", "pgup", "pgdown", "home", "end":
			s.descV, _ = s.descV.Update(msg)
			return m, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	beforeID := s.selectedProjectID()
	s.list, cmd = s.list.Update(msg)
	if s.selectedProjectID() != beforeID {
		s.loadDesc()
		s.descV.GotoTop()
	}
	return m, cmd
}
