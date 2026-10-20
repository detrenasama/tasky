package ui

import (
	"github.com/detrenasama/tasky/internal/db"
)

// moveSteps возвращает число шагов перемещения в направлении dir, чтобы
// встать на следующего видимого соседа: подряд идущие скрытые соседи
// пропускаются. ok=false — видимого соседа в этом направлении нет.
func moveSteps(idx, dir, n int, hidden func(i int) bool) (int, bool) {
	steps := 1
	for j := idx + dir; j >= 0 && j < n; j += dir {
		if !hidden(j) {
			return steps, true
		}
		steps++
	}
	return 0, false
}

// moveSelected перемещает выбранную задачу/подзадачу на одну видимую позицию
// (dir = -1 вверх, +1 вниз), пропуская скрытые соседи, и сохраняет выделение.
// При активном поиске перемещение недоступно: список отфильтрован, и видимую
// позицию не определить. Один шаг — один вызов Store, поэтому скрытые соседи
// физически сдвигаются на другую сторону.
func (s *tasksScreen) moveSelected(dir int) {
	if s.searchQuery != "" {
		return
	}
	kind, id := s.selectedKindID()
	if id == 0 {
		return
	}
	var (
		move  func(dir int) error
		steps int
		ok    bool
	)
	switch kind {
	case kindTask:
		idx := -1
		for i, t := range s.tasks {
			if t.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		steps, ok = moveSteps(idx, dir, len(s.tasks), func(i int) bool {
			return s.hiddenDue(s.tasks[i])
		})
		move = func(dir int) error { return s.store.MoveTask(id, dir) }
	case kindSubtask:
		parentID := s.selectedTaskID()
		var siblings []db.SubtaskWithTime
		for _, st := range s.subs {
			if st.TaskID == parentID {
				siblings = append(siblings, st)
			}
		}
		idx := -1
		for i, st := range siblings {
			if st.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return
		}
		steps, ok = moveSteps(idx, dir, len(siblings), func(i int) bool {
			return s.hiddenSubtaskDue(siblings[i])
		})
		move = func(dir int) error { return s.store.MoveSubtask(id, dir) }
	}
	if !ok {
		return
	}
	s.lastErr = nil
	for k := 0; k < steps; k++ {
		if err := move(dir); err != nil {
			s.lastErr = err
			break
		}
	}
	s.loadData()
	s.selectByKindID(kind, id)
	s.loadInfo()
	s.loadDesc()
}
