package rpc

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestProjectRoundTrip(t *testing.T) {
	in := db.Project{ID: 7, Name: "Проект", Desc: "описание", CreatedAt: time.Unix(1700000000, 0)}
	got := FromProject(ToProject(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestTaskRoundTrip(t *testing.T) {
	done := time.Unix(1700001000, 0)
	in := db.Task{ID: 3, ProjectID: 1, Title: "Задача", Description: "текст",
		Status: "В работе", CreatedAt: time.Unix(1700000000, 0), CompletedAt: &done, SubCount: 2}
	got := FromTask(ToTask(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestTaskRoundTripNilCompleted(t *testing.T) {
	in := db.Task{ID: 3, ProjectID: 1, Title: "Задача", CreatedAt: time.Unix(1700000000, 0)}
	got := FromTask(ToTask(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestSubtaskRoundTrip(t *testing.T) {
	done := time.Unix(1700001000, 0)
	active := int64(1700000500)
	in := db.SubtaskWithTime{ID: 9, TaskID: 3, Title: "Подзадача", Description: "текст",
		Status: "Новая", SortOrder: 4, CreatedAt: time.Unix(1700000000, 0),
		CompletedAt: &done, ActiveSince: &active, TotalSeconds: 300}
	got := FromSubtask(ToSubtask(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestTimeEntryRoundTrip(t *testing.T) {
	ended := time.Unix(1700003600, 0)
	in := db.TimeEntry{ID: 5, SubtaskID: 9, StartedAt: time.Unix(1700000000, 0), EndedAt: &ended, Note: "заметка"}
	got := FromTimeEntry(ToTimeEntry(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestLinkRoundTrip(t *testing.T) {
	in := db.Link{ID: 2, OwnerID: 7, Name: "сайт", URL: "https://example.com", CreatedAt: time.Unix(1700000000, 0)}
	got := FromLink(ToLink(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestJournalEntryRoundTrip(t *testing.T) {
	in := db.JournalEntry{ID: 11, SubtaskID: 9, CreatedAt: time.Unix(1700000000, 0), Text: "запись журнала"}
	got := FromJournalEntry(ToJournalEntry(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestStatusRoundTrip(t *testing.T) {
	in := db.StatusDef{ID: 2, Name: "В работе", Type: "in_progress", Color: "yellow",
		NotePrompt: "почему?", IsQuick: true, SortOrder: 3}
	got := FromStatus(ToStatus(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestStatusHistoryRoundTrip(t *testing.T) {
	in := db.StatusHistoryEntry{From: "Новая", To: "В работе", Note: "взялся",
		CreatedAt: time.Unix(1700000000, 0)}
	got := FromStatusHistory(ToStatusHistory(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestTagTypeRoundTrip(t *testing.T) {
	in := db.TagType{ID: 1, Name: "Jira", Kind: "task_id", Color: "blue", SortOrder: 2}
	got := FromTagType(ToTagType(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestTagRoundTrip(t *testing.T) {
	in := db.Tag{ID: 4, TaskID: 3, TypeID: 1, TypeName: "Jira", Kind: "task_id",
		Color: "blue", Text: "TASK-123", URL: "https://example.com/TASK-123",
		CreatedAt: time.Unix(1700000000, 0)}
	got := FromTag(ToTag(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestReportEntryRoundTrip(t *testing.T) {
	in := db.ReportEntry{ProjectID: 1, ProjectName: "П", TaskID: 3, TaskTitle: "З",
		SubtaskID: 9, SubtaskTitle: "Пз", Seconds: 150}
	got := FromReportEntry(ToReportEntry(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestReportJournalEntryRoundTrip(t *testing.T) {
	in := db.ReportJournalEntry{SubtaskID: 9, CreatedAt: time.Unix(1700000000, 0), Text: "запись"}
	got := FromReportJournalEntry(ToReportJournalEntry(in))
	if !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip: %+v != %+v", got, in)
	}
}

func TestOwnerRoundTrip(t *testing.T) {
	if FromOwner(Owner(db.OwnerTask)) != db.OwnerTask {
		t.Error("OwnerTask round-trip")
	}
	if FromOwner(Owner(db.OwnerSubtask)) != db.OwnerSubtask {
		t.Error("OwnerSubtask round-trip")
	}
}

func TestTagMapRoundTrip(t *testing.T) {
	in := map[int64][]db.Tag{
		3: {{ID: 4, TaskID: 3, Text: "TASK-123"}, {ID: 5, TaskID: 3, Text: "bug"}},
		7: nil,
	}
	got := TagMapFromProto(TagMapToProto(in))
	if len(got) != 2 || len(got[3]) != 2 || got[3][0].Text != "TASK-123" {
		t.Errorf("map round-trip: %+v", got)
	}
}

func TestDBErrorToStatus(t *testing.T) {
	se := DBErrorToStatus(db.ErrStatusInUse)
	if status.Code(se) != codes.FailedPrecondition || status.Convert(se).Message() != msgStatusInUse {
		t.Errorf("ErrStatusInUse: %v", se)
	}
	se = DBErrorToStatus(db.ErrTagTypeInUse)
	if status.Code(se) != codes.FailedPrecondition || status.Convert(se).Message() != msgTagTypeInUse {
		t.Errorf("ErrTagTypeInUse: %v", se)
	}
	if DBErrorToStatus(nil) != nil {
		t.Error("nil должен оставаться nil")
	}
	if status.Code(DBErrorToStatus(errors.New("boom"))) != codes.Internal {
		t.Error("прочая ошибка должна стать Internal")
	}
}

func TestStatusToDBError(t *testing.T) {
	if !reflect.DeepEqual(StatusToDBError(status.Error(codes.FailedPrecondition, msgStatusInUse)), db.ErrStatusInUse) {
		t.Error("StatusInUse не восстановлен")
	}
	if !reflect.DeepEqual(StatusToDBError(status.Error(codes.FailedPrecondition, msgTagTypeInUse)), db.ErrTagTypeInUse) {
		t.Error("TagTypeInUse не восстановлен")
	}
	if StatusToDBError(nil) != nil {
		t.Error("nil должен оставаться nil")
	}
	other := status.Error(codes.NotFound, "x")
	if StatusToDBError(other) != other {
		t.Error("прочие статусы должны передаваться как есть")
	}
}
