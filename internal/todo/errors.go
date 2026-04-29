package todo

import "errors"

// ErrNotFound is returned when a task ID does not exist in the store.
var ErrNotFound = errors.New("task not found")

// ErrEmptyText is returned when a task's text is empty or whitespace-only.
var ErrEmptyText = errors.New("task text cannot be empty")
