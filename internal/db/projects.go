package db

import (
	"database/sql"
	"fmt"
	"time"
)

func Projects(conn *sql.DB) ([]Project, error) {
	rows, err := conn.Query(
		"SELECT id, name, description, created_at FROM projects ORDER BY created_at, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		var created int64
		if err := rows.Scan(&p.ID, &p.Name, &p.Desc, &created); err != nil {
			return nil, err
		}
		p.CreatedAt = time.Unix(created, 0)
		out = append(out, p)
	}
	return out, rows.Err()
}

func CreateProject(conn *sql.DB, name string) (Project, error) {
	var p Project
	now := time.Now().Unix()
	res, err := conn.Exec("INSERT INTO projects (name, created_at) VALUES (?, ?)", name, now)
	if err != nil {
		return p, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return p, err
	}
	p = Project{ID: id, Name: name, CreatedAt: time.Unix(now, 0)}
	return p, nil
}

func DeleteProject(conn *sql.DB, id int64) error {
	_, err := conn.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

// ProjectDescription возвращает описание проекта.
func ProjectDescription(conn *sql.DB, id int64) (string, error) {
	var desc string
	err := conn.QueryRow("SELECT description FROM projects WHERE id = ?", id).Scan(&desc)
	return desc, err
}

func UpdateProjectDescription(conn *sql.DB, id int64, text string) error {
	_, err := conn.Exec("UPDATE projects SET description = ? WHERE id = ?", text, id)
	return err
}

// ProjectLinks возвращает ссылки проекта по порядку добавления.
func ProjectLinks(conn *sql.DB, projectID int64) ([]Link, error) {
	return linksFor(conn, "project_links", "project_id", projectID)
}

func CreateProjectLink(conn *sql.DB, projectID int64, name, url string) (Link, error) {
	return createLink(conn, "project_links", "project_id", projectID, name, url)
}

func DeleteProjectLink(conn *sql.DB, id int64) error {
	return deleteLink(conn, "project_links", id)
}

// ProjectLinksTexts возвращает карту id проекта → объединённые названия и
// адреса его ссылок (для полнотекстового поиска по проектам).
func ProjectLinksTexts(conn *sql.DB) (map[int64]string, error) {
	rows, err := conn.Query(`
SELECT project_id, GROUP_CONCAT(name || ' ' || url, '\n')
FROM project_links
GROUP BY project_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			return nil, err
		}
		out[id] = text
	}
	return out, rows.Err()
}

// linksFor возвращает ссылки владельца (проекта/задачи/подзадачи) по порядку
// добавления.
func linksFor(conn *sql.DB, table, ownerCol string, ownerID int64) ([]Link, error) {
	rows, err := conn.Query(fmt.Sprintf(
		"SELECT id, %s, name, url, created_at FROM %s WHERE %s = ? ORDER BY created_at, id",
		ownerCol, table, ownerCol), ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Link
	for rows.Next() {
		var l Link
		var created int64
		if err := rows.Scan(&l.ID, &l.OwnerID, &l.Name, &l.URL, &created); err != nil {
			return nil, err
		}
		l.CreatedAt = time.Unix(created, 0)
		out = append(out, l)
	}
	return out, rows.Err()
}

func createLink(conn *sql.DB, table, ownerCol string, ownerID int64, name, url string) (Link, error) {
	var l Link
	now := time.Now().Unix()
	res, err := conn.Exec(fmt.Sprintf(
		"INSERT INTO %s (%s, name, url, created_at) VALUES (?, ?, ?, ?)",
		table, ownerCol), ownerID, name, url, now)
	if err != nil {
		return l, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return l, err
	}
	l = Link{ID: id, OwnerID: ownerID, Name: name, URL: url, CreatedAt: time.Unix(now, 0)}
	return l, nil
}

func deleteLink(conn *sql.DB, table string, id int64) error {
	_, err := conn.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", table), id)
	return err
}
