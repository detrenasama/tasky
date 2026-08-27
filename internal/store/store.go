// Package store определяет интерфейс доступа к данным приложения. Он
// реализуется двумя способами: на прямом доступе к SQLite (internal/store)
// и через gRPC-клиента (internal/client). Типы данных — из internal/db.
package store

import (
	"time"

	"github.com/detrenasama/tasky/internal/db"
)

// Store — единая точка доступа к данным: задачи, подзадачи, проекты,
// учёт времени, статусы, теги, настройки и отчёты.
type Store interface {
	// Проекты.
	Projects() ([]db.Project, error)
	CreateProject(name string) (db.Project, error)
	DeleteProject(id int64) error
	ProjectDescription(id int64) (string, error)
	UpdateProjectDescription(id int64, text string) error
	ProjectLinks(projectID int64) ([]db.Link, error)
	CreateProjectLink(projectID int64, name, url string) (db.Link, error)
	UpdateProjectLink(id int64, name, url string) error
	DeleteProjectLink(id int64) error
	ProjectLinksTexts() (map[int64]string, error)

	// Задачи и подзадачи.
	TasksByProject(projectID int64) ([]db.Task, error)
	SubtasksByProject(projectID int64) ([]db.SubtaskWithTime, error)
	SubtasksWithTime(taskID int64) ([]db.SubtaskWithTime, error)
	CreateTask(projectID int64, title string) (db.Task, error)
	DeleteTask(id int64) error
	CreateSubtask(taskID int64, title string) (db.SubtaskWithTime, error)
	DeleteSubtask(id int64) error
	MoveTask(id int64, dir int) error
	MoveSubtask(id int64, dir int) error
	UpdateTaskTitle(id int64, title string) error
	UpdateSubtaskTitle(id int64, title string) error
	TaskDescription(id int64) (string, error)
	SubtaskDescription(id int64) (string, error)
	UpdateTaskDescription(id int64, text string) error
	UpdateSubtaskDescription(id int64, text string) error

	// Ссылки задач и подзадач.
	TaskLinks(taskID int64) ([]db.Link, error)
	SubtaskLinks(subtaskID int64) ([]db.Link, error)
	CreateTaskLink(taskID int64, name, url string) (db.Link, error)
	UpdateTaskLink(id int64, name, url string) error
	CreateSubtaskLink(subtaskID int64, name, url string) (db.Link, error)
	UpdateSubtaskLink(id int64, name, url string) error
	DeleteTaskLink(id int64) error
	DeleteSubtaskLink(id int64) error

	// Журнал подзадач.
	JournalEntries(subtaskID int64) ([]db.JournalEntry, error)
	CreateJournalEntry(subtaskID int64, text string) (db.JournalEntry, error)
	UpdateJournalEntry(id int64, text string) error
	JournalTexts(projectID int64) (map[int64]string, error)

	// Чек-листы подзадач.
	ChecklistItems(subtaskID int64) ([]db.ChecklistItem, error)
	ChecklistCounts(projectID int64) (map[int64][2]int, error)
	CreateChecklistItem(subtaskID int64, text string) (db.ChecklistItem, error)
	UpdateChecklistItemText(id int64, text string) error
	SetChecklistItemStatus(id int64, status string) error
	MoveChecklistItem(id int64, dir int) error
	DeleteChecklistItem(id int64) error

	// Учёт времени.
	StartSession(subtaskID int64, now time.Time) error
	StopSession(subtaskID int64, now time.Time) error
	TimeEntriesBySubtask(subtaskID int64) ([]db.TimeEntry, error)
	UpdateTimeEntry(id int64, startedAt time.Time, endedAt *time.Time) error
	DeleteTimeEntry(id int64) error
	RunningSession() (*db.SubtaskWithTime, error)
	TodayTotal(now time.Time) (time.Duration, error)
	WeeklyTotal(now time.Time) (time.Duration, error)

	// Записи времени для отчёта и пересечений.
	TimeEntriesInRange(from, to time.Time, projectID int64) ([]db.TimeEntryInfo, error)

	// Отчёты.
	ReportEntries(from, to time.Time, projectID int64) ([]db.ReportEntry, error)
	JournalEntriesByRange(from, to time.Time) ([]db.ReportJournalEntry, error)
	TagsByTasks(taskIDs []int64) (map[int64][]db.Tag, error)

	// Каталог статусов и история.
	Statuses() ([]db.StatusDef, error)
	CreateStatus(name, typ, color, note string, quick bool) (db.StatusDef, error)
	UpdateStatus(id int64, name, typ, color, note string, quick bool) error
	DeleteStatus(id int64) error
	SetStatus(owner db.StatusOwner, id int64, to, note string, now time.Time) error
	StatusHistory(owner db.StatusOwner, id int64) ([]db.StatusHistoryEntry, error)

	// Каталог типов тегов и теги задач.
	TagTypes() ([]db.TagType, error)
	CreateTagType(name, kind, color string) (db.TagType, error)
	UpdateTagType(id int64, name, kind, color string) error
	DeleteTagType(id int64) error
	TaskTags(taskID int64) ([]db.Tag, error)
	TagsByProject(projectID int64) (map[int64][]db.Tag, error)
	CreateTag(taskID, typeID int64, text, url string) (db.Tag, error)
	UpdateTag(id, typeID int64, text, url string) error
	DeleteTag(id int64) error

	// Настройки.
	GetSetting(key string) (string, bool, error)
	SetSetting(key, value string) error
}
