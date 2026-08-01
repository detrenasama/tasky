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
