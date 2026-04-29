package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"todo/internal/todo"
)

// taskItem wraps a todo.Task for display in the list widget.
type taskItem struct {
	task     todo.Task
	position int // 1-based position within the active block; 0 for done items
}

func (t taskItem) Title() string       { return t.task.Text }
func (t taskItem) Description() string { return "" }
func (t taskItem) FilterValue() string { return t.task.Text }

// itemDelegate is a compact list delegate with done-state styling.
type itemDelegate struct{}

func (d itemDelegate) Height() int                              { return 1 }
func (d itemDelegate) Spacing() int                             { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ti := item.(taskItem)
	selected := index == m.Index()

	var line string
	if ti.task.Done {
		text := fmt.Sprintf("✓ %s", ti.task.Text)
		if selected {
			line = selectedDoneStyle.Render(">  " + text)
		} else {
			line = doneStyle.Render("   " + text)
		}
	} else {
		text := fmt.Sprintf("[%d] %s", ti.position, ti.task.Text)
		if selected {
			line = selectedStyle.Render(">  " + text)
		} else {
			line = "   " + text
		}
	}
	fmt.Fprint(w, line)
}
