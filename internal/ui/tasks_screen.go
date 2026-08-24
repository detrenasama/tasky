package ui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"strings"
	"time"
)

type taskMode int

const (
	taskBrowse taskMode = iota
	taskInput
	taskTitleEdit
	taskConfirm
	taskDescEdit
	taskLinkEdit
	taskLinks
	taskLinkConfirm
	taskJournal
	taskStatusPick
	taskStatusNote
	taskSearch
	taskTags
	taskTagEdit
	taskTagTypePick
	taskTagConfirm
	taskChecklist
	taskChecklistConfirm
)

type taskFocus int

const (
	taskFocusList taskFocus = iota
	taskFocusDesc
)

type paneKind int

const (
	kindTask paneKind = iota
	kindSubtask
)

type taskItem struct {
	t        db.Task
	expanded bool
	scr      *tasksScreen
}

func (i taskItem) FilterValue() string { return i.t.Title }

func (i taskItem) Title() string {
	marker := "▸"
	if i.expanded {
		marker = "▾"
	}
	return statusBar(i.scr.statusColor(i.t.Status)) + " " + marker + " " +
		i.t.Title + i.scr.tagsChips(i.t.ID)
}

func (i taskItem) Description() string {
	plural := "подзадач"
	if i.t.SubCount == 1 {
		plural = "подзадача"
	}
	return i.scr.statusText(i.t.Status) + " · " + fmt.Sprintf("%d %s", i.t.SubCount, plural)
}

type subItem struct {
	st  db.SubtaskWithTime
	scr *tasksScreen
}

func (i subItem) FilterValue() string { return i.st.Title }

func (i subItem) Title() string {
	badge := ""
	if c, ok := i.scr.checklistCounts[i.st.ID]; ok && c[1] > 0 {
		badge = theme.Faint(fmt.Sprintf("[%d/%d] ", c[0], c[1]))
	}
	return "  " + statusBar(i.scr.statusColor(i.st.Status)) + " ├ " + badge + i.st.Title
}

func (i subItem) Description() string {
	text := i.scr.statusText(i.st.Status)
	total := time.Duration(i.st.TotalSeconds) * time.Second
	if i.st.ActiveSince != nil {
		elapsed := i.scr.now.Sub(time.Unix(*i.st.ActiveSince, 0))
		return "  " + text + " · идет " + fmtElapsed(elapsed)
	}
	return "  " + text + " · " + fmtDur(total)
}

type tasksScreen struct {
	store    store.Store
	projects []db.Project
	projIdx  int
	tasks    []db.Task
	subs     []db.SubtaskWithTime
	list     list.Model
	items    []list.Item
	expanded map[int64]bool
	today    time.Duration
	weekly   time.Duration
	now      time.Time

	listDelegate selDelegate
	linkDelegate *list.DefaultDelegate

	updateVer string

	entries []db.TimeEntry
	run     *db.SubtaskWithTime

	midH  int
	listW int
	descW int

	mode        taskMode
	input       textinput.Model
	inputKind   paneKind
	editID      int64
	confirmKind paneKind
	confirmID   int64
	lastErr     error

	focus taskFocus

	desc          string
	links         []db.Link
	journal       []db.JournalEntry
	history       []db.StatusHistoryEntry
	descText      textarea.Model
	descV         viewport.Model
	linkName      textinput.Model
	linkInput     textinput.Model
	linkList      list.Model
	journalText   textarea.Model
	confirmLinkID int64
	editLinkID    int64
	journalEditID int64

	statuses     []db.StatusDef
	statusPick   pickList
	statusNote   textarea.Model
	statusKind   paneKind
	statusID     int64
	statusTarget *db.StatusDef

	hideDays     int
	searchInput  textinput.Model
	searchQuery  string
	journalTexts map[int64]string

	tagTypes     []db.TagType
	tagTypePick  pickList
	tagPick      pickList
	tags         []db.Tag
	tagsMap      map[int64][]db.Tag
	tagsText     map[int64]string
	tagTaskID    int64
	tagEditID    int64
	tagEditType  int64
	tagEditText  textinput.Model
	tagEditURL   textinput.Model
	tagEditFocus int
	tagConfirmID int64

	checklistCounts    map[int64][2]int
	checklistItems     []db.ChecklistItem
	checklistPick      pickList
	checklistInput     textinput.Model
	checklistEditID    int64
	checklistNew       bool
	checklistConfirmID int64
}

func newTasksScreen(st store.Store) *tasksScreen {
	s := &tasksScreen{store: st, expanded: map[int64]bool{}, now: time.Now()}

	d := newListDelegate()
	d.ShowDescription = true
	s.listDelegate = d
	s.list = list.New(nil, d, 80, 20)
	s.list.SetShowTitle(false)
	s.list.SetShowHelp(false)
	s.list.SetShowPagination(false)
	s.list.SetShowStatusBar(false)
	s.list.DisableQuitKeybindings()
	// встроенный фильтр списка (/ + перехват букв экраном) не работает —
	// полнотекстовый поиск реализован своим режимом taskSearch
	s.list.SetFilteringEnabled(false)

	s.input = textinput.New()
	s.input.Placeholder = "Название"
	s.input.Prompt = "> "
	s.input.CharLimit = 64
	s.input.Width = 40

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
	s.descText.Placeholder = "Описание задачи или подзадачи"
	s.descText.ShowLineNumbers = false
	s.descText.SetWidth(60)
	s.descText.SetHeight(10)

	s.journalText = textarea.New()
	s.journalText.Placeholder = "Что делаю или сделал…"
	s.journalText.ShowLineNumbers = false
	s.journalText.SetWidth(60)
	s.journalText.SetHeight(10)

	s.statusNote = textarea.New()
	s.statusNote.Placeholder = "Заметка к переходу…"
	s.statusNote.ShowLineNumbers = false
	s.statusNote.SetWidth(60)
	s.statusNote.SetHeight(3)
	s.statusPick.setVisible(12)

	s.searchInput = textinput.New()
	s.searchInput.Placeholder = "название, описание, журнал…"
	s.searchInput.Prompt = "> "
	s.searchInput.CharLimit = 64
	s.searchInput.Width = 40

	s.tagEditText = textinput.New()
	s.tagEditText.Placeholder = "GW-567, 4455, …"
	s.tagEditText.Prompt = "> "
	s.tagEditText.CharLimit = 64
	s.tagEditText.Width = 40

	s.tagEditURL = textinput.New()
	s.tagEditURL.Placeholder = "https://… (необязательно)"
	s.tagEditURL.Prompt = "> "
	s.tagEditURL.CharLimit = 512
	s.tagEditURL.Width = 40

	s.tagTypePick.setVisible(12)
	s.tagPick.setVisible(12)

	s.checklistInput = textinput.New()
	s.checklistInput.Placeholder = "Текст пункта"
	s.checklistInput.Prompt = "> "
	s.checklistInput.CharLimit = 256
	s.checklistInput.Width = 50
	s.checklistPick.setVisible(12)

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

	s.descV = viewport.New(1, 1)
	return s
}

func (s *tasksScreen) load() {
	s.projects, _ = s.store.Projects()
	if s.projIdx >= len(s.projects) {
		s.projIdx = 0
	}
	s.statuses, _ = s.store.Statuses()
	s.hideDays = loadHideDays(s.store)
	items := make([]pickItem, 0, len(s.statuses))
	for _, st := range s.statuses {
		items = append(items, pickItem{value: st.ID, label: st.Name})
	}
	s.statusPick.items = items
	s.tagTypes, _ = s.store.TagTypes()
	ttItems := make([]pickItem, 0, len(s.tagTypes))
	for _, tt := range s.tagTypes {
		ttItems = append(ttItems, pickItem{value: tt.ID,
			label: colorPreview(tt.Color) + " " + tt.Name})
	}
	s.tagTypePick.items = ttItems
	s.loadData()
	s.today, _ = s.store.TodayTotal(s.now)
	s.weekly, _ = s.store.WeeklyTotal(s.now)
}

func (s *tasksScreen) resize(w, h int) {
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
	s.list.SetHeight(max(s.midH-2, 1))
	s.descV.Width = max(descW-frame, 1)
	s.descV.Height = max(s.midH-1, 1)
	s.descText.SetWidth(max(descW-frame, 1))
	s.descText.SetHeight(max(s.midH-1, 1))
	s.journalText.SetWidth(max(descW-frame, 1))
	s.journalText.SetHeight(10)
	s.checklistPick.setVisible(max(4, min(h-8, 12)))
	s.refreshDesc()
}

// retheme пересобирает стили делегатов списков после смены темы.
func (s *tasksScreen) retheme() {
	theme.ApplyToDelegate(&s.listDelegate.DefaultDelegate)
	s.list.SetDelegate(s.listDelegate)
	theme.ApplyToDelegate(s.linkDelegate)
	s.linkList.SetDelegate(s.linkDelegate)
}

func (s *tasksScreen) footer(w int) string {
	if s.mode == taskDescEdit {
		return "Ctrl+S — сохранить · Esc — отмена"
	}
	// подсказки действий перенесены в палитру команд (Ctrl+P); здесь не выводим
	return ""
}

func (s *tasksScreen) view(w, h int) string {
	leftStyle := theme.Pane(false)
	// colW — полная ширина левой колонки (контент + 2 пробела-разделителя),
	// равна ширине списка; заголовок/отступ/дополнение рендерятся через
	// theme.Pane (её фрейм 2 уже входит в colW), поэтому их ширина совпадает.
	colW := s.listW
	if s.descW > 0 {
		colW += 2
	}
	W := max(colW-leftStyle.GetHorizontalFrameSize(), 0)
	H := max(s.midH, 1)
	titleLine := theme.HeaderStyle.Render("Задачи")
	if name := s.currentProjectName(); name != "" {
		projNameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
		right := projNameStyle.Render(name)
		gap := W - lipgloss.Width(titleLine) - lipgloss.Width(right)
		if gap < 1 {
			right = projNameStyle.Render(truncateWEnd(name, max(W-lipgloss.Width(titleLine)-1, 1)))
			gap = 1
		}
		titleLine = titleLine + strings.Repeat(" ", gap) + right
	}
	titleTxt := padW(titleLine, W)
	var body string
	switch {
	case len(s.projects) == 0:
		body = renderPane(leftStyle, padLines("Нет проектов.\nНажмите p и создайте проект.", W, max(H-2, 1)))
	case s.searchQuery != "" && len(s.items) == 0:
		body = renderPane(leftStyle, padLines("Ничего не найдено по запросу\n«"+s.searchQuery+"».", W, max(H-2, 1)))
	case len(s.tasks) == 0:
		body = renderPane(leftStyle, padLines("Задач в проекте нет.", W, max(H-2, 1)))
	default:
		// bubbles/list не дополняет строки до ширины — паддинг внутри
		// делегата (selDelegate закрашивает каждую строку целиком)
		body = s.list.View()
	}
	// заголовок страницы (фон контента) + отступ 1 строка + тело списка
	// (строки несут собственный фон: серый для выделенной, контент для
	// остальных), затем дополнение до H строк фоном контента.
	left := renderPane(leftStyle, titleTxt) + "\n" + renderPane(leftStyle, padW("", W)) + "\n" + body
	lines := strings.Split(left, "\n")
	for len(lines) < H {
		lines = append(lines, renderPane(leftStyle, padW("", W)))
	}
	if len(lines) > H {
		lines = lines[:H]
	}
	left = strings.Join(lines, "\n")

	cols := []string{left}
	if s.descW > 0 {
		cols = append(cols, s.descBox())
	}
	bodyOut := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	return padH(bodyOut, w, h)
}

// rightContent возвращает содержимое правой колонки (без панели) заданной
// высоты; панель и «Tasky vX» снизу добавляет app.rightRail.
func (s *tasksScreen) rightContent(h int) string {
	bot := s.infoBottom()
	botH := len(strings.Split(bot, "\n"))
	topH := h - botH - 1
	if topH < 3 {
		topH = 3
	}
	return s.infoTop(topH) + "\n\n" + bot
}

func (s *tasksScreen) dialog() (string, bool) {
	switch s.mode {
	case taskInput:
		kind := "задача"
		if s.inputKind == kindSubtask {
			kind = "подзадача"
		}
		body := s.input.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Новая " + kind, body: body,
			primary: "Enter — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case taskTitleEdit:
		kind := "задача"
		if s.inputKind == kindSubtask {
			kind = "подзадача"
		}
		body := s.input.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Название " + kind, body: body,
			primary: "Enter — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case taskConfirm:
		name := ""
		if s.confirmKind == kindTask {
			for _, t := range s.tasks {
				if t.ID == s.confirmID {
					name = t.Title
				}
			}
		} else {
			for _, st := range s.subs {
				if st.ID == s.confirmID {
					name = st.Title
				}
			}
		}
		var d dialog
		if s.confirmKind == kindTask {
			d = dialog{title: "Удаление задачи",
				body:    fmt.Sprintf("Удалить задачу «%s» с подзадачами и временем?", name),
				primary: "y — да", esc: "n — нет"}
		} else {
			d = dialog{title: "Удаление подзадачи",
				body:    fmt.Sprintf("Удалить подзадачу «%s» и её время?", name),
				primary: "y — да", esc: "n — нет"}
		}
		return d.render(), true
	case taskDescEdit:
		return "", false
	case taskLinkEdit:
		title := "Добавить ссылку"
		if s.editLinkID != 0 {
			title = "Изменить ссылку"
		}
		body := s.linkName.View() + "\n" + s.linkInput.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: body,
			primary: "Enter — сохранить · Tab — поле", esc: "Esc — отмена"}
		return d.render(), true
	case taskLinks:
		var body string
		if len(s.links) == 0 {
			body = theme.Faint("Ссылок нет. n — добавить.")
		} else {
			body = s.linkList.View()
		}
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Не удалось открыть: "+s.lastErr.Error())
		}
		hint := "n — новая · Esc — закрыть"
		if len(s.links) > 0 {
			hint = "Enter — открыть · e — изменить · d — удалить · " + hint
		}
		d := dialog{title: "Ссылки",
			body:    body,
			primary: hint}
		return d.render(), true
	case taskLinkConfirm:
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
	case taskJournal:
		title := "Запись в журнал"
		if s.journalEditID != 0 {
			title = "Изменение записи"
		}
		body := s.journalText.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: body,
			primary: "Ctrl+S — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case taskStatusPick:
		d := dialog{title: "Статус",
			body:    s.statusPick.view(),
			primary: "Enter — выбрать", esc: "Esc — отмена"}
		return d.render(), true
	case taskStatusNote:
		title := "Заметка"
		if s.statusTarget != nil && s.statusTarget.NotePrompt != "" {
			title = s.statusTarget.Name + " — " + s.statusTarget.NotePrompt
		}
		body := s.statusNote.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: body,
			primary: "Ctrl+S — применить", esc: "Esc — отмена"}
		return d.render(), true
	case taskSearch:
		body := s.searchInput.View()
		if s.searchQuery != "" {
			body += "\n\n" + theme.Faint("Найдено: "+fmt.Sprint(len(s.items))+" элементов")
		}
		d := dialog{title: "Поиск", body: body,
			primary: "Enter — применить", esc: "Esc — отмена"}
		return d.render(), true
	case taskTags:
		body := s.tagPick.view()
		if len(s.tags) == 0 {
			body = theme.Faint("Тегов нет. n — добавить.")
		}
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Теги",
			body:    body,
			primary: "Enter — изменить · n — новый · d — удалить", esc: "Esc — закрыть"}
		return d.render(), true
	case taskTagEdit:
		title := "Новый тег"
		if s.tagEditID != 0 {
			title = "Тег"
		}
		name, color := s.tagTypeName(s.tagEditType)
		lines := []string{
			"Тип:      " + colorPreview(color) + " " + name,
			"Значение: " + s.tagEditText.View(),
			"URL:      " + s.tagEditURL.View(),
		}
		var body []string
		for i, l := range lines {
			if i == s.tagEditFocus {
				body = append(body, theme.HeaderStyle.Render("▸ "+l))
			} else {
				body = append(body, "  "+l)
			}
		}
		body = append(body, "", theme.Faint("Тип — Enter, Ctrl+S — сохранить"))
		inner := strings.Join(body, "\n")
		if s.lastErr != nil {
			inner += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: inner,
			primary: "Ctrl+S — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case taskTagTypePick:
		d := dialog{title: "Тип тега",
			body:    s.tagTypePick.view(),
			primary: "Enter — выбрать", esc: "Esc — отмена"}
		return d.render(), true
	case taskTagConfirm:
		label := ""
		for _, t := range s.tags {
			if t.ID == s.tagConfirmID {
				label = t.Text
			}
		}
		d := dialog{title: "Удаление тега",
			body:    fmt.Sprintf("Удалить тег «%s»?", label),
			primary: "y — удалить", esc: "n — нет"}
		return d.render(), true

	case taskChecklist:
		var body []string
		if len(s.checklistPick.items) == 0 {
			body = append(body, theme.Faint("Список пуст. n — добавить пункт."))
		} else {
			for i := range s.checklistPick.items {
				label := "  " + s.checklistPick.items[i].label
				if i == s.checklistPick.sel {
					label = theme.HeaderStyle.Render("▸ ") + s.checklistPick.items[i].label
				}
				body = append(body, label)
			}
		}
		inner := strings.Join(body, "\n")
		if s.checklistNew {
			inner += "\n\n" + s.checklistInput.View()
		}
		if s.lastErr != nil {
			inner += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		primary := "Enter — переключить"
		if s.checklistNew {
			primary = "Enter — сохранить"
		}
		d := dialog{title: "Чек-лист",
			body:    inner,
			primary: primary,
			esc:     "Esc — закрыть"}
		return d.render(), true

	case taskChecklistConfirm:
		label := ""
		for _, ci := range s.checklistItems {
			if ci.ID == s.checklistConfirmID {
				label = ci.Text
			}
		}
		d := dialog{title: "Удаление пункта чек-листа",
			body:    fmt.Sprintf("Удалить пункт «%s»?", label),
			primary: "y — удалить", esc: "n — нет"}
		return d.render(), true
	}
	return "", false
}

// descBox — средняя колонка: описание, ссылки и журнал выбранного элемента
// (прокручиваемый viewport); при редактировании описания вместо контента —
// textarea.
func (s *tasksScreen) descBox() string {
	style := theme.Pane(false)
	if s.mode == taskDescEdit {
		return renderPane(style, padLines(s.descText.View(), max(s.descW-style.GetHorizontalFrameSize(), 0), max(s.midH-style.GetVerticalFrameSize(), 0)))
	}
	return renderPane(style, s.descV.View())
}
