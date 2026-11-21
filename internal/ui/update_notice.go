package ui

import (
	"github.com/detrenasama/tasky/changelog"
	"github.com/detrenasama/tasky/internal/update"
)

// changelogSeenKey — версия, изменения которой пользователь уже видел.
const changelogSeenKey = "seen_version"

// initChangelog решает, что показать при старте: секцию текущей версии при
// первом запуске или все секции между seen_version и текущей версией.
// Сама модалка открывается на первом WindowSizeMsg (см. openPendingChangelog),
// когда известны размеры окна.
func (m *model) initChangelog() {
	cur := update.Canonical(m.version)
	if cur == "" {
		return
	}
	seen, ok, err := m.store.GetSetting(changelogSeenKey)
	if err != nil {
		return
	}
	seen = update.Canonical(seen)
	if !ok || seen == "" {
		m.pendingChangelog = changelog.Current(cur)
		if len(m.pendingChangelog) == 0 {
			_ = m.store.SetSetting(changelogSeenKey, cur)
		}
		return
	}
	switch update.Compare(cur, seen) {
	case 0:
		return
	case -1:
		// Откат на более старую версию: молча запоминаем её, чтобы при
		// следующем обновлении снова показать изменения.
		_ = m.store.SetSetting(changelogSeenKey, cur)
	default:
		m.pendingChangelog = changelog.Between(seen, cur)
		if len(m.pendingChangelog) == 0 {
			_ = m.store.SetSetting(changelogSeenKey, cur)
		}
	}
}

// openPendingChangelog открывает отложенную модалку «Что нового» и записывает
// просмотренную версию при закрытии.
func (m *model) openPendingChangelog() {
	if len(m.pendingChangelog) == 0 || m.modal.Active() {
		return
	}
	entries := m.pendingChangelog
	m.pendingChangelog = nil
	cur := update.Canonical(m.version)
	title, body := changelogContent(entries, cur)
	w, h := changelogModalSize(m.width, m.height)
	var onClose func()
	if cur != "" {
		onClose = func() { _ = m.store.SetSetting(changelogSeenKey, cur) }
	}
	m.modal.Show(newChangelogModal(title, body, w, h, onClose))
}

// openCurrentChangelog открывает полную историю изменений до текущей версии
// из палитры команд; seen_version при этом не меняется.
func (m *model) openCurrentChangelog() {
	cur := update.Canonical(m.version)
	var entries []changelog.Entry
	if cur == "" {
		// dev-сборка: показываем весь встроенный changelog.
		entries = changelog.All()
	} else {
		// Только версии не новее текущей — старый бинарник не должен
		// показывать изменения будущих релизов.
		entries = changelog.Between("", cur)
	}
	title, body := changelogContent(entries, "")
	w, h := changelogModalSize(m.width, m.height)
	m.modal.Show(newChangelogModal(title, body, w, h, nil))
}

func changelogContent(entries []changelog.Entry, cur string) (title, body string) {
	title = "Что нового в Tasky"
	if cur != "" {
		title += " " + cur
	}
	body = changelog.Render(entries)
	if body == "" {
		if cur == "" {
			body = "Список изменений пуст."
		} else {
			body = "Для этой версии список изменений не найден."
		}
	}
	return title, body
}

// changelogModalSize подбирает размеры тела модалки под терминал.
func changelogModalSize(w, h int) (int, int) {
	maxW := dialogMaxW(w) - 6
	if maxW < 24 {
		maxW = 24
	}
	if maxW > 76 {
		maxW = 76
	}
	maxH := h - 8
	if maxH < 1 {
		maxH = 1
	}
	if maxH > 22 {
		maxH = 22
	}
	return maxW, maxH
}
