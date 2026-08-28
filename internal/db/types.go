package db

import "time"

type Project struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Desc      string    `json:"desc"`
	CreatedAt time.Time `json:"created_at"`
}

type Task struct {
	ID          int64      `json:"id"`
	ProjectID   int64      `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
	SubCount    int        `json:"sub_count"`
}

type SubtaskWithTime struct {
	ID           int64      `json:"id"`
	TaskID       int64      `json:"task_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	SortOrder    int64      `json:"sort_order"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	TotalSeconds int64      `json:"total_seconds"`
	ActiveSince  *int64     `json:"active_since"`
}

type TimeEntry struct {
	ID        int64      `json:"id"`
	SubtaskID int64      `json:"subtask_id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
	Note      string     `json:"note"`
}

// TimeEntryInfo — запись учёта времени вместе с названиями подзадачи/задачи/
// проекта (для отчёта и обнаружения пересечений).
type TimeEntryInfo struct {
	ID           int64      `json:"id"`
	SubtaskID    int64      `json:"subtask_id"`
	SubtaskTitle string     `json:"subtask_title"`
	TaskTitle    string     `json:"task_title"`
	ProjectName  string     `json:"project_name"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	Note         string     `json:"note"`
}

// Link — ссылка проекта/задачи/подзадачи (поле OwnerID — id владельца).
type Link struct {
	ID        int64     `json:"id"`
	OwnerID   int64     `json:"owner_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type JournalEntry struct {
	ID        int64     `json:"id"`
	SubtaskID int64     `json:"subtask_id"`
	CreatedAt time.Time `json:"created_at"`
	Text      string    `json:"text"`
}

// ChecklistItem — элемент чек-листа подзадачи. Status: new | in_progress |
// done | cancelled (new — не выполнено).
type ChecklistItem struct {
	ID              int64     `json:"id"`
	SubtaskID       int64     `json:"subtask_id"`
	Text            string    `json:"text"`
	Status          string    `json:"status"`
	SortOrder       int64     `json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`
	StatusChangedAt time.Time `json:"status_changed_at"`
}

// StatusDef — настраиваемый статус из каталога statuses.
type StatusDef struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Color      string `json:"color"`
	NotePrompt string `json:"note_prompt"`
	IsQuick    bool   `json:"is_quick"`
	SortOrder  int    `json:"sort_order"`
}

// StatusHistoryEntry — запись истории смены статуса задачи.
type StatusHistoryEntry struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// TagType — настраиваемый тип тега из каталога tag_types. Kind: text |
// task_id (номер задачи внешнего сервиса).
type TagType struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
}

// Tag — тег задачи: значение + ссылка на тип. TypeName/Kind/Color
// денормализованы из типа для отображения без джойнов.
type Tag struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	TypeID    int64     `json:"type_id"`
	TypeName  string    `json:"type_name"`
	Kind      string    `json:"kind"`
	Color     string    `json:"color"`
	Text      string    `json:"text"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}
