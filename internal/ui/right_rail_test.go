package ui

import (
	"strings"
	"testing"
)

func TestRightRailUpdateNotice(t *testing.T) {
	out := stripANSI(rightRail("v0.5.4", "v0.6.0", rightW, 30, ""))
	if !strings.Contains(out, "Tasky v0.5.4") {
		t.Fatalf("нет текущей версии:\n%s", out)
	}
	if !strings.Contains(out, "↑ доступно v0.6.0 · tasky upgrade") {
		t.Fatalf("нет строки обновления:\n%s", out)
	}
	if strings.Index(out, "Tasky v0.5.4") > strings.Index(out, "↑ доступно") {
		t.Fatalf("строка обновления должна быть под версией:\n%s", out)
	}
}

func TestRightRailWithoutUpdate(t *testing.T) {
	out := stripANSI(rightRail("v0.5.4", "", rightW, 30, ""))
	if !strings.Contains(out, "Tasky v0.5.4") {
		t.Fatalf("нет версии:\n%s", out)
	}
	if strings.Contains(out, "доступно") {
		t.Fatalf("лишняя строка обновления:\n%s", out)
	}
}

func TestUpdateMsgSetsNotice(t *testing.T) {
	m, _ := newChangelogTestModel(t, "v0.5.4", "v0.5.4")
	base := resizeModel(t, m, 120, 40)
	mm, _ := base.Update(updateMsg("v0.6.0"))
	got := mm.(model)
	if got.updateVer != "v0.6.0" {
		t.Fatalf("updateVer = %q", got.updateVer)
	}
	if !strings.Contains(stripANSI(got.View()), "↑ доступно v0.6.0") {
		t.Fatal("строка обновления не попала в View")
	}
}
