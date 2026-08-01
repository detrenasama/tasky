package main

// settingsScreen — страница «Настройки» (заглушка).
type settingsScreen struct{}

func newSettingsScreen() *settingsScreen { return &settingsScreen{} }

func (s *settingsScreen) load()           {}
func (s *settingsScreen) resize(w, h int) {}

func (s *settingsScreen) header(w int) string {
	return padW(headerStyle.Render("Tasky")+"  "+faint("Настройки"), w)
}

func (s *settingsScreen) footer(w int) string {
	return padW(faint("q — выход · esc — назад"), w)
}

func (s *settingsScreen) view(w, h int) string {
	return padH(dimBox.Render("Настройки — раздел в разработке."), w, h)
}

func (s *settingsScreen) dialog() (string, bool) { return "", false }
