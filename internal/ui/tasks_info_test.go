package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/db"
)

// TestSubtaskInfoParentContext — при выбранной подзадаче в правой панели
// сверху виден контекст родительской задачи: название, теги, статус, срок и
// разделитель перед данными подзадачи.
func TestSubtaskInfoParentContext(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	tt, err := db.CreateTagType(conn, "Тест-тип", "task_id", "#ff0000")
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"ABC-1", "Поездка"} {
		if _, err := db.CreateTag(conn, task.ID, tt.ID, text, ""); err != nil {
			t.Fatal(err)
		}
	}
	due := time.Now().Add(-24 * time.Hour)
	if err := db.SetTaskDates(conn, task.ID, &due, nil); err != nil {
		t.Fatal(err)
	}
	s.loadData()
	m := &model{tasks: s}
	selectFirstSubtask(m)

	plain := stripANSI(s.infoTop(30))
	for _, want := range []string{
		"Задача:", "T", "Теги:", "[ABC-1]", "[Поездка]",
		"Статус: Новая", "Срок:", "(просрочено)", "─",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("в контексте родителя нет %q:\n%s", want, plain)
		}
	}
	// строка родителя стоит до названия подзадачи
	if i, j := strings.Index(plain, "Задача:"), strings.Index(plain, "S"); i < 0 || j < 0 || i > j {
		t.Errorf("блок родителя должен быть выше подзадачи:\n%s", plain)
	}
}

// TestSubtaskInfoParentWithoutTagsOrDue — у задачи без тегов и срока
// соответствующие строки не выводятся, но название и статус остаются.
func TestSubtaskInfoParentWithoutTagsOrDue(t *testing.T) {
	_, s, _, _ := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)

	plain := stripANSI(s.infoTop(30))
	if !strings.Contains(plain, "Задача:") || !strings.Contains(plain, "T") {
		t.Errorf("нет контекста родителя:\n%s", plain)
	}
	if strings.Contains(plain, "Теги:") {
		t.Error("строка «Теги:» без тегов")
	}
	if strings.Contains(plain, "Срок:") {
		t.Error("строка «Срок:» без срока")
	}
}

// TestTaskInfoNoParentBlock — у выбранной задачи нет дублирующего блока
// «Задача:» (у неё уже есть собственный заголовок).
func TestTaskInfoNoParentBlock(t *testing.T) {
	_, s, _, _ := tasksSeedProject(t)
	plain := stripANSI(s.infoTop(30))
	if strings.Contains(plain, "Задача:") {
		t.Errorf("блок родителя показан для самой задачи:\n%s", plain)
	}
}

// TestSubtaskInfoLongParentTitle — длинное название родителя переносится по
// словам целиком (хвост не обрезается).
func TestSubtaskInfoLongParentTitle(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	long := "Очень длинное название родительской задачи с множеством слов для проверки переноса"
	if err := db.UpdateTaskTitle(conn, task.ID, long); err != nil {
		t.Fatal(err)
	}
	s.loadData()
	m := &model{tasks: s}
	selectFirstSubtask(m)

	plain := stripANSI(s.infoTop(40))
	if !strings.Contains(plain, "переноса") {
		t.Errorf("хвост длинного названия обрезан:\n%s", plain)
	}
	for _, line := range strings.Split(plain, "\n") {
		if lipgloss.Width(line) > rightW-4 {
			t.Errorf("строка шире панели (%d): %q", lipgloss.Width(line), line)
		}
	}
}

// TestSubtaskInfoManyTags — чипы тегов упаковываются по строкам и не теряются.
func TestSubtaskInfoManyTags(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	tt, err := db.CreateTagType(conn, "Тест-тип", "task_id", "#00ff00")
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, text := range []string{"AAA-111", "BBB-222", "CCC-333", "DDD-444", "EEE-555", "FFF-666"} {
		if _, err := db.CreateTag(conn, task.ID, tt.ID, text, ""); err != nil {
			t.Fatal(err)
		}
		texts = append(texts, "["+text+"]")
	}
	s.loadData()
	m := &model{tasks: s}
	selectFirstSubtask(m)

	plain := stripANSI(s.infoTop(40))
	for _, tag := range texts {
		if !strings.Contains(plain, tag) {
			t.Errorf("потерян тег %s:\n%s", tag, plain)
		}
	}
}

// TestSubtaskInfoHiddenParent — родитель, скрытый фильтром завершённых, всё
// равно показывается в контексте (он берётся из s.tasks, а не из списка).
func TestSubtaskInfoHiddenParent(t *testing.T) {
	conn, s, task, st := tasksSeedProject(t)
	at := time.Now().Add(-8 * 24 * time.Hour)
	if err := db.SetStatus(conn, db.OwnerTask, task.ID, "Выполнена", "", at); err != nil {
		t.Fatal(err)
	}
	s.load()

	s.searchQuery = "S"
	s.buildItems()
	s.selectByKindID(kindSubtask, st.ID)

	plain := stripANSI(s.infoTop(40))
	if !strings.Contains(plain, "Задача:") || !strings.Contains(plain, "T") {
		t.Errorf("скрытый родитель не показан в контексте:\n%s", plain)
	}
}

// TestTagsLinesWrap — хелпер упаковки чипов не превышает заданную ширину и не
// разрывает метки.
func TestTagsLinesWrap(t *testing.T) {
	tags := []db.Tag{
		{Text: "AAA-111", Color: "#ff0000"},
		{Text: "BBB-222", Color: "#00ff00"},
		{Text: "CCC-333", Color: "#0000ff"},
	}
	lines := tagsLines("Теги:", tags, 20)
	if len(lines) < 2 {
		t.Fatalf("ожидалось несколько строк, получено %d: %v", len(lines), lines)
	}
	for _, line := range lines {
		if lipgloss.Width(line) > 20 {
			t.Errorf("строка шире 20: %q (%d)", stripANSI(line), lipgloss.Width(line))
		}
	}
	plain := stripANSI(strings.Join(lines, "\n"))
	for _, tag := range tags {
		if !strings.Contains(plain, "["+tag.Text+"]") {
			t.Errorf("потерян тег %s: %s", tag.Text, plain)
		}
	}
}
