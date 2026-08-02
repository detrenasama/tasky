package db

import (
	"database/sql"
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
func ProjectLinks(conn *sql.DB, projectID int64) ([]ProjectLink, error) {
	rows, err := conn.Query(`
SELECT id, project_id, name, url, created_at FROM project_links
WHERE project_id = ?
ORDER BY created_at, id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProjectLink
	for rows.Next() {
		var l ProjectLink
		var created int64
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.URL, &created); err != nil {
			return nil, err
		}
		l.CreatedAt = time.Unix(created, 0)
		out = append(out, l)
	}
	return out, rows.Err()
}

func CreateProjectLink(conn *sql.DB, projectID int64, name, url string) (ProjectLink, error) {
	var l ProjectLink
	now := time.Now().Unix()
	res, err := conn.Exec(
		"INSERT INTO project_links (project_id, name, url, created_at) VALUES (?, ?, ?, ?)",
		projectID, name, url, now)
	if err != nil {
		return l, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return l, err
	}
	l = ProjectLink{ID: id, ProjectID: projectID, Name: name, URL: url, CreatedAt: time.Unix(now, 0)}
	return l, nil
}

func DeleteProjectLink(conn *sql.DB, id int64) error {
	_, err := conn.Exec("DELETE FROM project_links WHERE id = ?", id)
	return err
}
