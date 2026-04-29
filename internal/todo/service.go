package todo

import "strings"

// Service applies business rules over a Store.
type Service struct {
	store Store
}

// NewService returns a Service backed by the given Store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Add creates a new active task. Returns ErrEmptyText if text is blank.
func (s *Service) Add(text string) (Task, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Task{}, ErrEmptyText
	}
	return s.store.Save(Task{Text: text})
}

// List returns all tasks in display order: active first, then completed.
func (s *Service) List() ([]Task, error) {
	return s.store.All()
}

// Edit updates the text of the task with the given id.
// Returns ErrEmptyText if text is blank, ErrNotFound if the id is missing.
func (s *Service) Edit(id int, text string) (Task, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Task{}, ErrEmptyText
	}
	tasks, err := s.store.All()
	if err != nil {
		return Task{}, err
	}
	for _, t := range tasks {
		if t.ID == id {
			t.Text = text
			if err := s.store.Update(t); err != nil {
				return Task{}, err
			}
			return t, nil
		}
	}
	return Task{}, ErrNotFound
}

// Delete removes the task with the given id.
func (s *Service) Delete(id int) error {
	return s.store.Delete(id)
}

// Move slides the task by delta positions within its active/done region.
// Cross-region moves and boundary moves are silently rejected.
func (s *Service) Move(id int, delta int) error {
	tasks, err := s.store.All()
	if err != nil {
		return err
	}
	idx := -1
	for i, t := range tasks {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotFound
	}
	targetIdx := idx + delta
	if targetIdx < 0 || targetIdx >= len(tasks) {
		return nil // at boundary, silent no-op
	}
	// Reject moves that cross the active/done boundary.
	if tasks[idx].Done != tasks[targetIdx].Done {
		return nil
	}
	return s.store.Move(id, delta)
}

// ToggleDone flips the Done state of the task and repositions it to maintain
// the active-then-done storage invariant:
//   - becoming done → moves to the end of the list
//   - becoming active → moves to the end of the active block
func (s *Service) ToggleDone(id int) (Task, error) {
	tasks, err := s.store.All()
	if err != nil {
		return Task{}, err
	}
	idx := -1
	for i, t := range tasks {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return Task{}, ErrNotFound
	}
	found := tasks[idx]
	found.Done = !found.Done
	if err := s.store.Update(found); err != nil {
		return Task{}, err
	}
	// Reposition to maintain the active-then-done invariant.
	// tasks is the pre-update snapshot; tasks[idx] still has the original Done value.
	var target int
	if found.Done {
		target = len(tasks) - 1
	} else {
		// End of active block: one past the last non-self active task.
		target = 0
		for i, t := range tasks {
			if i != idx && !t.Done {
				target = i + 1
			}
		}
	}
	if target != idx {
		if err := s.store.Move(id, target-idx); err != nil {
			return Task{}, err
		}
	}
	return found, nil
}
