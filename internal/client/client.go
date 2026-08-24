// Package client — gRPC-клиент сервиса Tasky, реализующий store.Store.
// Все вызовы синхронные с таймаутом 5с (локальный unix-сокет — микросекунды).
package client

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/rpc"
	"github.com/detrenasama/tasky/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// Client реализует store.Store (проверка на этапе компиляции).
var _ store.Store = (*Client)(nil)

// callTimeout — таймаут одного RPC.
const callTimeout = 5 * time.Second

// dialTimeout — таймаут установления соединения.
const dialTimeout = 3 * time.Second

// Client — реализация store.Store поверх gRPC.
type Client struct {
	conn *grpc.ClientConn
	rpc  rpc.TaskyClient
}

// Dial подключается к серверу на unix-сокете. Сначала быстрая проверка
// существования сокета и его живости (net.Dial), затем gRPC-подключение
// с ожиданием готовности. Ошибка — сервер не запущен или сокет битый.
func Dial(socketPath string) (*Client, error) {
	fi, err := os.Stat(socketPath)
	if err != nil || fi.Mode()&os.ModeSocket == 0 {
		return nil, fmt.Errorf("сокет не найден: %s", socketPath)
	}
	probe, err := net.DialTimeout("unix", socketPath, 200*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("сервер не отвечает на %s: %w", socketPath, err)
	}
	probe.Close()

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	conn, err := grpc.NewClient("unix://"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	conn.Connect()
	for conn.GetState() != connectivity.Ready {
		if !conn.WaitForStateChange(ctx, conn.GetState()) {
			conn.Close()
			return nil, ctx.Err()
		}
	}
	return &Client{conn: conn, rpc: rpc.NewTaskyClient(conn)}, nil
}

// Close закрывает соединение.
func (c *Client) Close() error { return c.conn.Close() }

func callCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), callTimeout)
}

// --- Проекты ---

func (c *Client) Projects() ([]db.Project, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.ListProjects(ctx, &rpc.Empty{})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromProjects(resp.Projects), nil
}

func (c *Client) CreateProject(name string) (db.Project, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateProject(ctx, &rpc.ProjectNameRequest{Name: name})
	if err != nil {
		return db.Project{}, rpc.StatusToDBError(err)
	}
	return rpc.FromProject(resp.Project), nil
}

func (c *Client) DeleteProject(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteProject(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

func (c *Client) ProjectDescription(id int64) (string, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.ProjectDescription(ctx, &rpc.IDRequest{Id: id})
	if err != nil {
		return "", rpc.StatusToDBError(err)
	}
	return resp.Value, nil
}

func (c *Client) UpdateProjectDescription(id int64, text string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateProjectDescription(ctx, &rpc.TextRequest{Id: id, Text: text})
	return rpc.StatusToDBError(err)
}

func (c *Client) ProjectLinks(projectID int64) ([]db.Link, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.ProjectLinks(ctx, &rpc.ProjectIDRequest{ProjectId: projectID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromLinks(resp.Links), nil
}

func (c *Client) CreateProjectLink(projectID int64, name, url string) (db.Link, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateProjectLink(ctx,
		&rpc.LinkOwnerRequest{OwnerId: projectID, Name: name, Url: url})
	if err != nil {
		return db.Link{}, rpc.StatusToDBError(err)
	}
	return rpc.FromLink(resp.Link), nil
}

func (c *Client) UpdateProjectLink(id int64, name, url string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateProjectLink(ctx,
		&rpc.LinkNameRequest{Id: id, Name: name, Url: url})
	if err != nil {
		return rpc.StatusToDBError(err)
	}
	return nil
}

func (c *Client) DeleteProjectLink(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteProjectLink(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

func (c *Client) ProjectLinksTexts() (map[int64]string, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.ProjectLinksTexts(ctx, &rpc.Empty{})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return resp.Texts, nil
}

// --- Задачи и подзадачи ---

func (c *Client) TasksByProject(projectID int64) ([]db.Task, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.TasksByProject(ctx, &rpc.ProjectIDRequest{ProjectId: projectID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromTasks(resp.Tasks), nil
}

func (c *Client) SubtasksByProject(projectID int64) ([]db.SubtaskWithTime, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.SubtasksByProject(ctx, &rpc.ProjectIDRequest{ProjectId: projectID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromSubtasks(resp.Subtasks), nil
}

func (c *Client) SubtasksWithTime(taskID int64) ([]db.SubtaskWithTime, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.SubtasksWithTime(ctx, &rpc.TaskIDRequest{TaskId: taskID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromSubtasks(resp.Subtasks), nil
}

func (c *Client) CreateTask(projectID int64, title string) (db.Task, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateTask(ctx,
		&rpc.CreateTaskRequest{ProjectId: projectID, Title: title})
	if err != nil {
		return db.Task{}, rpc.StatusToDBError(err)
	}
	return rpc.FromTask(resp.Task), nil
}

func (c *Client) DeleteTask(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteTask(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

func (c *Client) CreateSubtask(taskID int64, title string) (db.SubtaskWithTime, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateSubtask(ctx,
		&rpc.CreateSubtaskRequest{TaskId: taskID, Title: title})
	if err != nil {
		return db.SubtaskWithTime{}, rpc.StatusToDBError(err)
	}
	return rpc.FromSubtask(resp.Subtask), nil
}

func (c *Client) DeleteSubtask(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteSubtask(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

func (c *Client) MoveTask(id int64, dir int) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.MoveTask(ctx, &rpc.MoveRequest{Id: id, Dir: int32(dir)})
	return rpc.StatusToDBError(err)
}

func (c *Client) MoveSubtask(id int64, dir int) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.MoveSubtask(ctx, &rpc.MoveRequest{Id: id, Dir: int32(dir)})
	return rpc.StatusToDBError(err)
}

func (c *Client) UpdateTaskTitle(id int64, title string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateTaskTitle(ctx, &rpc.TitleRequest{Id: id, Title: title})
	return rpc.StatusToDBError(err)
}

func (c *Client) UpdateSubtaskTitle(id int64, title string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateSubtaskTitle(ctx, &rpc.TitleRequest{Id: id, Title: title})
	return rpc.StatusToDBError(err)
}

func (c *Client) TaskDescription(id int64) (string, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.TaskDescription(ctx, &rpc.IDRequest{Id: id})
	if err != nil {
		return "", rpc.StatusToDBError(err)
	}
	return resp.Value, nil
}

func (c *Client) SubtaskDescription(id int64) (string, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.SubtaskDescription(ctx, &rpc.IDRequest{Id: id})
	if err != nil {
		return "", rpc.StatusToDBError(err)
	}
	return resp.Value, nil
}

func (c *Client) UpdateTaskDescription(id int64, text string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateTaskDescription(ctx, &rpc.TextRequest{Id: id, Text: text})
	return rpc.StatusToDBError(err)
}

func (c *Client) UpdateSubtaskDescription(id int64, text string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateSubtaskDescription(ctx, &rpc.TextRequest{Id: id, Text: text})
	return rpc.StatusToDBError(err)
}

// --- Ссылки задач и подзадач ---

func (c *Client) TaskLinks(taskID int64) ([]db.Link, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.TaskLinks(ctx, &rpc.TaskIDRequest{TaskId: taskID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromLinks(resp.Links), nil
}

func (c *Client) SubtaskLinks(subtaskID int64) ([]db.Link, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.SubtaskLinks(ctx, &rpc.SubtaskIDRequest{SubtaskId: subtaskID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromLinks(resp.Links), nil
}

func (c *Client) CreateTaskLink(taskID int64, name, url string) (db.Link, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateTaskLink(ctx,
		&rpc.LinkOwnerRequest{OwnerId: taskID, Name: name, Url: url})
	if err != nil {
		return db.Link{}, rpc.StatusToDBError(err)
	}
	return rpc.FromLink(resp.Link), nil
}

func (c *Client) UpdateTaskLink(id int64, name, url string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateTaskLink(ctx,
		&rpc.LinkNameRequest{Id: id, Name: name, Url: url})
	if err != nil {
		return rpc.StatusToDBError(err)
	}
	return nil
}

func (c *Client) CreateSubtaskLink(subtaskID int64, name, url string) (db.Link, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateSubtaskLink(ctx,
		&rpc.LinkOwnerRequest{OwnerId: subtaskID, Name: name, Url: url})
	if err != nil {
		return db.Link{}, rpc.StatusToDBError(err)
	}
	return rpc.FromLink(resp.Link), nil
}

func (c *Client) UpdateSubtaskLink(id int64, name, url string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateSubtaskLink(ctx,
		&rpc.LinkNameRequest{Id: id, Name: name, Url: url})
	if err != nil {
		return rpc.StatusToDBError(err)
	}
	return nil
}

func (c *Client) DeleteTaskLink(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteTaskLink(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

func (c *Client) DeleteSubtaskLink(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteSubtaskLink(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

// --- Журнал подзадач ---

func (c *Client) JournalEntries(subtaskID int64) ([]db.JournalEntry, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.JournalEntries(ctx, &rpc.SubtaskIDRequest{SubtaskId: subtaskID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromJournalEntries(resp.Entries), nil
}

func (c *Client) CreateJournalEntry(subtaskID int64, text string) (db.JournalEntry, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateJournalEntry(ctx,
		&rpc.JournalTextRequest{Id: subtaskID, Text: text})
	if err != nil {
		return db.JournalEntry{}, rpc.StatusToDBError(err)
	}
	return rpc.FromJournalEntry(resp.Entry), nil
}

func (c *Client) UpdateJournalEntry(id int64, text string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateJournalEntry(ctx, &rpc.JournalTextRequest{Id: id, Text: text})
	return rpc.StatusToDBError(err)
}

func (c *Client) JournalTexts(projectID int64) (map[int64]string, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.JournalTexts(ctx, &rpc.ProjectIDRequest{ProjectId: projectID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return resp.Texts, nil
}

func (c *Client) ChecklistItems(subtaskID int64) ([]db.ChecklistItem, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.ChecklistItems(ctx, &rpc.SubtaskIDRequest{SubtaskId: subtaskID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromChecklistItems(resp.Items), nil
}

func (c *Client) ChecklistCounts(projectID int64) (map[int64][2]int, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.ChecklistCounts(ctx, &rpc.ProjectIDRequest{ProjectId: projectID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromChecklistCounts(resp), nil
}

func (c *Client) CreateChecklistItem(subtaskID int64, text string) (db.ChecklistItem, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateChecklistItem(ctx,
		&rpc.JournalTextRequest{Id: subtaskID, Text: text})
	if err != nil {
		return db.ChecklistItem{}, rpc.StatusToDBError(err)
	}
	return rpc.FromChecklistItem(resp.Item), nil
}

func (c *Client) UpdateChecklistItemText(id int64, text string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateChecklistItemText(ctx, &rpc.JournalTextRequest{Id: id, Text: text})
	return rpc.StatusToDBError(err)
}

func (c *Client) SetChecklistItemStatus(id int64, status string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.SetChecklistItemStatus(ctx, &rpc.ChecklistStatusRequest{Id: id, Status: status})
	return rpc.StatusToDBError(err)
}

func (c *Client) MoveChecklistItem(id int64, dir int) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.MoveChecklistItem(ctx, &rpc.MoveRequest{Id: id, Dir: int32(dir)})
	return rpc.StatusToDBError(err)
}

func (c *Client) DeleteChecklistItem(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteChecklistItem(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

// --- Учёт времени ---

func (c *Client) StartSession(subtaskID int64, now time.Time) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.StartSession(ctx,
		&rpc.TimeRequest{SubtaskId: subtaskID, Now: now.Unix()})
	return rpc.StatusToDBError(err)
}

func (c *Client) StopSession(subtaskID int64, now time.Time) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.StopSession(ctx,
		&rpc.TimeRequest{SubtaskId: subtaskID, Now: now.Unix()})
	return rpc.StatusToDBError(err)
}

func (c *Client) TimeEntriesBySubtask(subtaskID int64) ([]db.TimeEntry, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.TimeEntriesBySubtask(ctx, &rpc.SubtaskIDRequest{SubtaskId: subtaskID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromTimeEntries(resp.Entries), nil
}

func (c *Client) RunningSession() (*db.SubtaskWithTime, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.RunningSession(ctx, &rpc.Empty{})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	if resp.Subtask == nil {
		return nil, nil
	}
	st := rpc.FromSubtask(resp.Subtask)
	return &st, nil
}

func (c *Client) TodayTotal(now time.Time) (time.Duration, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.TodayTotal(ctx, &rpc.Empty{})
	if err != nil {
		return 0, rpc.StatusToDBError(err)
	}
	return time.Duration(resp.Seconds) * time.Second, nil
}

func (c *Client) WeeklyTotal(now time.Time) (time.Duration, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.WeeklyTotal(ctx, &rpc.Empty{})
	if err != nil {
		return 0, rpc.StatusToDBError(err)
	}
	return time.Duration(resp.Seconds) * time.Second, nil
}

// --- Отчёты ---

func (c *Client) ReportEntries(from, to time.Time, projectID int64) ([]db.ReportEntry, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.ReportEntries(ctx, &rpc.RangeRequest{
		From: from.Unix(), To: to.Unix(), ProjectId: projectID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromReportEntries(resp.Entries), nil
}

func (c *Client) JournalEntriesByRange(from, to time.Time) ([]db.ReportJournalEntry, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.JournalEntriesByRange(ctx,
		&rpc.RangeNoProjectRequest{From: from.Unix(), To: to.Unix()})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromReportJournalEntries(resp.Entries), nil
}

func (c *Client) TagsByTasks(taskIDs []int64) (map[int64][]db.Tag, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.TagsByTasks(ctx, &rpc.TagsByTasksRequest{TaskIds: taskIDs})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.TagMapFromProto(resp), nil
}

// --- Каталог статусов и история ---

func (c *Client) Statuses() ([]db.StatusDef, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.ListStatuses(ctx, &rpc.Empty{})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromStatuses(resp.Statuses), nil
}

func (c *Client) CreateStatus(name, typ, color, note string, quick bool) (db.StatusDef, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateStatus(ctx,
		&rpc.StatusRequest{Name: name, Type: typ, Color: color, Note: note, Quick: quick})
	if err != nil {
		return db.StatusDef{}, rpc.StatusToDBError(err)
	}
	return rpc.FromStatus(resp.Status), nil
}

func (c *Client) UpdateStatus(id int64, name, typ, color, note string, quick bool) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateStatus(ctx, &rpc.StatusUpdateRequest{
		Id: id, Name: name, Type: typ, Color: color, Note: note, Quick: quick})
	return rpc.StatusToDBError(err)
}

func (c *Client) DeleteStatus(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteStatus(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

func (c *Client) SetStatus(owner db.StatusOwner, id int64, to, note string, now time.Time) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.SetStatus(ctx, &rpc.SetStatusRequest{
		Owner: rpc.Owner(owner), Id: id, To: to, Note: note, Now: now.Unix()})
	return rpc.StatusToDBError(err)
}

func (c *Client) StatusHistory(owner db.StatusOwner, id int64) ([]db.StatusHistoryEntry, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.StatusHistory(ctx,
		&rpc.HistoryRequest{Owner: rpc.Owner(owner), Id: id})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromStatusHistoryEntries(resp.Entries), nil
}

// --- Каталог типов тегов и теги задач ---

func (c *Client) TagTypes() ([]db.TagType, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.ListTagTypes(ctx, &rpc.Empty{})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromTagTypes(resp.TagTypes), nil
}

func (c *Client) CreateTagType(name, kind, color string) (db.TagType, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateTagType(ctx,
		&rpc.TagTypeRequest{Name: name, Kind: kind, Color: color})
	if err != nil {
		return db.TagType{}, rpc.StatusToDBError(err)
	}
	return rpc.FromTagType(resp.TagType), nil
}

func (c *Client) UpdateTagType(id int64, name, kind, color string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateTagType(ctx,
		&rpc.TagTypeUpdateRequest{Id: id, Name: name, Kind: kind, Color: color})
	return rpc.StatusToDBError(err)
}

func (c *Client) DeleteTagType(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteTagType(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

func (c *Client) TaskTags(taskID int64) ([]db.Tag, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.TaskTags(ctx, &rpc.TaskIDRequest{TaskId: taskID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.FromTags(resp.Tags), nil
}

func (c *Client) TagsByProject(projectID int64) (map[int64][]db.Tag, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.TagsByProject(ctx, &rpc.ProjectIDRequest{ProjectId: projectID})
	if err != nil {
		return nil, rpc.StatusToDBError(err)
	}
	return rpc.TagMapFromProto(resp), nil
}

func (c *Client) CreateTag(taskID, typeID int64, text, url string) (db.Tag, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.CreateTag(ctx,
		&rpc.TagRequest{TaskId: taskID, TypeId: typeID, Text: text, Url: url})
	if err != nil {
		return db.Tag{}, rpc.StatusToDBError(err)
	}
	return rpc.FromTag(resp.Tag), nil
}

func (c *Client) UpdateTag(id, typeID int64, text, url string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.UpdateTag(ctx,
		&rpc.TagUpdateRequest{Id: id, TypeId: typeID, Text: text, Url: url})
	return rpc.StatusToDBError(err)
}

func (c *Client) DeleteTag(id int64) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.DeleteTag(ctx, &rpc.IDRequest{Id: id})
	return rpc.StatusToDBError(err)
}

// --- Настройки ---

func (c *Client) GetSetting(key string) (string, bool, error) {
	ctx, cancel := callCtx()
	defer cancel()
	resp, err := c.rpc.GetSetting(ctx, &rpc.SettingRequest{Key: key})
	if err != nil {
		return "", false, rpc.StatusToDBError(err)
	}
	return resp.Value, resp.Ok, nil
}

func (c *Client) SetSetting(key, value string) error {
	ctx, cancel := callCtx()
	defer cancel()
	_, err := c.rpc.SetSetting(ctx, &rpc.SettingRequest{Key: key, Value: value})
	return rpc.StatusToDBError(err)
}
