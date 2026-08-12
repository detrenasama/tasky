package db

import "database/sql"

// GetSetting возвращает значение настройки по ключу (ok=false, если
// настройка не задана).
func GetSetting(conn *sql.DB, key string) (string, bool, error) {
	var v string
	err := conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// SetSetting сохраняет или заменяет значение настройки.
func SetSetting(conn *sql.DB, key, value string) error {
	_, err := conn.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) "+
			"ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value)
	return err
}
