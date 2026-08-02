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
	Status      string
	CreatedAt   time.Time
	CompletedAt *time.Time
	SubCount    int
}

type SubtaskWithTime struct {
	ID           int64
	TaskID       int64
	Title        string
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

type ProjectLink struct {
	ID        int64
	ProjectID int64
	Name      string
	URL       string
	CreatedAt time.Time
}
