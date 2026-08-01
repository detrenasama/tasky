package main

// reportsScreen — страница «Отчеты» (заглушка).
type reportsScreen struct{}

func newReportsScreen() *reportsScreen { return &reportsScreen{} }

func (s *reportsScreen) load()           {}
func (s *reportsScreen) resize(w, h int) {}

func (s *reportsScreen) header(w int) string {
	return padW(headerStyle.Render("Tasky")+"  "+faint("Отчеты"), w)
}

func (s *reportsScreen) footer(w int) string {
	return padW(faint("q — выход · esc — назад"), w)
}

func (s *reportsScreen) view(w, h int) string {
	return padH(dimBox.Render("Отчеты — раздел в разработке."), w, h)
}

func (s *reportsScreen) dialog() (string, bool) { return "", false }
