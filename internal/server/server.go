// Package server — gRPC-сервис Tasky поверх store.Store (SQLite). Тонкая
// обёртка: запрос конвертируется в вызов хранилища, ответ — обратно в proto.
package server

import (
	"context"
	"time"

	"github.com/detrenasama/tasky/internal/rpc"
	"github.com/detrenasama/tasky/internal/store"
	"google.golang.org/grpc"
)

// Server реализует rpc.TaskyServer.
type Server struct {
	rpc.UnimplementedTaskyServer
	st store.Store
}

// New создаёт обработчик gRPC-сервиса поверх хранилища.
func New(st store.Store) *Server {
	return &Server{st: st}
}

// Register регистрирует сервис на gRPC-сервере.
func Register(gs *grpc.Server, st store.Store) {
	rpc.RegisterTaskyServer(gs, New(st))
}

// --- Проекты ---

func (s *Server) ListProjects(_ context.Context, _ *rpc.Empty) (*rpc.ProjectListResponse, error) {
	ps, err := s.st.Projects()
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.ProjectListResponse{Projects: rpc.ToProjects(ps)}, nil
}

func (s *Server) CreateProject(_ context.Context, req *rpc.ProjectNameRequest) (*rpc.ProjectResponse, error) {
	p, err := s.st.CreateProject(req.Name)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.ProjectResponse{Project: rpc.ToProject(p)}, nil
}

func (s *Server) DeleteProject(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteProject(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) ProjectDescription(_ context.Context, req *rpc.IDRequest) (*rpc.StringResponse, error) {
	v, err := s.st.ProjectDescription(req.Id)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.StringResponse{Value: v}, nil
}

func (s *Server) UpdateProjectDescription(_ context.Context, req *rpc.TextRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateProjectDescription(req.Id, req.Text); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) ProjectLinks(_ context.Context, req *rpc.ProjectIDRequest) (*rpc.LinkListResponse, error) {
	ls, err := s.st.ProjectLinks(req.ProjectId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.LinkListResponse{Links: rpc.ToLinks(ls)}, nil
}

func (s *Server) CreateProjectLink(_ context.Context, req *rpc.LinkOwnerRequest) (*rpc.LinkResponse, error) {
	l, err := s.st.CreateProjectLink(req.OwnerId, req.Name, req.Url)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.LinkResponse{Link: rpc.ToLink(l)}, nil
}

func (s *Server) UpdateProjectLink(_ context.Context, req *rpc.LinkNameRequest) (*rpc.LinkResponse, error) {
	if err := s.st.UpdateProjectLink(req.Id, req.Name, req.Url); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.LinkResponse{}, nil
}

func (s *Server) DeleteProjectLink(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteProjectLink(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) ProjectLinksTexts(_ context.Context, _ *rpc.Empty) (*rpc.ProjectLinksTextsResponse, error) {
	m, err := s.st.ProjectLinksTexts()
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.ProjectLinksTextsResponse{Texts: m}, nil
}

// --- Задачи и подзадачи ---

func (s *Server) TasksByProject(_ context.Context, req *rpc.ProjectIDRequest) (*rpc.TaskListResponse, error) {
	ts, err := s.st.TasksByProject(req.ProjectId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.TaskListResponse{Tasks: rpc.ToTasks(ts)}, nil
}

func (s *Server) SubtasksByProject(_ context.Context, req *rpc.ProjectIDRequest) (*rpc.SubtaskListResponse, error) {
	ss, err := s.st.SubtasksByProject(req.ProjectId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.SubtaskListResponse{Subtasks: rpc.ToSubtasks(ss)}, nil
}

func (s *Server) SubtasksWithTime(_ context.Context, req *rpc.TaskIDRequest) (*rpc.SubtaskListResponse, error) {
	ss, err := s.st.SubtasksWithTime(req.TaskId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.SubtaskListResponse{Subtasks: rpc.ToSubtasks(ss)}, nil
}

func (s *Server) CreateTask(_ context.Context, req *rpc.CreateTaskRequest) (*rpc.TaskResponse, error) {
	t, err := s.st.CreateTask(req.ProjectId, req.Title)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.TaskResponse{Task: rpc.ToTask(t)}, nil
}

func (s *Server) DeleteTask(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteTask(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) CreateSubtask(_ context.Context, req *rpc.CreateSubtaskRequest) (*rpc.SubtaskResponse, error) {
	st, err := s.st.CreateSubtask(req.TaskId, req.Title)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.SubtaskResponse{Subtask: rpc.ToSubtask(st)}, nil
}

func (s *Server) DeleteSubtask(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteSubtask(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) MoveTask(_ context.Context, req *rpc.MoveRequest) (*rpc.Empty, error) {
	if err := s.st.MoveTask(req.Id, int(req.Dir)); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) MoveSubtask(_ context.Context, req *rpc.MoveRequest) (*rpc.Empty, error) {
	if err := s.st.MoveSubtask(req.Id, int(req.Dir)); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) UpdateTaskTitle(_ context.Context, req *rpc.TitleRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateTaskTitle(req.Id, req.Title); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) UpdateSubtaskTitle(_ context.Context, req *rpc.TitleRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateSubtaskTitle(req.Id, req.Title); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) TaskDescription(_ context.Context, req *rpc.IDRequest) (*rpc.StringResponse, error) {
	v, err := s.st.TaskDescription(req.Id)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.StringResponse{Value: v}, nil
}

func (s *Server) SubtaskDescription(_ context.Context, req *rpc.IDRequest) (*rpc.StringResponse, error) {
	v, err := s.st.SubtaskDescription(req.Id)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.StringResponse{Value: v}, nil
}

func (s *Server) UpdateTaskDescription(_ context.Context, req *rpc.TextRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateTaskDescription(req.Id, req.Text); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) UpdateSubtaskDescription(_ context.Context, req *rpc.TextRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateSubtaskDescription(req.Id, req.Text); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

// --- Ссылки задач и подзадач ---

func (s *Server) TaskLinks(_ context.Context, req *rpc.TaskIDRequest) (*rpc.LinkListResponse, error) {
	ls, err := s.st.TaskLinks(req.TaskId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.LinkListResponse{Links: rpc.ToLinks(ls)}, nil
}

func (s *Server) SubtaskLinks(_ context.Context, req *rpc.SubtaskIDRequest) (*rpc.LinkListResponse, error) {
	ls, err := s.st.SubtaskLinks(req.SubtaskId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.LinkListResponse{Links: rpc.ToLinks(ls)}, nil
}

func (s *Server) CreateTaskLink(_ context.Context, req *rpc.LinkOwnerRequest) (*rpc.LinkResponse, error) {
	l, err := s.st.CreateTaskLink(req.OwnerId, req.Name, req.Url)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.LinkResponse{Link: rpc.ToLink(l)}, nil
}

func (s *Server) UpdateTaskLink(_ context.Context, req *rpc.LinkNameRequest) (*rpc.LinkResponse, error) {
	if err := s.st.UpdateTaskLink(req.Id, req.Name, req.Url); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.LinkResponse{}, nil
}

func (s *Server) CreateSubtaskLink(_ context.Context, req *rpc.LinkOwnerRequest) (*rpc.LinkResponse, error) {
	l, err := s.st.CreateSubtaskLink(req.OwnerId, req.Name, req.Url)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.LinkResponse{Link: rpc.ToLink(l)}, nil
}

func (s *Server) UpdateSubtaskLink(_ context.Context, req *rpc.LinkNameRequest) (*rpc.LinkResponse, error) {
	if err := s.st.UpdateSubtaskLink(req.Id, req.Name, req.Url); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.LinkResponse{}, nil
}

func (s *Server) DeleteTaskLink(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteTaskLink(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) DeleteSubtaskLink(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteSubtaskLink(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

// --- Журнал подзадач ---

func (s *Server) JournalEntries(_ context.Context, req *rpc.SubtaskIDRequest) (*rpc.JournalEntryListResponse, error) {
	es, err := s.st.JournalEntries(req.SubtaskId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.JournalEntryListResponse{Entries: rpc.ToJournalEntries(es)}, nil
}

func (s *Server) CreateJournalEntry(_ context.Context, req *rpc.JournalTextRequest) (*rpc.JournalEntryResponse, error) {
	e, err := s.st.CreateJournalEntry(req.Id, req.Text)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.JournalEntryResponse{Entry: rpc.ToJournalEntry(e)}, nil
}

func (s *Server) UpdateJournalEntry(_ context.Context, req *rpc.JournalTextRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateJournalEntry(req.Id, req.Text); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) JournalTexts(_ context.Context, req *rpc.ProjectIDRequest) (*rpc.JournalTextsResponse, error) {
	m, err := s.st.JournalTexts(req.ProjectId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.JournalTextsResponse{Texts: m}, nil
}

// --- Чек-листы подзадач ---

func (s *Server) ChecklistItems(_ context.Context, req *rpc.SubtaskIDRequest) (*rpc.ChecklistItemListResponse, error) {
	items, err := s.st.ChecklistItems(req.SubtaskId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.ChecklistItemListResponse{Items: rpc.ToChecklistItems(items)}, nil
}

func (s *Server) ChecklistCounts(_ context.Context, req *rpc.ProjectIDRequest) (*rpc.ChecklistCountsResponse, error) {
	m, err := s.st.ChecklistCounts(req.ProjectId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return rpc.ToChecklistCounts(m), nil
}

func (s *Server) CreateChecklistItem(_ context.Context, req *rpc.JournalTextRequest) (*rpc.ChecklistItemResponse, error) {
	it, err := s.st.CreateChecklistItem(req.Id, req.Text)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.ChecklistItemResponse{Item: rpc.ToChecklistItem(it)}, nil
}

func (s *Server) UpdateChecklistItemText(_ context.Context, req *rpc.JournalTextRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateChecklistItemText(req.Id, req.Text); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) SetChecklistItemStatus(_ context.Context, req *rpc.ChecklistStatusRequest) (*rpc.Empty, error) {
	if err := s.st.SetChecklistItemStatus(req.Id, req.Status); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) MoveChecklistItem(_ context.Context, req *rpc.MoveRequest) (*rpc.Empty, error) {
	if err := s.st.MoveChecklistItem(req.Id, int(req.Dir)); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) DeleteChecklistItem(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteChecklistItem(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

// --- Учёт времени ---

func (s *Server) StartSession(_ context.Context, req *rpc.TimeRequest) (*rpc.Empty, error) {
	if err := s.st.StartSession(req.SubtaskId, time.Unix(req.Now, 0)); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) StopSession(_ context.Context, req *rpc.TimeRequest) (*rpc.Empty, error) {
	if err := s.st.StopSession(req.SubtaskId, time.Unix(req.Now, 0)); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) TimeEntriesBySubtask(_ context.Context, req *rpc.SubtaskIDRequest) (*rpc.TimeEntryListResponse, error) {
	es, err := s.st.TimeEntriesBySubtask(req.SubtaskId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.TimeEntryListResponse{Entries: rpc.ToTimeEntries(es)}, nil
}

func (s *Server) RunningSession(_ context.Context, _ *rpc.Empty) (*rpc.RunningSessionResponse, error) {
	st, err := s.st.RunningSession()
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	if st == nil {
		return &rpc.RunningSessionResponse{}, nil
	}
	return &rpc.RunningSessionResponse{Subtask: rpc.ToSubtask(*st)}, nil
}

func (s *Server) TodayTotal(_ context.Context, _ *rpc.Empty) (*rpc.DurationResponse, error) {
	d, err := s.st.TodayTotal(time.Now())
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.DurationResponse{Seconds: int64(d / time.Second)}, nil
}

func (s *Server) WeeklyTotal(_ context.Context, _ *rpc.Empty) (*rpc.DurationResponse, error) {
	d, err := s.st.WeeklyTotal(time.Now())
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.DurationResponse{Seconds: int64(d / time.Second)}, nil
}

// --- Отчёты ---

func (s *Server) ReportEntries(_ context.Context, req *rpc.RangeRequest) (*rpc.ReportEntryListResponse, error) {
	es, err := s.st.ReportEntries(time.Unix(req.From, 0), time.Unix(req.To, 0), req.ProjectId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.ReportEntryListResponse{Entries: rpc.ToReportEntries(es)}, nil
}

func (s *Server) JournalEntriesByRange(_ context.Context, req *rpc.RangeNoProjectRequest) (*rpc.ReportJournalEntryListResponse, error) {
	es, err := s.st.JournalEntriesByRange(time.Unix(req.From, 0), time.Unix(req.To, 0))
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.ReportJournalEntryListResponse{Entries: rpc.ToReportJournalEntries(es)}, nil
}

func (s *Server) TagsByTasks(_ context.Context, req *rpc.TagsByTasksRequest) (*rpc.TagsMapResponse, error) {
	m, err := s.st.TagsByTasks(req.TaskIds)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return rpc.TagMapToProto(m), nil
}

// --- Каталог статусов и история ---

func (s *Server) ListStatuses(_ context.Context, _ *rpc.Empty) (*rpc.StatusListResponse, error) {
	sts, err := s.st.Statuses()
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.StatusListResponse{Statuses: rpc.ToStatuses(sts)}, nil
}

func (s *Server) CreateStatus(_ context.Context, req *rpc.StatusRequest) (*rpc.StatusResponse, error) {
	st, err := s.st.CreateStatus(req.Name, req.Type, req.Color, req.Note, req.Quick)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.StatusResponse{Status: rpc.ToStatus(st)}, nil
}

func (s *Server) UpdateStatus(_ context.Context, req *rpc.StatusUpdateRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateStatus(req.Id, req.Name, req.Type, req.Color, req.Note, req.Quick); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) DeleteStatus(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteStatus(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) SetStatus(_ context.Context, req *rpc.SetStatusRequest) (*rpc.Empty, error) {
	if err := s.st.SetStatus(rpc.FromOwner(req.Owner), req.Id, req.To, req.Note,
		time.Unix(req.Now, 0)); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) StatusHistory(_ context.Context, req *rpc.HistoryRequest) (*rpc.HistoryListResponse, error) {
	es, err := s.st.StatusHistory(rpc.FromOwner(req.Owner), req.Id)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.HistoryListResponse{Entries: rpc.ToStatusHistoryEntries(es)}, nil
}

// --- Каталог типов тегов и теги задач ---

func (s *Server) ListTagTypes(_ context.Context, _ *rpc.Empty) (*rpc.TagTypeListResponse, error) {
	ts, err := s.st.TagTypes()
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.TagTypeListResponse{TagTypes: rpc.ToTagTypes(ts)}, nil
}

func (s *Server) CreateTagType(_ context.Context, req *rpc.TagTypeRequest) (*rpc.TagTypeResponse, error) {
	t, err := s.st.CreateTagType(req.Name, req.Kind, req.Color)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.TagTypeResponse{TagType: rpc.ToTagType(t)}, nil
}

func (s *Server) UpdateTagType(_ context.Context, req *rpc.TagTypeUpdateRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateTagType(req.Id, req.Name, req.Kind, req.Color); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) DeleteTagType(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteTagType(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) TaskTags(_ context.Context, req *rpc.TaskIDRequest) (*rpc.TagListResponse, error) {
	ts, err := s.st.TaskTags(req.TaskId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.TagListResponse{Tags: rpc.ToTags(ts)}, nil
}

func (s *Server) TagsByProject(_ context.Context, req *rpc.ProjectIDRequest) (*rpc.TagsMapResponse, error) {
	m, err := s.st.TagsByProject(req.ProjectId)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return rpc.TagMapToProto(m), nil
}

func (s *Server) CreateTag(_ context.Context, req *rpc.TagRequest) (*rpc.TagResponse, error) {
	t, err := s.st.CreateTag(req.TaskId, req.TypeId, req.Text, req.Url)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.TagResponse{Tag: rpc.ToTag(t)}, nil
}

func (s *Server) UpdateTag(_ context.Context, req *rpc.TagUpdateRequest) (*rpc.Empty, error) {
	if err := s.st.UpdateTag(req.Id, req.TypeId, req.Text, req.Url); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

func (s *Server) DeleteTag(_ context.Context, req *rpc.IDRequest) (*rpc.Empty, error) {
	if err := s.st.DeleteTag(req.Id); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}

// --- Настройки ---

func (s *Server) GetSetting(_ context.Context, req *rpc.SettingRequest) (*rpc.GetSettingResponse, error) {
	v, ok, err := s.st.GetSetting(req.Key)
	if err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.GetSettingResponse{Value: v, Ok: ok}, nil
}

func (s *Server) SetSetting(_ context.Context, req *rpc.SettingRequest) (*rpc.Empty, error) {
	if err := s.st.SetSetting(req.Key, req.Value); err != nil {
		return nil, rpc.DBErrorToStatus(err)
	}
	return &rpc.Empty{}, nil
}
