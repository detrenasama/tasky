package ui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui/theme"
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
