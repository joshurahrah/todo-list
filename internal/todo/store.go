package todo

// Store is the persistence boundary. Implementations can be swapped without changing Service.
// All() must return tasks in display order: active tasks first, then completed.
type Store interface {
	// Save assigns an ID to task, persists it, and returns the stored task.
	Save(task Task) (Task, error)
	// All returns all tasks in display order.
	All() ([]Task, error)
	// Update replaces the stored task matching task.ID. Returns ErrNotFound if missing.
	Update(task Task) error
	// Delete removes the task with the given ID. Returns ErrNotFound if missing.
	Delete(id int) error
	// Move slides the task with the given ID by delta positions (positive = down).
	// Clamped at list boundaries. Uses insert semantics, not swap.
	Move(id int, delta int) error
	// DeleteCompleted removes all completed tasks. Returns the count removed.
	DeleteCompleted() (int, error)
}
