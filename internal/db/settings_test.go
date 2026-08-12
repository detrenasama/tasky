package db

import (
	"testing"
)

func TestSettingsGetSet(t *testing.T) {
	conn := openTestDB(t)
	defer conn.Close()

	if v, ok, err := GetSetting(conn, "hide_days"); err != nil || ok || v != "" {
		t.Fatalf("несуществующая настройка: v=%q ok=%v err=%v", v, ok, err)
	}

	if err := SetSetting(conn, "hide_days", "14"); err != nil {
		t.Fatal(err)
	}
	if v, ok, err := GetSetting(conn, "hide_days"); err != nil || !ok || v != "14" {
		t.Fatalf("после записи: v=%q ok=%v err=%v", v, ok, err)
	}

	if err := SetSetting(conn, "hide_days", "0"); err != nil {
		t.Fatal(err)
	}
	if v, _, _ := GetSetting(conn, "hide_days"); v != "0" {
		t.Fatalf("перезапись не сработала: %q", v)
	}

	if err := SetSetting(conn, "other", "x"); err != nil {
		t.Fatal(err)
	}
	if v, _, _ := GetSetting(conn, "hide_days"); v != "0" {
		t.Fatalf("другая настройка затерла hide_days: %q", v)
	}
}
