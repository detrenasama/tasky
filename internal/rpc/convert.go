package rpc

import (
	"errors"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Конвертации типов internal/db в proto-сообщения и обратно. Используются
// сервером (db → proto) и клиентом (proto → db).

func ToProject(p db.Project) *Project {
	return &Project{Id: p.ID, Name: p.Name, Desc: p.Desc, CreatedAt: p.CreatedAt.Unix()}
}

func FromProject(p *Project) db.Project {
	return db.Project{ID: p.Id, Name: p.Name, Desc: p.Desc,
		CreatedAt: time.Unix(p.CreatedAt, 0)}
}

func ToProjects(ps []db.Project) []*Project {
	out := make([]*Project, len(ps))
	for i, p := range ps {
		out[i] = ToProject(p)
	}
	return out
}

func FromProjects(ps []*Project) []db.Project {
	out := make([]db.Project, len(ps))
	for i, p := range ps {
		out[i] = FromProject(p)
	}
	return out
}

func ToTask(t db.Task) *Task {
	p := &Task{Id: t.ID, ProjectId: t.ProjectID, Title: t.Title,
		Description: t.Description, Status: t.Status, CreatedAt: t.CreatedAt.Unix(),
		SubCount: int32(t.SubCount)}
	if t.CompletedAt != nil {
		p.CompletedAt = Int64Ptr(t.CompletedAt.Unix())
	}
	return p
}

func FromTask(t *Task) db.Task {
	out := db.Task{ID: t.Id, ProjectID: t.ProjectId, Title: t.Title,
		Description: t.Description, Status: t.Status, CreatedAt: time.Unix(t.CreatedAt, 0),
		SubCount: int(t.SubCount)}
	if t.CompletedAt != nil {
		c := time.Unix(*t.CompletedAt, 0)
		out.CompletedAt = &c
	}
	return out
}

func ToTasks(ts []db.Task) []*Task {
	out := make([]*Task, len(ts))
	for i, t := range ts {
		out[i] = ToTask(t)
	}
	return out
}

func FromTasks(ts []*Task) []db.Task {
	out := make([]db.Task, len(ts))
	for i, t := range ts {
		out[i] = FromTask(t)
	}
	return out
}

func ToSubtask(s db.SubtaskWithTime) *SubtaskWithTime {
	p := &SubtaskWithTime{Id: s.ID, TaskId: s.TaskID, Title: s.Title,
		Description: s.Description, Status: s.Status, SortOrder: s.SortOrder,
		CreatedAt: s.CreatedAt.Unix(), TotalSeconds: s.TotalSeconds}
	if s.CompletedAt != nil {
		p.CompletedAt = Int64Ptr(s.CompletedAt.Unix())
	}
	if s.ActiveSince != nil {
		p.ActiveSince = Int64Ptr(*s.ActiveSince)
	}
	return p
}

func FromSubtask(s *SubtaskWithTime) db.SubtaskWithTime {
	out := db.SubtaskWithTime{ID: s.Id, TaskID: s.TaskId, Title: s.Title,
		Description: s.Description, Status: s.Status, SortOrder: s.SortOrder,
		CreatedAt: time.Unix(s.CreatedAt, 0), TotalSeconds: s.TotalSeconds}
	if s.CompletedAt != nil {
		c := time.Unix(*s.CompletedAt, 0)
		out.CompletedAt = &c
	}
	if s.ActiveSince != nil {
		out.ActiveSince = s.ActiveSince
	}
	return out
}

func ToSubtasks(ss []db.SubtaskWithTime) []*SubtaskWithTime {
	out := make([]*SubtaskWithTime, len(ss))
	for i, s := range ss {
		out[i] = ToSubtask(s)
	}
	return out
}

func FromSubtasks(ss []*SubtaskWithTime) []db.SubtaskWithTime {
	out := make([]db.SubtaskWithTime, len(ss))
	for i, s := range ss {
		out[i] = FromSubtask(s)
	}
	return out
}

func ToTimeEntry(e db.TimeEntry) *TimeEntry {
	p := &TimeEntry{Id: e.ID, SubtaskId: e.SubtaskID, StartedAt: e.StartedAt.Unix(),
		Note: e.Note}
	if e.EndedAt != nil {
		p.EndedAt = Int64Ptr(e.EndedAt.Unix())
	}
	return p
}

func FromTimeEntry(e *TimeEntry) db.TimeEntry {
	out := db.TimeEntry{ID: e.Id, SubtaskID: e.SubtaskId,
		StartedAt: time.Unix(e.StartedAt, 0), Note: e.Note}
	if e.EndedAt != nil {
		t := time.Unix(*e.EndedAt, 0)
		out.EndedAt = &t
	}
	return out
}

func ToTimeEntries(es []db.TimeEntry) []*TimeEntry {
	out := make([]*TimeEntry, len(es))
	for i, e := range es {
		out[i] = ToTimeEntry(e)
	}
	return out
}

func FromTimeEntries(es []*TimeEntry) []db.TimeEntry {
	out := make([]db.TimeEntry, len(es))
	for i, e := range es {
		out[i] = FromTimeEntry(e)
	}
	return out
}

func ToTimeEntryInfo(e db.TimeEntryInfo) *TimeEntryInfo {
	p := &TimeEntryInfo{Id: e.ID, SubtaskId: e.SubtaskID, SubtaskTitle: e.SubtaskTitle,
		TaskTitle: e.TaskTitle, ProjectName: e.ProjectName, StartedAt: e.StartedAt.Unix(),
		Note: e.Note}
	if e.EndedAt != nil {
		p.EndedAt = Int64Ptr(e.EndedAt.Unix())
	}
	return p
}

func FromTimeEntryInfo(e *TimeEntryInfo) db.TimeEntryInfo {
	out := db.TimeEntryInfo{ID: e.Id, SubtaskID: e.SubtaskId, SubtaskTitle: e.SubtaskTitle,
		TaskTitle: e.TaskTitle, ProjectName: e.ProjectName,
		StartedAt: time.Unix(e.StartedAt, 0), Note: e.Note}
	if e.EndedAt != nil {
		t := time.Unix(*e.EndedAt, 0)
		out.EndedAt = &t
	}
	return out
}

func ToTimeEntryInfos(es []db.TimeEntryInfo) []*TimeEntryInfo {
	out := make([]*TimeEntryInfo, len(es))
	for i, e := range es {
		out[i] = ToTimeEntryInfo(e)
	}
	return out
}

func FromTimeEntryInfos(es []*TimeEntryInfo) []db.TimeEntryInfo {
	out := make([]db.TimeEntryInfo, len(es))
	for i, e := range es {
		out[i] = FromTimeEntryInfo(e)
	}
	return out
}

func ToLink(l db.Link) *Link {
	return &Link{Id: l.ID, OwnerId: l.OwnerID, Name: l.Name, Url: l.URL,
		CreatedAt: l.CreatedAt.Unix()}
}

func FromLink(l *Link) db.Link {
	return db.Link{ID: l.Id, OwnerID: l.OwnerId, Name: l.Name, URL: l.Url,
		CreatedAt: time.Unix(l.CreatedAt, 0)}
}

func ToLinks(ls []db.Link) []*Link {
	out := make([]*Link, len(ls))
	for i, l := range ls {
		out[i] = ToLink(l)
	}
	return out
}

func FromLinks(ls []*Link) []db.Link {
	out := make([]db.Link, len(ls))
	for i, l := range ls {
		out[i] = FromLink(l)
	}
	return out
}

func ToJournalEntry(e db.JournalEntry) *JournalEntry {
	return &JournalEntry{Id: e.ID, SubtaskId: e.SubtaskID,
		CreatedAt: e.CreatedAt.Unix(), Text: e.Text}
}

func FromJournalEntry(e *JournalEntry) db.JournalEntry {
	return db.JournalEntry{ID: e.Id, SubtaskID: e.SubtaskId,
		CreatedAt: time.Unix(e.CreatedAt, 0), Text: e.Text}
}

func ToJournalEntries(es []db.JournalEntry) []*JournalEntry {
	out := make([]*JournalEntry, len(es))
	for i, e := range es {
		out[i] = ToJournalEntry(e)
	}
	return out
}

func FromJournalEntries(es []*JournalEntry) []db.JournalEntry {
	out := make([]db.JournalEntry, len(es))
	for i, e := range es {
		out[i] = FromJournalEntry(e)
	}
	return out
}

func ToStatus(st db.StatusDef) *StatusDef {
	return &StatusDef{Id: st.ID, Name: st.Name, Type: st.Type, Color: st.Color,
		NotePrompt: st.NotePrompt, IsQuick: st.IsQuick, SortOrder: int32(st.SortOrder)}
}

func FromStatus(st *StatusDef) db.StatusDef {
	return db.StatusDef{ID: st.Id, Name: st.Name, Type: st.Type, Color: st.Color,
		NotePrompt: st.NotePrompt, IsQuick: st.IsQuick, SortOrder: int(st.SortOrder)}
}

func ToStatuses(sts []db.StatusDef) []*StatusDef {
	out := make([]*StatusDef, len(sts))
	for i, st := range sts {
		out[i] = ToStatus(st)
	}
	return out
}

func FromStatuses(sts []*StatusDef) []db.StatusDef {
	out := make([]db.StatusDef, len(sts))
	for i, st := range sts {
		out[i] = FromStatus(st)
	}
	return out
}

func ToStatusHistory(e db.StatusHistoryEntry) *StatusHistoryEntry {
	return &StatusHistoryEntry{From: e.From, To: e.To, Note: e.Note,
		CreatedAt: e.CreatedAt.Unix()}
}

func FromStatusHistory(e *StatusHistoryEntry) db.StatusHistoryEntry {
	return db.StatusHistoryEntry{From: e.From, To: e.To, Note: e.Note,
		CreatedAt: time.Unix(e.CreatedAt, 0)}
}

func ToStatusHistoryEntries(es []db.StatusHistoryEntry) []*StatusHistoryEntry {
	out := make([]*StatusHistoryEntry, len(es))
	for i, e := range es {
		out[i] = ToStatusHistory(e)
	}
	return out
}

func FromStatusHistoryEntries(es []*StatusHistoryEntry) []db.StatusHistoryEntry {
	out := make([]db.StatusHistoryEntry, len(es))
	for i, e := range es {
		out[i] = FromStatusHistory(e)
	}
	return out
}

func ToTagType(t db.TagType) *TagType {
	return &TagType{Id: t.ID, Name: t.Name, Kind: t.Kind, Color: t.Color,
		SortOrder: int32(t.SortOrder)}
}

func FromTagType(t *TagType) db.TagType {
	return db.TagType{ID: t.Id, Name: t.Name, Kind: t.Kind, Color: t.Color,
		SortOrder: int(t.SortOrder)}
}

func ToTagTypes(ts []db.TagType) []*TagType {
	out := make([]*TagType, len(ts))
	for i, t := range ts {
		out[i] = ToTagType(t)
	}
	return out
}

func FromTagTypes(ts []*TagType) []db.TagType {
	out := make([]db.TagType, len(ts))
	for i, t := range ts {
		out[i] = FromTagType(t)
	}
	return out
}

func ToTag(t db.Tag) *Tag {
	return &Tag{Id: t.ID, TaskId: t.TaskID, TypeId: t.TypeID, TypeName: t.TypeName,
		Kind: t.Kind, Color: t.Color, Text: t.Text, Url: t.URL,
		CreatedAt: t.CreatedAt.Unix()}
}

func FromTag(t *Tag) db.Tag {
	return db.Tag{ID: t.Id, TaskID: t.TaskId, TypeID: t.TypeId, TypeName: t.TypeName,
		Kind: t.Kind, Color: t.Color, Text: t.Text, URL: t.Url,
		CreatedAt: time.Unix(t.CreatedAt, 0)}
}

func ToTags(ts []db.Tag) []*Tag {
	out := make([]*Tag, len(ts))
	for i, t := range ts {
		out[i] = ToTag(t)
	}
	return out
}

func FromTags(ts []*Tag) []db.Tag {
	out := make([]db.Tag, len(ts))
	for i, t := range ts {
		out[i] = FromTag(t)
	}
	return out
}

func ToReportEntry(e db.ReportEntry) *ReportEntry {
	return &ReportEntry{ProjectId: e.ProjectID, ProjectName: e.ProjectName,
		TaskId: e.TaskID, TaskTitle: e.TaskTitle, SubtaskId: e.SubtaskID,
		SubtaskTitle: e.SubtaskTitle, Seconds: e.Seconds}
}

func FromReportEntry(e *ReportEntry) db.ReportEntry {
	return db.ReportEntry{ProjectID: e.ProjectId, ProjectName: e.ProjectName,
		TaskID: e.TaskId, TaskTitle: e.TaskTitle, SubtaskID: e.SubtaskId,
		SubtaskTitle: e.SubtaskTitle, Seconds: e.Seconds}
}

func ToReportEntries(es []db.ReportEntry) []*ReportEntry {
	out := make([]*ReportEntry, len(es))
	for i, e := range es {
		out[i] = ToReportEntry(e)
	}
	return out
}

func FromReportEntries(es []*ReportEntry) []db.ReportEntry {
	out := make([]db.ReportEntry, len(es))
	for i, e := range es {
		out[i] = FromReportEntry(e)
	}
	return out
}

func ToReportJournalEntry(e db.ReportJournalEntry) *ReportJournalEntry {
	return &ReportJournalEntry{SubtaskId: e.SubtaskID, CreatedAt: e.CreatedAt.Unix(),
		Text: e.Text}
}

func FromReportJournalEntry(e *ReportJournalEntry) db.ReportJournalEntry {
	return db.ReportJournalEntry{SubtaskID: e.SubtaskId,
		CreatedAt: time.Unix(e.CreatedAt, 0), Text: e.Text}
}

func ToReportJournalEntries(es []db.ReportJournalEntry) []*ReportJournalEntry {
	out := make([]*ReportJournalEntry, len(es))
	for i, e := range es {
		out[i] = ToReportJournalEntry(e)
	}
	return out
}

func FromReportJournalEntries(es []*ReportJournalEntry) []db.ReportJournalEntry {
	out := make([]db.ReportJournalEntry, len(es))
	for i, e := range es {
		out[i] = FromReportJournalEntry(e)
	}
	return out
}

// Owner — владелец статуса.
func Owner(o db.StatusOwner) StatusOwner {
	if o == db.OwnerSubtask {
		return StatusOwner_OWNER_SUBTASK
	}
	return StatusOwner_OWNER_TASK
}

func FromOwner(o StatusOwner) db.StatusOwner {
	if o == StatusOwner_OWNER_SUBTASK {
		return db.OwnerSubtask
	}
	return db.OwnerTask
}

// TagMapToProto превращает map[id]теги в proto.TagsMapResponse.
func TagMapToProto(m map[int64][]db.Tag) *TagsMapResponse {
	out := &TagsMapResponse{Tags: make(map[int64]*TagList, len(m))}
	for id, ts := range m {
		out.Tags[id] = &TagList{Tags: ToTags(ts)}
	}
	return out
}

// TagMapFromProto превращает proto.TagsMapResponse в map[id]теги.
func TagMapFromProto(m *TagsMapResponse) map[int64][]db.Tag {
	out := make(map[int64][]db.Tag, len(m.GetTags()))
	for id, tl := range m.GetTags() {
		out[id] = FromTags(tl.GetTags())
	}
	return out
}

// Int64Ptr — указатель для proto optional-int64.
func Int64Ptr(v int64) *int64 { return &v }

// Конверсии чек-листов подзадач.
func ToChecklistItem(ci db.ChecklistItem) *ChecklistItem {
	return &ChecklistItem{Id: ci.ID, SubtaskId: ci.SubtaskID, Text: ci.Text,
		Status: ci.Status, SortOrder: ci.SortOrder, CreatedAt: ci.CreatedAt.Unix(),
		StatusChangedAt: ci.StatusChangedAt.Unix()}
}

func FromChecklistItem(ci *ChecklistItem) db.ChecklistItem {
	return db.ChecklistItem{ID: ci.Id, SubtaskID: ci.SubtaskId, Text: ci.Text,
		Status: ci.Status, SortOrder: ci.SortOrder,
		CreatedAt:       time.Unix(ci.CreatedAt, 0),
		StatusChangedAt: time.Unix(ci.StatusChangedAt, 0)}
}

func ToChecklistItems(cis []db.ChecklistItem) []*ChecklistItem {
	out := make([]*ChecklistItem, len(cis))
	for i, ci := range cis {
		out[i] = ToChecklistItem(ci)
	}
	return out
}

func FromChecklistItems(cis []*ChecklistItem) []db.ChecklistItem {
	out := make([]db.ChecklistItem, len(cis))
	for i, ci := range cis {
		out[i] = FromChecklistItem(ci)
	}
	return out
}

func ToChecklistCounts(m map[int64][2]int) *ChecklistCountsResponse {
	out := &ChecklistCountsResponse{Done: make(map[int64]int64, len(m)),
		Total: make(map[int64]int64, len(m))}
	for id, v := range m {
		out.Done[id] = int64(v[0])
		out.Total[id] = int64(v[1])
	}
	return out
}

func FromChecklistCounts(r *ChecklistCountsResponse) map[int64][2]int {
	out := make(map[int64][2]int, len(r.GetDone()))
	for id, done := range r.GetDone() {
		out[id] = [2]int{int(done), int(r.GetTotal()[id])}
	}
	return out
}

// Sentinels — имена ошибок, передаваемые в status.Message.
const (
	msgStatusInUse   = "status_in_use"
	msgTagTypeInUse  = "tag_type_in_use"
	msgUnknownStatus = "unknown_status"
)

// DBErrorToStatus переводит ошибки internal/db в gRPC-статусы. Sentinel-ошибки
// получают код FailedPrecondition и имя в message; прочее — Internal.
func DBErrorToStatus(err error) error {
	switch {
	case errors.Is(err, db.ErrStatusInUse):
		return status.Error(codes.FailedPrecondition, msgStatusInUse)
	case errors.Is(err, db.ErrTagTypeInUse):
		return status.Error(codes.FailedPrecondition, msgTagTypeInUse)
	case err == nil:
		return nil
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// StatusToDBError восстанавливает sentinel-ошибки internal/db из gRPC-статуса;
// прочие ошибки передаются как есть.
func StatusToDBError(err error) error {
	if err == nil {
		return nil
	}
	if status.Code(err) == codes.FailedPrecondition {
		switch status.Convert(err).Message() {
		case msgStatusInUse:
			return db.ErrStatusInUse
		case msgTagTypeInUse:
			return db.ErrTagTypeInUse
		}
	}
	return err
}
