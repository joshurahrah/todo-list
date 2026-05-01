package todo

import "sync"

// MemStore is an in-memory Store implementation. Safe for concurrent use.
type MemStore struct {
	mu     sync.Mutex
	tasks  []Task
	nextID int
}

// NewMemStore returns an empty MemStore with IDs starting at 1.
func NewMemStore() *MemStore {
	return &MemStore{nextID: 1}
}

// Save assigns an ID to task, appends it, and returns the stored task.
func (m *MemStore) Save(task Task) (Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task.ID = m.nextID
	m.nextID++
	m.tasks = append(m.tasks, task)
	return task, nil
}

// All returns a copy of all tasks in current storage order.
func (m *MemStore) All() ([]Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Task, len(m.tasks))
	copy(result, m.tasks)
	return result, nil
}

// Update replaces the stored task with matching ID. Returns ErrNotFound if missing.
func (m *MemStore) Update(task Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.tasks {
		if t.ID == task.ID {
			m.tasks[i] = task
			return nil
		}
	}
	return ErrNotFound
}

// Delete removes the task with the given ID. Returns ErrNotFound if missing.
func (m *MemStore) Delete(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.tasks {
		if t.ID == id {
			m.tasks = append(m.tasks[:i], m.tasks[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

// DeleteCompleted removes all completed tasks and returns the count removed.
func (m *MemStore) DeleteCompleted() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []Task
	count := 0
	for _, t := range m.tasks {
		if t.Done {
			count++
		} else {
			kept = append(kept, t)
		}
	}
	m.tasks = kept
	return count, nil
}

// snapshot returns a copy of tasks and the current nextID for serialization.
func (m *MemStore) snapshot() ([]Task, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tasks := make([]Task, len(m.tasks))
	copy(tasks, m.tasks)
	return tasks, m.nextID
}

// newMemStoreFromSnapshot constructs a MemStore pre-populated with the given tasks and nextID.
func newMemStoreFromSnapshot(tasks []Task, nextID int) *MemStore {
	s := &MemStore{nextID: nextID}
	s.tasks = make([]Task, len(tasks))
	copy(s.tasks, tasks)
	return s
}

// Move slides the task with the given ID by delta positions using insert semantics.
// The target index is clamped to [0, len-1]. Positive delta moves down, negative up.
func (m *MemStore) Move(id int, delta int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := -1
	for i, t := range m.tasks {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotFound
	}
	target := idx + delta
	if target < 0 {
		target = 0
	}
	if target >= len(m.tasks) {
		target = len(m.tasks) - 1
	}
	if target == idx {
		return nil
	}
	task := m.tasks[idx]
	m.tasks = append(m.tasks[:idx], m.tasks[idx+1:]...)
	m.tasks = append(m.tasks[:target], append([]Task{task}, m.tasks[target:]...)...)
	return nil
}
