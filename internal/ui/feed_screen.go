package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui/theme"
)

type feedItem struct {
	it db.FeedItem
}

func (i feedItem) FilterValue() string { return i.it.Title }

func (i feedItem) Title() string {
	prefix := ""
	if !i.it.IsRead {
		prefix = theme.ErrorStyle.Render("● ") // red dot
	} else {
		prefix = theme.Faint("  ")
	}
	kindLabel := feedKindLabel(i.it.Kind)
	ts := i.it.CreatedAt.Format("02.01")
	return prefix + kindLabel + " · " + i.it.Title + theme.Faint(" "+ts)
}

func (i feedItem) Description() string {
	if i.it.Body != "" {
		lines := strings.Split(i.it.Body, "\n")
		return truncateW(lines[0], 60)
	}
	return feedKindLabel(i.it.Kind)
}

func feedKindLabel(k string) string {
	switch k {
	case "overdue":
		return "Просрочено"
	case "due_today":
		return "Срок сегодня"
	case "due_tomorrow":
		return "Срок завтра"
	case "due_week":
		return "Срок неделя"
	case "scheduled_today":
		return "Запланировано"
	case "update":
		return "Обновление"
	default:
		if strings.HasPrefix(k, "plugin:") {
			return strings.TrimPrefix(k, "plugin:")
		}
		return k
	}
}

type feedScreen struct {
	store        store.Store
	items        []db.FeedItem
	filtered     []db.FeedItem
	list         list.Model
	descV        viewport.Model
	filterKind   string
	unreadOnly   bool
	showArchived bool
	midH         int
	listW        int
	descW        int
	listDelegate selDelegate
	lastErr      error
}

func newFeedScreen(st store.Store) *feedScreen {
	s := &feedScreen{store: st}
	d := newListDelegate()
	d.ShowDescription = true
	s.listDelegate = d
	s.list = list.New(nil, d, 80, 20)
	s.list.SetShowTitle(false)
	s.list.SetShowHelp(false)
	s.list.SetShowPagination(false)
	s.list.SetShowStatusBar(false)
	s.list.DisableQuitKeybindings()
	s.list.SetFilteringEnabled(false)
	s.descV = viewport.New(1, 1)
	return s
}

func (s *feedScreen) load() {
	filter := db.FeedFilter{
		Kind:            s.filterKind,
		UnreadOnly:      s.unreadOnly,
		IncludeArchived: s.showArchived,
	}
	items, _ := s.store.FeedItems(filter)
	s.items = items
	s.rebuild()
}

func (s *feedScreen) rebuild() {
	// items already filtered by store; just build list
	listItems := make([]list.Item, len(s.items))
	s.filtered = s.items
	for i, it := range s.items {
		listItems[i] = feedItem{it: it}
	}
	s.list.SetItems(listItems)
	s.refreshDesc()
}

func (s *feedScreen) selected() *db.FeedItem {
	if it, ok := s.list.SelectedItem().(feedItem); ok {
		// find in filtered to get pointer? return copy
		for i := range s.filtered {
			if s.filtered[i].ID == it.it.ID {
				return &s.filtered[i]
			}
		}
	}
	return nil
}

func (s *feedScreen) markSelectedRead() {
	sel := s.selected()
	if sel == nil || sel.IsRead {
		return
	}
	_ = s.store.MarkFeedRead(sel.ID)
	sel.IsRead = true
	// update backing items slice
	for i := range s.items {
		if s.items[i].ID == sel.ID {
			s.items[i].IsRead = true
			break
		}
	}
	// refresh list item title (red dot)
	s.rebuild()
}

func (s *feedScreen) refreshDesc() {
	sel := s.selected()
	w := max(s.descV.Width, 1)
	var body []string
	if sel == nil {
		if len(s.items) == 0 {
			body = append(body, theme.Faint("Лента пуста."))
			if !s.showArchived {
				body = append(body, "", theme.Faint("a — показать архив"))
			}
		} else {
			body = append(body, theme.Faint("Выберите запись."))
		}
	} else {
		body = append(body, theme.HeaderStyle.Render(sel.Title))
		body = append(body, "", theme.Faint(feedKindLabel(sel.Kind)+" · "+sel.CreatedAt.Format("02.01.2006 15:04")))
		if sel.Body != "" {
			body = append(body, "", wrapText(sel.Body, w))
		}
		if sel.RefTaskID != nil {
			body = append(body, "", theme.Faint("Задача #"+itoa(*sel.RefTaskID)))
		}
		if sel.RefSubtaskID != nil {
			body = append(body, theme.Faint("Подзадача #"+itoa(*sel.RefSubtaskID)))
		}
	}
	s.descV.SetContent(strings.Join(body, "\n"))
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func (s *feedScreen) resize(w, h int) {
	s.midH = max(h, 3)
	avail := w
	listW, descW := 0, 0
	if avail >= 100 {
		listW = (avail - 2) / 2
		descW = avail - 2 - listW
	} else {
		listW = avail
	}
	s.listW, s.descW = listW, descW
	frame := theme.Pane(false).GetHorizontalFrameSize()
	colW := s.listW
	if s.descW > 0 {
		colW = s.listW + 2
	}
	s.list.SetWidth(colW)
	s.list.SetHeight(max(s.midH-2, 1))
	s.descV.Width = max(descW-frame, 1)
	s.descV.Height = max(s.midH-1, 1)
	s.refreshDesc()
}

func (s *feedScreen) retheme() {
	theme.ApplyToDelegate(&s.listDelegate.DefaultDelegate)
	s.list.SetDelegate(s.listDelegate)
}

func (s *feedScreen) view(w, h int) string {
	title := renderPane(theme.Pane(false), padLines(theme.HeaderStyle.Render("Лента"), max(w-2, 1), 2))
	filterLine := s.filterLine(w)
	listPane := ""
	if len(s.items) == 0 {
		listPane = renderPane(theme.Pane(false), padLines(theme.Faint("Лента пуста. / — фильтр · a — архив"), max(s.listW-2, 1), max(s.midH-2, 1)))
	} else {
		// render list via its view
		listContent := s.list.View()
		listPane = renderPane(theme.Pane(false), padLines(listContent, max(s.listW-2, 1), max(s.midH-2, 1)))
	}
	descPane := renderPane(theme.Pane(false), padLines(s.descV.View(), max(s.descW-2, 1), max(s.midH-2, 1)))
	mid := ""
	if s.descW > 0 {
		// two columns
		linesL := strings.Split(listPane, "\n")
		linesR := strings.Split(descPane, "\n")
		maxL := max(len(linesL), len(linesR))
		out := make([]string, maxL)
		for i := 0; i < maxL; i++ {
			l := ""
			if i < len(linesL) {
				l = linesL[i]
			} else {
				l = padW("", s.listW)
			}
			r := ""
			if i < len(linesR) {
				r = linesR[i]
			} else {
				r = padW("", s.descW)
			}
			out[i] = l + "  " + r
		}
		mid = strings.Join(out, "\n")
	} else {
		mid = listPane
	}
	return padH(strings.Join([]string{title, filterLine, mid}, "\n"), w, h)
}

func (s *feedScreen) filterLine(w int) string {
	parts := []string{}
	if s.filterKind == "" {
		parts = append(parts, "все")
	} else {
		parts = append(parts, feedKindLabel(s.filterKind))
	}
	if s.unreadOnly {
		parts = append(parts, "непрочитанные")
	} else {
		parts = append(parts, "все записи")
	}
	if s.showArchived {
		parts = append(parts, "архив")
	}
	line := theme.Faint(strings.Join(parts, " · ")) + theme.Faint("  |  f — категория · u — непрочитанные · a — архив")
	return renderPane(theme.Pane(false), padLines(line, max(w-2, 1), 1))
}

func (s *feedScreen) footer(w int) string {
	return "↑/↓ выбор · Enter — открыть задачу · f — фильтр · u — непрочитанные · a — архив · d — удалить архив"
}

func (s *feedScreen) rightContent(h int) string {
	// лента — правая колонка как у остальных: пусто (info в центре)
	return ""
}

func (s *feedScreen) dialog() (string, bool) { return "", false }

func (s *feedScreen) isFiltering() bool { return false }

func (s *feedScreen) updateFeed(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// handle filter keys before list
	switch msg.String() {
	case "f":
		// cycle kind: "" -> overdue -> due_today -> due_tomorrow -> due_week -> scheduled_today -> update -> ""
		order := []string{"", "overdue", "due_today", "due_tomorrow", "due_week", "scheduled_today", "update"}
		idx := 0
		for i, k := range order {
			if k == s.filterKind {
				idx = i
				break
			}
		}
		s.filterKind = order[(idx+1)%len(order)]
		s.load()
		return nil, nil
	case "u":
		s.unreadOnly = !s.unreadOnly
		s.load()
		return nil, nil
	case "a":
		s.showArchived = !s.showArchived
		s.load()
		return nil, nil
	case "d":
		if s.showArchived {
			// delete archived without confirm for now (todo confirm)
			_, _ = s.store.DeleteArchivedFeed()
			s.load()
		}
		return nil, nil
	case "enter":
		sel := s.selected()
		if sel != nil {
			// could jump to task; for now just mark read
			_ = s.store.MarkFeedRead(sel.ID)
		}
		return nil, nil
	}
	// list navigation
	prevIdx := s.list.Index()
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	if s.list.Index() != prevIdx {
		s.markSelectedRead()
		s.refreshDesc()
	}
	return nil, cmd
}

var _ = time.Now
