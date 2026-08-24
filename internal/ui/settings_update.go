package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbletea"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

// updateSettings обрабатывает клавиши страницы настроек (включая модалки
// выбора и ввода).
func (m *model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.settings
	switch s.mode {
	case settingsDirInput:
		s.dirInput, _ = s.dirInput.Update(msg)
		switch msg.String() {
		case "enter":
			v := strings.TrimSpace(s.dirInput.Value())
			if v == "" {
				v = "reports"
			}
			s.cfg.saveDir = v
			s.mode = settingsBrowse
		case "esc":
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsProjList:
		switch msg.String() {
		case "up":
			s.projPick.move(-1)
		case "down":
			s.projPick.move(1)
		case "pgup":
			s.projPick.move(-s.projPick.visible)
		case "pgdown":
			s.projPick.move(s.projPick.visible)
		case "enter":
			if it, ok := s.projPick.selected(); ok {
				s.cfg.projectID = it.value
			}
			s.mode = settingsBrowse
		case "esc":
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsPeriodList:
		switch msg.String() {
		case "up":
			s.periodPick.move(-1)
		case "down":
			s.periodPick.move(1)
		case "pgup":
			s.periodPick.move(-s.periodPick.visible)
		case "pgdown":
			s.periodPick.move(s.periodPick.visible)
		case "enter":
			if it, ok := s.periodPick.selected(); ok {
				if reportPeriod(it.value) == periodCustom {
					s.periodInput.SetValue(s.customInputValue())
					s.periodInput.Focus()
					s.lastErr = nil
					s.mode = settingsPeriodInput
				} else {
					s.cfg.period = reportPeriod(it.value)
					s.mode = settingsBrowse
				}
			}
		case "esc":
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsPeriodInput:
		s.periodInput, _ = s.periodInput.Update(msg)
		switch msg.String() {
		case "enter":
			from, to, err := parseCustomPeriod(s.periodInput.Value())
			if err != nil {
				s.lastErr = err
				return m, nil
			}
			s.cfg.customFrom, s.cfg.customTo = from, to
			s.cfg.period = periodCustom
			s.lastErr = nil
			s.mode = settingsBrowse
		case "esc":
			s.lastErr = nil
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsHideInput:
		s.hideInput, _ = s.hideInput.Update(msg)
		switch msg.String() {
		case "enter":
			n, err := parseHideDays(s.hideInput.Value())
			if err != nil {
				s.lastErr = err
				return m, nil
			}
			if err := s.store.SetSetting("hide_days", strconv.Itoa(n)); err != nil {
				s.lastErr = err
				return m, nil
			}
			s.hideDays = n
			s.lastErr = nil
			s.mode = settingsBrowse
		case "esc":
			s.lastErr = nil
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsStatusList:
		switch msg.String() {
		case "up":
			s.statusPick.move(-1)
		case "down":
			s.statusPick.move(1)
		case "pgup":
			s.statusPick.move(-s.statusPick.visible)
		case "pgdown":
			s.statusPick.move(s.statusPick.visible)
		case "enter":
			if it, ok := s.statusPick.selected(); ok {
				s.openStatusEdit(it.value)
			}
		case "n":
			s.openStatusEdit(0)
		case "d":
			if it, ok := s.statusPick.selected(); ok {
				s.statusDelID = it.value
				s.mode = settingsStatusConfirm
			}
		case "esc":
			s.lastErr = nil
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsStatusEdit:
		switch msg.String() {
		case "up":
			s.editFocus = (s.editFocus + 4) % 5
			s.focusEditField()
		case "down", "tab":
			s.editFocus = (s.editFocus + 1) % 5
			s.focusEditField()
		case "enter":
			switch s.editFocus {
			case 0:
				s.editFocus = 1
				s.focusEditField()
			case 1:
				s.editType = (s.editType + 1) % 3
			case 2:
				s.colorFromTag = false
				s.colorPick.sel = s.editColor
				s.colorPick.clampScroll()
				s.mode = settingsColorPick
			case 3:
				s.editQuick = !s.editQuick
			case 4:
				s.saveStatusEdit()
			}
		case "ctrl+s":
			s.saveStatusEdit()
		case "esc":
			s.lastErr = nil
			s.mode = settingsStatusList
		default:
			if s.editFocus == 0 {
				s.editName, _ = s.editName.Update(msg)
			} else if s.editFocus == 4 {
				s.editNote, _ = s.editNote.Update(msg)
			}
		}
		return m, nil
	case settingsColorPick:
		switch msg.String() {
		case "up":
			s.colorPick.move(-1)
		case "down":
			s.colorPick.move(1)
		case "pgup":
			s.colorPick.move(-s.colorPick.visible)
		case "pgdown":
			s.colorPick.move(s.colorPick.visible)
		case "enter":
			if it, ok := s.colorPick.selected(); ok {
				s.editColor = int(it.value)
			}
			if s.colorFromTag {
				s.mode = settingsTagTypeEdit
			} else {
				s.mode = settingsStatusEdit
			}
		case "esc":
			if s.colorFromTag {
				s.mode = settingsTagTypeEdit
			} else {
				s.mode = settingsStatusEdit
			}
		}
		return m, nil
	case settingsStatusConfirm:
		switch msg.String() {
		case "y", "enter":
			if err := s.store.DeleteStatus(s.statusDelID); err != nil {
				s.lastErr = err
			} else {
				s.lastErr = nil
				s.load()
			}
			s.mode = settingsStatusList
		case "n", "esc":
			s.mode = settingsStatusList
		}
		return m, nil
	case settingsTagTypeList:
		switch msg.String() {
		case "up":
			s.tagTypePick.move(-1)
		case "down":
			s.tagTypePick.move(1)
		case "pgup":
			s.tagTypePick.move(-s.tagTypePick.visible)
		case "pgdown":
			s.tagTypePick.move(s.tagTypePick.visible)
		case "enter":
			if it, ok := s.tagTypePick.selected(); ok {
				s.openTagTypeEdit(it.value)
			}
		case "n":
			s.openTagTypeEdit(0)
		case "d":
			if it, ok := s.tagTypePick.selected(); ok {
				s.tagTypeDelID = it.value
				s.mode = settingsTagTypeConfirm
			}
		case "esc":
			s.lastErr = nil
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsTagTypeEdit:
		switch msg.String() {
		case "up":
			s.editFocus = (s.editFocus + 2) % 3
			s.focusTagTypeField()
		case "down", "tab":
			s.editFocus = (s.editFocus + 1) % 3
			s.focusTagTypeField()
		case "enter":
			switch s.editFocus {
			case 0:
				s.editFocus = 1
				s.focusTagTypeField()
			case 1:
				s.editKind = (s.editKind + 1) % 2
			case 2:
				s.colorFromTag = true
				s.colorPick.sel = s.editColor
				s.colorPick.clampScroll()
				s.mode = settingsColorPick
			}
		case "ctrl+s":
			s.saveTagTypeEdit()
		case "esc":
			s.lastErr = nil
			s.mode = settingsTagTypeList
		default:
			if s.editFocus == 0 {
				s.editName, _ = s.editName.Update(msg)
			}
		}
		return m, nil
	case settingsTagTypeConfirm:
		switch msg.String() {
		case "y", "enter":
			if err := s.store.DeleteTagType(s.tagTypeDelID); err != nil {
				s.lastErr = err
			} else {
				s.lastErr = nil
				s.load()
			}
			s.mode = settingsTagTypeList
		case "n", "esc":
			s.mode = settingsTagTypeList
		}
		return m, nil
	case settingsThemeList:
		switch msg.String() {
		case "up":
			s.themePick.move(-1)
		case "down":
			s.themePick.move(1)
		case "pgup":
			s.themePick.move(-s.themePick.visible)
		case "pgdown":
			s.themePick.move(s.themePick.visible)
		case "enter":
			if it, ok := s.themePick.selected(); ok {
				if err := theme.Apply(it.label); err != nil {
					s.lastErr = err
				} else {
					s.lastErr = nil
					s.store.SetSetting("theme", it.label)
					m.retheme()
				}
			}
			s.mode = settingsBrowse
		case "esc":
			s.mode = settingsBrowse
		}
		return m, nil
	}

	switch msg.String() {
	case "up":
		s.sel = (s.sel + 7) % 8
	case "down":
		s.sel = (s.sel + 1) % 8
	case "enter":
		switch s.sel {
		case 0:
			s.openPeriodPick()
		case 1:
			s.openProjPick()
		case 2:
			s.cfg.includeJournal = !s.cfg.includeJournal
		case 3:
			s.dirInput.SetValue(s.cfg.saveDir)
			s.dirInput.Focus()
			s.mode = settingsDirInput
		case 4:
			s.hideInput.SetValue(strconv.Itoa(s.hideDays))
			s.hideInput.Focus()
			s.mode = settingsHideInput
		case 5:
			s.lastErr = nil
			s.mode = settingsStatusList
		case 6:
			s.lastErr = nil
			s.mode = settingsTagTypeList
		case 7:
			s.lastErr = nil
			s.themePick.items = nil
			for _, n := range theme.Themes() {
				s.themePick.items = append(s.themePick.items, pickItem{label: n})
			}
			s.themePick.sel = 0
			for i, it := range s.themePick.items {
				if it.label == theme.ActiveName() {
					s.themePick.sel = i
				}
			}
			s.themePick.clampScroll()
			s.mode = settingsThemeList
		}
	}
	return m, nil
}
