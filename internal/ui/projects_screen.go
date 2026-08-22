package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
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
	store         store.Store
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

	listDelegate *list.DefaultDelegate
	linkDelegate *list.DefaultDelegate
}

func newProjectsScreen(st store.Store) *projectsScreen {
	s := &projectsScreen{store: st, mode: projBrowse, focus: projFocusList}
	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	theme.ApplyToDelegate(&d)
	s.listDelegate = &d
	s.list = list.New(nil, &d, 40, 20)
	s.list.SetShowTitle(false)
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
	s.projects, _ = s.store.Projects()
	s.linkTexts, _ = s.store.ProjectLinksTexts()
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
	s.desc, _ = s.store.ProjectDescription(pid)
	s.links, _ = s.store.ProjectLinks(pid)
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

// startNew открывает ввод имени нового проекта (из клавиши n или палитры).
func (s *projectsScreen) startNew() {
	s.mode = projInput
	s.input.Focus()
}

// startDelete открывает подтверждение удаления выбранного проекта.
func (s *projectsScreen) startDelete() {
	item, ok := s.list.SelectedItem().(projectItem)
	if !ok {
		return
	}
	s.confirmID = item.p.ID
	s.mode = projConfirm
}

// startEditDescription открывает инлайн-редактирование описания проекта
// (Enter в списке — то же, что раньше «e» в колонке описания).
func (s *projectsScreen) startEditDescription() {
	s.descText.SetValue(s.desc)
	s.mode = projDescEdit
	s.descText.Focus()
}

// startLinkInput открывает модалку добавления ссылки к проекту.
func (s *projectsScreen) startLinkInput() {
	s.lastErr = nil
	s.linkName.SetValue("")
	s.linkInput.SetValue("")
	s.mode = projLinkInput
	s.linkName.Focus()
	s.linkInput.Blur()
}

// openLinks открывает список ссылок проекта.
func (s *projectsScreen) openLinks() {
	s.lastErr = nil
	s.mode = projLinks
}

// startSearch открывает модалку поиска по проектам.
func (s *projectsScreen) startSearch() {
	s.lastErr = nil
	s.searchInput.SetValue(s.searchQuery)
	s.searchInput.Focus()
	s.mode = projSearch
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
			body = append(body, theme.Faint("Описание пустое. Нажмите Enter, чтобы добавить."))
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
	// вся ширина w (центральная область) — под список и описание; правая
	// колонка рендерится отдельно в app.rightRail.
	avail := w
	listW, descW := 0, 0
	if avail >= 100 {
		// список и описание равной ширины (1:1) с разделителем 2 пробела
		listW = (avail - 2) / 2
		descW = avail - 2 - listW
	} else {
		listW = avail
	}
	s.listW, s.descW = listW, descW
	frame := theme.Pane(false).GetHorizontalFrameSize()
	// 1 строка сверху — название страницы над списком
	s.list.SetWidth(listW - frame)
	// 2 строки сверху — название страницы + отступ
	s.list.SetHeight(max(s.midH-2, 1))
	s.descV.Width = max(descW-frame, 1)
	s.descV.Height = max(s.midH-1, 1)
	s.descText.SetWidth(max(descW-frame, 1))
	s.descText.SetHeight(max(s.midH-1, 1))
	s.refreshDesc()
}

// retheme пересобирает стили делегатов списков после смены темы.
func (s *projectsScreen) retheme() {
	theme.ApplyToDelegate(s.listDelegate)
	s.list.SetDelegate(s.listDelegate)
	theme.ApplyToDelegate(s.linkDelegate)
	s.linkList.SetDelegate(s.linkDelegate)
}

func (s *projectsScreen) footer(w int) string {
	if s.mode == projDescEdit {
		return "Ctrl+S — сохранить · Esc — отмена"
	}
	// подсказки действий перенесены в палитру команд (Ctrl+P); здесь не выводим
	return ""
}

func (s *projectsScreen) view(w, h int) string {
	leftStyle := theme.Pane(false)
	W := max(s.listW-leftStyle.GetHorizontalFrameSize(), 0)
	H := max(s.midH, 1)
	titleTxt := theme.HeaderStyle.Render("Проекты")
	var body string
	switch {
	case len(s.projects) == 0 && s.mode == projBrowse:
		body = "Проектов нет.\nНажмите n для создания."
	case s.searchQuery != "" && len(s.items) == 0:
		body = "Ничего не найдено по запросу\n«" + s.searchQuery + "»."
	default:
		// bubbles/list не дополняет строки до ширины — паддинг вручную
		body = s.list.View()
	}
	// заголовок страницы + нижний отступ 1 строка, затем тело — единая панель
	leftContent := titleTxt + "\n" + padW("", W) + "\n" + body
	left := renderPane(leftStyle, padLines(leftContent, W, H))

	cols := []string{left}
	if s.descW > 0 {
		cols = append(cols, "  ", s.descBox())
	}
	bodyOut := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	return padH(bodyOut, w, h)
}

// rightContent возвращает содержимое правой колонки (без панели); панель и
// «Tasky vX» снизу добавляет app.rightRail.
func (s *projectsScreen) rightContent(h int) string {
	return "Сводка и настройки\n\nКоэффициент времени:\n1.0"
}

// descBox — средняя колонка: описание проекта (переносимое, прокручиваемое)
// и блок «Ссылки»; при редактировании вместо контента — textarea.
func (s *projectsScreen) descBox() string {
	style := theme.Pane(false)
	if s.mode == projDescEdit {
		return renderPane(style, padLines(s.descText.View(), max(s.descW-style.GetHorizontalFrameSize(), 0), max(s.midH-style.GetVerticalFrameSize(), 0)))
	}
	return renderPane(style, s.descV.View())
}

// infoBox removed: правая колонка рендерится через rightRail/rightContent.

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
				_, err := s.store.CreateProject(name)
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
			s.store.DeleteProject(s.confirmID)
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
			s.store.UpdateProjectDescription(s.selectedProjectID(), s.descText.Value())
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
					_, err := s.store.CreateProjectLink(s.selectedProjectID(),
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
			s.store.DeleteProjectLink(s.confirmLinkID)
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
		// переключение фокуса между панелями упразднено
		return m, nil
	case "enter":
		s.startEditDescription()
		return m, nil
	case "pgup", "pgdown":
		s.descV, _ = s.descV.Update(msg)
		return m, nil
	case "left", "right":
		return m, nil
	case "e":
		// описание теперь открывается по Enter; «e» оставляем без действия
		return m, nil
	case "l":
		s.startLinkInput()
		return m, nil
	case "o":
		s.openLinks()
		return m, nil
	case "/":
		s.startSearch()
		return m, nil
	case "n":
		s.startNew()
		return m, nil
	case "d":
		s.startDelete()
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
