package db

import "time"

type Project struct {
	ID        int64
	Name      string
	Desc      string
	CreatedAt time.Time
}

type Task struct {
	ID          int64
	ProjectID   int64
	Title       string
	Description string
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time
	SubCount    int
}

type SubtaskWithTime struct {
	ID           int64
	TaskID       int64
	Title        string
	Description  string
	Status       string
	SortOrder    int64
	CreatedAt    time.Time
	CompletedAt  *time.Time
	TotalSeconds int64
	ActiveSince  *int64
}

type TimeEntry struct {
	ID        int64
	SubtaskID int64
	StartedAt time.Time
	EndedAt   *time.Time
	Note      string
}

// TimeEntryInfo — запись учёта времени вместе с названиями подзадачи/задачи/
// проекта (для отчёта и обнаружения пересечений).
type TimeEntryInfo struct {
	ID           int64
	SubtaskID    int64
	SubtaskTitle string
	TaskTitle    string
	ProjectName  string
	StartedAt    time.Time
	EndedAt      *time.Time
	Note         string
}

// Link — ссылка проекта/задачи/подзадачи (поле OwnerID — id владельца).
type Link struct {
	ID        int64
	OwnerID   int64
	Name      string
	URL       string
	CreatedAt time.Time
}

type JournalEntry struct {
	ID        int64
	SubtaskID int64
	CreatedAt time.Time
	Text      string
}

// ChecklistItem — элемент чек-листа подзадачи. Status: new | in_progress |
// done | cancelled (new — не выполнено).
type ChecklistItem struct {
	ID              int64
	SubtaskID       int64
	Text            string
	Status          string
	SortOrder       int64
	CreatedAt       time.Time
	StatusChangedAt time.Time
}

// StatusDef — настраиваемый статус из каталога statuses.
type StatusDef struct {
	ID         int64
	Name       string
	Type       string // new | in_progress | done
	Color      string
	NotePrompt string
	IsQuick    bool
	SortOrder  int
}

// StatusHistoryEntry — запись истории смены статуса задачи.
type StatusHistoryEntry struct {
	From      string
	To        string
	Note      string
	CreatedAt time.Time
}

// TagType — настраиваемый тип тега из каталога tag_types. Kind: text |
// task_id (номер задачи внешнего сервиса).
type TagType struct {
	ID        int64
	Name      string
	Kind      string
	Color     string
	SortOrder int
}

// Tag — тег задачи: значение + ссылка на тип. TypeName/Kind/Color
// денормализованы из типа для отображения без джойнов.
type Tag struct {
	ID        int64
	TaskID    int64
	TypeID    int64
	TypeName  string
	Kind      string
	Color     string
	Text      string
	URL       string
	CreatedAt time.Time
}
