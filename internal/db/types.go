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
