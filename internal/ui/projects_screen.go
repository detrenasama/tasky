package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui/theme"
)

type projMode int

const (
	projBrowse projMode = iota
	projInput
	projConfirm
	projDescModal
	projLinkEdit
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
	editLinkID    int64
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

	// плавная прокрутка: listTop — верхняя видимая строка, listH — высота
	// окна; bubbles/list «разворачивается» на все элементы (см. scroll.go).
	listTop int
	listH   int

	notice      string
	extEditPath string
	extEditMode int
	descViewer  *descViewer

	// модалка описания
	dmState  dmState
	dmPrev   dmState
	descWork string
	fullW    int
	fullH    int

	listDelegate selDelegate
	linkDelegate *list.DefaultDelegate
}

func newProjectsScreen(st store.Store) *projectsScreen {
	s := &projectsScreen{store: st, mode: projBrowse, focus: projFocusList}
	d := newListDelegate()
	d.ShowDescription = true
	s.listDelegate = d
	s.list = list.New(nil, d, 40, 20)
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
	s.linkList.SetShowTitle(false)
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
func (s *projectsScreen) loadDesc() {
	pid := s.selectedProjectID()
	s.desc, _ = s.store.ProjectDescription(pid)
	s.links, _ = s.store.ProjectLinks(pid)
	items := make([]list.Item, len(s.links))
	for i, l := range s.links {
		items[i] = linkItem{l}
	}
	s.linkList.SetItems(items)
	// разворачиваем список ссылок на все элементы, чтобы bubbles/list не
	// прыгал страницами (в модалке список ссылок не ограничен окном).
	sizeListHeight(&s.linkList, s.linkDelegate, len(s.links), 8)
	s.refreshDesc()
}

// clearSearch сбрасывает активный поиск (используется из main.go при esc).
func (s *projectsScreen) selectedProjectID() int64 {
	if item, ok := s.list.SelectedItem().(projectItem); ok {
		return item.p.ID
	}
	return 0
}

// startNew открывает ввод имени нового проекта (из клавиши n или палитры).
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
		// список и описание равной ширины (1:1) с разделителем 2 пробела.
		// Сам разделитель (2 пробела) включаем в ширину левой колонки, чтобы
		// фон выделенной строки (серый) доходил до самого разделителя без
		// цветового шва — иначе в конце серой строки оставались бы 2 ячейки
		// фона контента.
		listW = (avail - 2) / 2
		descW = avail - 2 - listW
	} else {
		listW = avail
	}
	s.listW, s.descW = listW, descW
	frame := theme.Pane(false).GetHorizontalFrameSize()
	// Левая колонка целиком (контент + 2 пробела-разделителя) — ширина, по
	// которой рендерится список, чтобы серый фон выделенной строки доходил
	// до самого разделителя без цветового шва (иначе в конце серой строки
	// оставались бы 2 ячейки фона контента).
	colW := s.listW
	if s.descW > 0 {
		colW = s.listW + 2
	}
	s.list.SetWidth(colW)
	s.listH = max(s.midH-2, 1)
	s.sizeList()
	s.syncScroll()
	s.descV.Width = max(descW-frame, 1)
	s.descV.Height = max(s.midH-1, 1)
	s.descText.SetWidth(max(descW-frame, 1))
	s.descText.SetHeight(max(s.midH-1, 1))
	s.refreshDesc()
	s.fullW = w + sideW + rightW + 4
	s.fullH = h + 5
}

// retheme пересобирает стили делегатов списков после смены темы.
func (s *projectsScreen) retheme() {
	theme.ApplyToDelegate(&s.listDelegate.DefaultDelegate)
	s.list.SetDelegate(s.listDelegate)
	theme.ApplyToDelegate(s.linkDelegate)
	s.linkList.SetDelegate(s.linkDelegate)
}
