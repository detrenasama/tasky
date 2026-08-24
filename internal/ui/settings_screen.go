package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui/theme"
)

type settingsMode int

const (
	settingsBrowse settingsMode = iota
	settingsDirInput
	settingsProjList
	settingsPeriodList
	settingsPeriodInput
	settingsHideInput
	settingsStatusList
	settingsStatusEdit
	settingsColorPick
	settingsStatusConfirm
	settingsTagTypeList
	settingsTagTypeEdit
	settingsTagTypeConfirm
	settingsThemeList
)

var statusTypeNames = []string{"Новый", "В работе", "Завершённый"}
var statusTypeCodes = []string{"new", "in_progress", "done"}

var tagKindNames = []string{"текст", "ид задачи"}
var tagKindCodes = []string{"text", "task_id"}

// settingsScreen — страница «Настройки»: настройки отчёта (период, фильтр
// проекта, журнал, каталог сохранения), скрытие завершённых задач и каталог
// статусов. Значения отчёта пишутся в общий reportConfig экрана отчётов.
type settingsScreen struct {
	store       store.Store
	cfg         *reportConfig
	mode        settingsMode
	sel         int
	projects    []db.Project
	dirInput    textinput.Model
	periodInput textinput.Model
	hideInput   textinput.Model
	hideDays    int
	projPick    pickList
	periodPick  pickList
	lastErr     error

	statuses     []db.StatusDef
	statusPick   pickList
	colorPick    pickList
	editName     textinput.Model
	editNote     textinput.Model
	editType     int
	editColor    int
	editQuick    bool
	editFocus    int
	statusEditID int64
	statusDelID  int64

	tagTypes      []db.TagType
	tagTypePick   pickList
	tagTypeEditID int64
	tagTypeDelID  int64
	editKind      int
	colorFromTag  bool // палитра открыта из редактора типа тега

	themePick pickList

	midH int
}

func newSettingsScreen(st store.Store, cfg *reportConfig) *settingsScreen {
	ti := textinput.New()
	ti.Placeholder = "reports"
	pi := textinput.New()
	pi.Placeholder = "02.08.2026 или 01.08.2026-05.08.2026"

	s := &settingsScreen{store: st, cfg: cfg, dirInput: ti, periodInput: pi}
	s.hideInput = textinput.New()
	s.hideInput.Placeholder = "7"
	s.hideInput.Prompt = "> "
	s.projPick.setVisible(12)
	s.periodPick.setVisible(12)
	s.statusPick.setVisible(12)
	s.colorPick.setVisible(12)
	s.tagTypePick.setVisible(12)
	s.themePick.setVisible(12)
	items := make([]pickItem, 0, len(periodNames)+1)
	for i, name := range periodNames {
		items = append(items, pickItem{value: int64(i), label: name})
	}
	items = append(items, pickItem{value: int64(periodCustom), label: "свой…"})
	s.periodPick.items = items
	s.editName = textinput.New()
	s.editName.Placeholder = "Название статуса"
	s.editName.Prompt = "> "
	s.editName.Width = 30
	s.editNote = textinput.New()
	s.editNote.Placeholder = "пусто — без заметки"
	s.editNote.Prompt = "> "
	s.editNote.Width = 30
	for i, c := range theme.StatusPalette {
		s.colorPick.items = append(s.colorPick.items, pickItem{
			value: int64(i),
			label: colorPreview(c) + " " + theme.PaletteNames[i],
		})
	}
	s.load()
	return s
}

func (s *settingsScreen) load() {
	s.projects, _ = s.store.Projects()
	if s.cfg.projectID != 0 {
		found := false
		for _, p := range s.projects {
			if p.ID == s.cfg.projectID {
				found = true
			}
		}
		if !found {
			s.cfg.projectID = 0
		}
	}
	items := make([]pickItem, 0, len(s.projects)+1)
	items = append(items, pickItem{value: 0, label: "все проекты"})
	for _, p := range s.projects {
		items = append(items, pickItem{value: p.ID, label: p.Name})
	}
	s.projPick.items = items
	s.statuses, _ = s.store.Statuses()
	sItems := make([]pickItem, 0, len(s.statuses))
	for _, st := range s.statuses {
		sItems = append(sItems, pickItem{value: st.ID, label: st.Name})
	}
	s.statusPick.items = sItems
	s.tagTypes, _ = s.store.TagTypes()
	tItems := make([]pickItem, 0, len(s.tagTypes))
	for _, tt := range s.tagTypes {
		tItems = append(tItems, pickItem{value: tt.ID, label: tt.Name})
	}
	s.tagTypePick.items = tItems
	s.hideDays = loadHideDays(s.store)
}

// loadHideDays читает порог скрытия завершённых задач из БД (по умолчанию 7).
func loadHideDays(st store.Store) int {
	v, ok, err := st.GetSetting("hide_days")
	if err != nil || !ok {
		return 7
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 {
		return 7
	}
	return n
}

func (s *settingsScreen) resize(w, h int) {
	s.midH = h
	visible := max(4, min(h-8, 12))
	s.projPick.setVisible(visible)
	s.periodPick.setVisible(visible)
	s.statusPick.setVisible(visible)
	s.colorPick.setVisible(visible)
	s.tagTypePick.setVisible(visible)
	s.themePick.setVisible(visible)
}

func (s *settingsScreen) footer(w int) string {
	return "↑/↓ — выбор · Enter — изменить · esc — назад"
}

// rightContent — для настроек правая колонка пока пустая (только «Tasky vX» снизу).
func (s *settingsScreen) rightContent(h int) string {
	return ""
}

// projectName — название проекта фильтра или «все проекты».
func (s *settingsScreen) projectName() string {
	if s.cfg.projectID == 0 {
		return "все проекты"
	}
	for _, p := range s.projects {
		if p.ID == s.cfg.projectID {
			return p.Name
		}
	}
	return "все проекты"
}

// periodName — текущий период для строки настроек.
func (s *settingsScreen) periodName() string {
	if s.cfg.period != periodCustom {
		return periodNames[s.cfg.period]
	}
	if s.cfg.customFrom.IsZero() {
		return "свой…"
	}
	if s.cfg.customTo.Sub(s.cfg.customFrom) > 24*time.Hour {
		return "свой · " + s.cfg.customFrom.Format("02.01.2006") + " – " +
			s.cfg.customTo.AddDate(0, 0, -1).Format("02.01.2006")
	}
	return "свой · " + s.cfg.customFrom.Format("02.01.2006")
}

func (s *settingsScreen) dirName() string {
	if s.cfg.saveDir == "" {
		return "reports"
	}
	return s.cfg.saveDir
}

func (s *settingsScreen) hideName() string {
	if s.hideDays <= 0 {
		return "выкл"
	}
	return fmt.Sprintf("%d дн", s.hideDays)
}

func (s *settingsScreen) view(w, h int) string {
	rows := []string{
		"Период:  " + s.periodName(),
		"Проект:  " + s.projectName(),
		"Журнал:  " + boolWord(s.cfg.includeJournal),
		"Каталог: " + s.dirName(),
		"Скрытие: " + s.hideName() + " (завершённые)",
		"Статусы: " + fmt.Sprintf("%d", len(s.statuses)),
		"Типы тегов: " + fmt.Sprintf("%d", len(s.tagTypes)),
		"Тема:    " + theme.ActiveName(),
	}
	var lines []string
	for i, r := range rows {
		if i == s.sel {
			lines = append(lines, theme.HeaderStyle.Render("▸ "+r))
		} else {
			lines = append(lines, "  "+r)
		}
	}
	inner := strings.Join(lines, "\n") + "\n\n" +
		theme.Faint("Enter — выбор из списка (журнал — вкл/выкл,")
	inner += "\n" + theme.Faint("каталог — ввод пути, скрытие — дни, статусы и типы тегов, тема — каталоги)")
	// название страницы + нижний отступ 1 строка
	title := renderPane(theme.Pane(false), padLines(theme.HeaderStyle.Render("Настройки"), max(w-2, 1), 2))
	style := theme.Pane(false)
	body := title + "\n" + renderPane(style, padLines(inner, max(w-2, 1), max(h-2-style.GetVerticalFrameSize(), 1)))
	return padH(body, w, h)
}

func boolWord(b bool) string {
	if b {
		return "вкл"
	}
	return "выкл"
}
