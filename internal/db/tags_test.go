package db

import (
	"errors"
	"testing"
)

func TestTagTypesCRUD(t *testing.T) {
	conn := openTestDB(t)

	types, err := TagTypes(conn)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 2 {
		t.Fatalf("сид по умолчанию: ожидалось 2 типа, получено %d", len(types))
	}
	if types[0].Name != "Jira" || types[0].Kind != "task_id" {
		t.Errorf("первый тип = %q/%q, ожидался Jira/task_id", types[0].Name, types[0].Kind)
	}
	if types[1].Name != "Трекер" {
		t.Errorf("второй тип = %q, ожидался Трекер", types[1].Name)
	}

	nt, err := CreateTagType(conn, "Текст", "text", "#8a8a8a")
	if err != nil {
		t.Fatal(err)
	}
	if nt.Kind != "text" || nt.Color != "#8a8a8a" {
		t.Errorf("CreateTagType = %+v", nt)
	}

	types, _ = TagTypes(conn)
	if len(types) != 3 || types[2].Name != "Текст" {
		t.Fatalf("после создания типов = %d, ожидалось 3", len(types))
	}

	if err := UpdateTagType(conn, nt.ID, "Тег-текст", "text", "#4ec9b0"); err != nil {
		t.Fatal(err)
	}
	types, _ = TagTypes(conn)
	if types[2].Name != "Тег-текст" || types[2].Color != "#4ec9b0" {
		t.Errorf("после UpdateTagType = %+v", types[2])
	}

	if err := DeleteTagType(conn, nt.ID); err != nil {
		t.Fatal(err)
	}
	types, _ = TagTypes(conn)
	if len(types) != 2 {
		t.Errorf("после удаления типов = %d, ожидалось 2", len(types))
	}
}

func TestTagTypeInUse(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	task, err := CreateTask(conn, pid, "t")
	if err != nil {
		t.Fatal(err)
	}
	types, _ := TagTypes(conn)
	if len(types) == 0 {
		t.Fatal("нет типов по умолчанию")
	}
	if _, err := CreateTag(conn, task.ID, types[0].ID, "GW-567", ""); err != nil {
		t.Fatal(err)
	}
	if err := DeleteTagType(conn, types[0].ID); !errors.Is(err, ErrTagTypeInUse) {
		t.Errorf("DeleteTagType используемого типа = %v, ожидался ErrTagTypeInUse", err)
	}
}

func TestTagsCRUD(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	task, err := CreateTask(conn, pid, "t")
	if err != nil {
		t.Fatal(err)
	}
	types, _ := TagTypes(conn)
	jira := types[0].ID

	tag, err := CreateTag(conn, task.ID, jira, "GW-567", "https://jira/GW-567")
	if err != nil {
		t.Fatal(err)
	}
	if tag.TypeName != "Jira" || tag.Color != "#569cd6" {
		t.Errorf("CreateTag денормализованные поля = %+v", tag)
	}

	if _, err := CreateTag(conn, task.ID, jira, "4455", ""); err != nil {
		t.Fatal(err)
	}

	tags, err := TaskTags(conn, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0].Text != "GW-567" || tags[1].Text != "4455" {
		t.Fatalf("TaskTags = %+v", tags)
	}

	if err := UpdateTag(conn, tag.ID, jira, "GW-999", "https://jira/GW-999"); err != nil {
		t.Fatal(err)
	}
	tags, _ = TaskTags(conn, task.ID)
	if tags[0].Text != "GW-999" || tags[0].URL != "https://jira/GW-999" {
		t.Errorf("после UpdateTag = %+v", tags[0])
	}

	if err := DeleteTag(conn, tag.ID); err != nil {
		t.Fatal(err)
	}
	tags, _ = TaskTags(conn, task.ID)
	if len(tags) != 1 || tags[0].Text != "4455" {
		t.Errorf("после DeleteTag = %+v", tags)
	}
}

func TestTagsByProjectAndTasks(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	task1, err := CreateTask(conn, pid, "t1")
	if err != nil {
		t.Fatal(err)
	}
	task2, err := CreateTask(conn, pid, "t2")
	if err != nil {
		t.Fatal(err)
	}
	types, _ := TagTypes(conn)

	if _, err := CreateTag(conn, task1.ID, types[0].ID, "GW-1", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTag(conn, task1.ID, types[1].ID, "4455", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTag(conn, task2.ID, types[0].ID, "GW-2", ""); err != nil {
		t.Fatal(err)
	}

	byProj, err := TagsByProject(conn, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(byProj[task1.ID]) != 2 || len(byProj[task2.ID]) != 1 {
		t.Errorf("TagsByProject = %+v", byProj)
	}
	if byProj[task1.ID][0].Text != "GW-1" || byProj[task1.ID][1].Text != "4455" {
		t.Errorf("порядок тегов задачи = %+v", byProj[task1.ID])
	}

	byTasks, err := TagsByTasks(conn, []int64{task1.ID, task2.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(byTasks[task1.ID]) != 2 || len(byTasks[task2.ID]) != 1 {
		t.Errorf("TagsByTasks = %+v", byTasks)
	}
	empty, err := TagsByTasks(conn, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Errorf("TagsByTasks(nil) = %+v", empty)
	}
}

func TestTagCascadeDelete(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	task, err := CreateTask(conn, pid, "t")
	if err != nil {
		t.Fatal(err)
	}
	types, _ := TagTypes(conn)
	if _, err := CreateTag(conn, task.ID, types[0].ID, "GW-567", ""); err != nil {
		t.Fatal(err)
	}

	if err := DeleteTask(conn, task.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := conn.QueryRow("SELECT COUNT(*) FROM task_tags").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("каскад: в task_tags осталось %d строк", n)
	}
}
