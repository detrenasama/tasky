package db

import (
	"strings"
	"testing"
	"time"
)

// TestProjectLinksTexts — карта id проекта → названия и адреса ссылок.
func TestProjectLinksTexts(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	now := time.Now().Unix()
	exec(t, conn, "INSERT INTO projects (name, created_at) VALUES ('other', ?)", now)
	var other int64
	if err := conn.QueryRow("SELECT id FROM projects WHERE name = 'other'").Scan(&other); err != nil {
		t.Fatal(err)
	}

	if _, err := CreateProjectLink(conn, pid, "Доки", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProjectLink(conn, pid, "", "https://example.org"); err != nil {
		t.Fatal(err)
	}

	texts, err := ProjectLinksTexts(conn)
	if err != nil {
		t.Fatal(err)
	}
	if len(texts) != 1 {
		t.Fatalf("проектов со ссылками: %d, ожидался 1", len(texts))
	}
	joined := texts[pid]
	if !strings.Contains(joined, "Доки") || !strings.Contains(joined, "https://example.com") ||
		!strings.Contains(joined, "https://example.org") {
		t.Errorf("текст ссылок = %q", joined)
	}
	if _, ok := texts[other]; ok {
		t.Error("у проекта other не должно быть ссылок")
	}
}

func TestProjectDescription(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)

	desc, err := ProjectDescription(conn, pid)
	if err != nil {
		t.Fatalf("ProjectDescription: %v", err)
	}
	if desc != "" {
		t.Errorf("описание по умолчанию = %q, ожидалось пустое", desc)
	}

	text := "Команда:\nИван — @ivan\nСайт: https://example.com"
	if err := UpdateProjectDescription(conn, pid, text); err != nil {
		t.Fatalf("UpdateProjectDescription: %v", err)
	}
	desc, err = ProjectDescription(conn, pid)
	if err != nil {
		t.Fatalf("ProjectDescription: %v", err)
	}
	if desc != text {
		t.Errorf("описание = %q, ожидалось %q", desc, text)
	}
}

func TestProjectLinks(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)

	links, err := ProjectLinks(conn, pid)
	if err != nil {
		t.Fatalf("ProjectLinks: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("ссылок: %d, ожидалось 0", len(links))
	}

	l1, err := CreateProjectLink(conn, pid, "Доки", "https://example.com")
	if err != nil {
		t.Fatalf("CreateProjectLink: %v", err)
	}
	l2, err := CreateProjectLink(conn, pid, "", "https://example.org")
	if err != nil {
		t.Fatalf("CreateProjectLink: %v", err)
	}
	if l1.URL != "https://example.com" || l1.Name != "Доки" || l1.OwnerID != pid || l1.ID == 0 {
		t.Errorf("ссылка l1 = %+v", l1)
	}

	links, err = ProjectLinks(conn, pid)
	if err != nil {
		t.Fatalf("ProjectLinks: %v", err)
	}
	if len(links) != 2 || links[0].URL != "https://example.com" || links[0].Name != "Доки" ||
		links[1].URL != "https://example.org" || links[1].Name != "" {
		t.Errorf("ссылки = %+v, ожидались 2 по порядку добавления", links)
	}

	if err := DeleteProjectLink(conn, l1.ID); err != nil {
		t.Fatalf("DeleteProjectLink: %v", err)
	}
	links, err = ProjectLinks(conn, pid)
	if err != nil {
		t.Fatalf("ProjectLinks: %v", err)
	}
	if len(links) != 1 || links[0].ID != l2.ID {
		t.Errorf("после удаления ссылки = %+v, ожидалась только %d", links, l2.ID)
	}
}

func TestProjectLinkCascadeDelete(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	if _, err := CreateProjectLink(conn, pid, "", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteProject(conn, pid); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := conn.QueryRow("SELECT COUNT(*) FROM project_links").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("после удаления проекта осталось %d ссылок", n)
	}
}
