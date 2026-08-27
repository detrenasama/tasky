package store

import (
	"database/sql"
	"time"

	"github.com/detrenasama/tasky/internal/db"
)

// SQLite — реализация Store на прямом доступе к SQLite. Используется
// gRPC-сервером и UI-тестами.
type SQLite struct {
	conn *sql.DB
}

// SQLite реализует Store (проверка на этапе компиляции).
var _ Store = (*SQLite)(nil)

// NewSQLite оборачивает открытое подключение к базе.
func NewSQLite(conn *sql.DB) *SQLite {
	return &SQLite{conn: conn}
}

func (s *SQLite) Projects() ([]db.Project, error) { return db.Projects(s.conn) }
func (s *SQLite) CreateProject(name string) (db.Project, error) {
	return db.CreateProject(s.conn, name)
}
func (s *SQLite) DeleteProject(id int64) error { return db.DeleteProject(s.conn, id) }
func (s *SQLite) ProjectDescription(id int64) (string, error) {
	return db.ProjectDescription(s.conn, id)
}
func (s *SQLite) UpdateProjectDescription(id int64, text string) error {
	return db.UpdateProjectDescription(s.conn, id, text)
}
func (s *SQLite) ProjectLinks(projectID int64) ([]db.Link, error) {
	return db.ProjectLinks(s.conn, projectID)
}
func (s *SQLite) CreateProjectLink(projectID int64, name, url string) (db.Link, error) {
	return db.CreateProjectLink(s.conn, projectID, name, url)
}
func (s *SQLite) UpdateProjectLink(id int64, name, url string) error {
	return db.UpdateProjectLink(s.conn, id, name, url)
}
func (s *SQLite) DeleteProjectLink(id int64) error             { return db.DeleteProjectLink(s.conn, id) }
func (s *SQLite) ProjectLinksTexts() (map[int64]string, error) { return db.ProjectLinksTexts(s.conn) }

func (s *SQLite) TasksByProject(projectID int64) ([]db.Task, error) {
	return db.TasksByProject(s.conn, projectID)
}
func (s *SQLite) SubtasksByProject(projectID int64) ([]db.SubtaskWithTime, error) {
	return db.SubtasksByProject(s.conn, projectID)
}
func (s *SQLite) SubtasksWithTime(taskID int64) ([]db.SubtaskWithTime, error) {
	return db.SubtasksWithTime(s.conn, taskID)
}
func (s *SQLite) CreateTask(projectID int64, title string) (db.Task, error) {
	return db.CreateTask(s.conn, projectID, title)
}
func (s *SQLite) DeleteTask(id int64) error { return db.DeleteTask(s.conn, id) }
func (s *SQLite) CreateSubtask(taskID int64, title string) (db.SubtaskWithTime, error) {
	return db.CreateSubtask(s.conn, taskID, title)
}
func (s *SQLite) DeleteSubtask(id int64) error        { return db.DeleteSubtask(s.conn, id) }
func (s *SQLite) MoveTask(id int64, dir int) error    { return db.MoveTask(s.conn, id, dir) }
func (s *SQLite) MoveSubtask(id int64, dir int) error { return db.MoveSubtask(s.conn, id, dir) }
func (s *SQLite) UpdateTaskTitle(id int64, title string) error {
	return db.UpdateTaskTitle(s.conn, id, title)
}
func (s *SQLite) UpdateSubtaskTitle(id int64, title string) error {
	return db.UpdateSubtaskTitle(s.conn, id, title)
}
func (s *SQLite) TaskDescription(id int64) (string, error) { return db.TaskDescription(s.conn, id) }
func (s *SQLite) SubtaskDescription(id int64) (string, error) {
	return db.SubtaskDescription(s.conn, id)
}
func (s *SQLite) UpdateTaskDescription(id int64, text string) error {
	return db.UpdateTaskDescription(s.conn, id, text)
}
func (s *SQLite) UpdateSubtaskDescription(id int64, text string) error {
	return db.UpdateSubtaskDescription(s.conn, id, text)
}

func (s *SQLite) TaskLinks(taskID int64) ([]db.Link, error) { return db.TaskLinks(s.conn, taskID) }
func (s *SQLite) SubtaskLinks(subtaskID int64) ([]db.Link, error) {
	return db.SubtaskLinks(s.conn, subtaskID)
}
func (s *SQLite) CreateTaskLink(taskID int64, name, url string) (db.Link, error) {
	return db.CreateTaskLink(s.conn, taskID, name, url)
}
func (s *SQLite) UpdateTaskLink(id int64, name, url string) error {
	return db.UpdateTaskLink(s.conn, id, name, url)
}
func (s *SQLite) CreateSubtaskLink(subtaskID int64, name, url string) (db.Link, error) {
	return db.CreateSubtaskLink(s.conn, subtaskID, name, url)
}
func (s *SQLite) UpdateSubtaskLink(id int64, name, url string) error {
	return db.UpdateSubtaskLink(s.conn, id, name, url)
}
func (s *SQLite) DeleteTaskLink(id int64) error    { return db.DeleteTaskLink(s.conn, id) }
func (s *SQLite) DeleteSubtaskLink(id int64) error { return db.DeleteSubtaskLink(s.conn, id) }

func (s *SQLite) JournalEntries(subtaskID int64) ([]db.JournalEntry, error) {
	return db.JournalEntries(s.conn, subtaskID)
}
func (s *SQLite) CreateJournalEntry(subtaskID int64, text string) (db.JournalEntry, error) {
	return db.CreateJournalEntry(s.conn, subtaskID, text)
}
func (s *SQLite) UpdateJournalEntry(id int64, text string) error {
	return db.UpdateJournalEntry(s.conn, id, text)
}
func (s *SQLite) JournalTexts(projectID int64) (map[int64]string, error) {
	return db.JournalTexts(s.conn, projectID)
}

func (s *SQLite) ChecklistItems(subtaskID int64) ([]db.ChecklistItem, error) {
	return db.ChecklistItems(s.conn, subtaskID)
}
func (s *SQLite) ChecklistCounts(projectID int64) (map[int64][2]int, error) {
	return db.ChecklistCounts(s.conn, projectID)
}
func (s *SQLite) CreateChecklistItem(subtaskID int64, text string) (db.ChecklistItem, error) {
	return db.CreateChecklistItem(s.conn, subtaskID, text)
}
func (s *SQLite) UpdateChecklistItemText(id int64, text string) error {
	return db.UpdateChecklistItemText(s.conn, id, text)
}
func (s *SQLite) SetChecklistItemStatus(id int64, status string) error {
	return db.SetChecklistItemStatus(s.conn, id, status)
}
func (s *SQLite) MoveChecklistItem(id int64, dir int) error {
	return db.MoveChecklistItem(s.conn, id, dir)
}
func (s *SQLite) DeleteChecklistItem(id int64) error { return db.DeleteChecklistItem(s.conn, id) }

func (s *SQLite) StartSession(subtaskID int64, now time.Time) error {
	return db.StartSession(s.conn, subtaskID, now)
}
func (s *SQLite) StopSession(subtaskID int64, now time.Time) error {
	return db.StopSession(s.conn, subtaskID, now)
}
func (s *SQLite) TimeEntriesBySubtask(subtaskID int64) ([]db.TimeEntry, error) {
	return db.TimeEntriesBySubtask(s.conn, subtaskID)
}
func (s *SQLite) UpdateTimeEntry(id int64, startedAt time.Time, endedAt *time.Time) error {
	return db.UpdateTimeEntry(s.conn, id, startedAt, endedAt)
}
func (s *SQLite) DeleteTimeEntry(id int64) error { return db.DeleteTimeEntry(s.conn, id) }
func (s *SQLite) TimeEntriesInRange(from, to time.Time, projectID int64) ([]db.TimeEntryInfo, error) {
	return db.TimeEntriesInRange(s.conn, from, to, projectID)
}
func (s *SQLite) RunningSession() (*db.SubtaskWithTime, error) { return db.RunningSession(s.conn) }
func (s *SQLite) TodayTotal(now time.Time) (time.Duration, error) {
	return db.TodayTotal(s.conn, now)
}
func (s *SQLite) WeeklyTotal(now time.Time) (time.Duration, error) {
	return db.WeeklyTotal(s.conn, now)
}

func (s *SQLite) ReportEntries(from, to time.Time, projectID int64) ([]db.ReportEntry, error) {
	return db.ReportEntries(s.conn, from, to, projectID)
}
func (s *SQLite) JournalEntriesByRange(from, to time.Time) ([]db.ReportJournalEntry, error) {
	return db.JournalEntriesByRange(s.conn, from, to)
}
func (s *SQLite) TagsByTasks(taskIDs []int64) (map[int64][]db.Tag, error) {
	return db.TagsByTasks(s.conn, taskIDs)
}

func (s *SQLite) Statuses() ([]db.StatusDef, error) { return db.Statuses(s.conn) }
func (s *SQLite) CreateStatus(name, typ, color, note string, quick bool) (db.StatusDef, error) {
	return db.CreateStatus(s.conn, name, typ, color, note, quick)
}
func (s *SQLite) UpdateStatus(id int64, name, typ, color, note string, quick bool) error {
	return db.UpdateStatus(s.conn, id, name, typ, color, note, quick)
}
func (s *SQLite) DeleteStatus(id int64) error { return db.DeleteStatus(s.conn, id) }
func (s *SQLite) SetStatus(owner db.StatusOwner, id int64, to, note string, now time.Time) error {
	return db.SetStatus(s.conn, owner, id, to, note, now)
}
func (s *SQLite) StatusHistory(owner db.StatusOwner, id int64) ([]db.StatusHistoryEntry, error) {
	return db.StatusHistory(s.conn, owner, id)
}

func (s *SQLite) TagTypes() ([]db.TagType, error) { return db.TagTypes(s.conn) }
func (s *SQLite) CreateTagType(name, kind, color string) (db.TagType, error) {
	return db.CreateTagType(s.conn, name, kind, color)
}
func (s *SQLite) UpdateTagType(id int64, name, kind, color string) error {
	return db.UpdateTagType(s.conn, id, name, kind, color)
}
func (s *SQLite) DeleteTagType(id int64) error            { return db.DeleteTagType(s.conn, id) }
func (s *SQLite) TaskTags(taskID int64) ([]db.Tag, error) { return db.TaskTags(s.conn, taskID) }
func (s *SQLite) TagsByProject(projectID int64) (map[int64][]db.Tag, error) {
	return db.TagsByProject(s.conn, projectID)
}
func (s *SQLite) CreateTag(taskID, typeID int64, text, url string) (db.Tag, error) {
	return db.CreateTag(s.conn, taskID, typeID, text, url)
}
func (s *SQLite) UpdateTag(id, typeID int64, text, url string) error {
	return db.UpdateTag(s.conn, id, typeID, text, url)
}
func (s *SQLite) DeleteTag(id int64) error { return db.DeleteTag(s.conn, id) }

func (s *SQLite) GetSetting(key string) (string, bool, error) { return db.GetSetting(s.conn, key) }
func (s *SQLite) SetSetting(key, value string) error          { return db.SetSetting(s.conn, key, value) }
