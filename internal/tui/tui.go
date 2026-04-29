package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"todo/internal/todo"
)

// Run starts the full-screen TUI backed by svc. Blocks until the user quits.
func Run(svc *todo.Service) error {
	p := tea.NewProgram(initialModel(svc), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type mode int

const (
	modeList mode = iota
	modeAdd
	modeEdit
	modeConfirmDelete
)

type model struct {
	svc      *todo.Service
	list     list.Model
	input    textinput.Model
	mode     mode
	errMsg   string
	editID   int
	deleteID int
}

func initialModel(svc *todo.Service) model {
	ti := textinput.New()
	ti.CharLimit = 120

	l := list.New(nil, itemDelegate{}, 0, 0)
	l.Title = "Tasks"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)

	return model{svc: svc, list: l, input: ti, mode: modeList}
}

func (m model) Init() tea.Cmd { return nil }

// refreshItems reloads list contents from the service and restores the cursor to preferID.
func refreshItems(svc *todo.Service, l *list.Model, preferID int) error {
	tasks, err := svc.List()
	if err != nil {
		return err
	}
	activePos := 0
	items := make([]list.Item, len(tasks))
	for i, t := range tasks {
		pos := 0
		if !t.Done {
			activePos++
			pos = activePos
		}
		items[i] = taskItem{task: t, position: pos}
	}
	l.SetItems(items)
	for i, item := range l.Items() {
		if item.(taskItem).task.ID == preferID {
			l.Select(i)
			break
		}
	}
	return nil
}

func selectedTaskID(l list.Model) (int, bool) {
	item, ok := l.SelectedItem().(taskItem)
	if !ok {
		return 0, false
	}
	return item.task.ID, true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-4)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.errMsg = ""
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeAdd:
			return m.updateAdd(msg)
		case modeEdit:
			return m.updateEdit(msg)
		case modeConfirmDelete:
			return m.updateConfirmDelete(msg)
		}
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "a":
		m.input.Placeholder = "task description"
		m.input.Prompt = "New task: "
		m.input.SetValue("")
		cmd := m.input.Focus()
		m.mode = modeAdd
		return m, cmd

	case "e":
		id, ok := selectedTaskID(m.list)
		if !ok {
			return m, nil
		}
		selected := m.list.SelectedItem().(taskItem)
		m.input.Placeholder = ""
		m.input.Prompt = "Edit task: "
		m.input.SetValue(selected.task.Text)
		m.input.CursorEnd()
		cmd := m.input.Focus()
		m.editID = id
		m.mode = modeEdit
		return m, cmd

	case "d":
		id, ok := selectedTaskID(m.list)
		if !ok {
			return m, nil
		}
		m.deleteID = id
		m.mode = modeConfirmDelete
		return m, nil

	case " ":
		id, ok := selectedTaskID(m.list)
		if !ok {
			return m, nil
		}
		if _, err := m.svc.ToggleDone(id); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if err := refreshItems(m.svc, &m.list, id); err != nil {
			m.errMsg = err.Error()
		}
		return m, nil

	case "k":
		id, ok := selectedTaskID(m.list)
		if !ok {
			return m, nil
		}
		if err := m.svc.Move(id, -1); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if err := refreshItems(m.svc, &m.list, id); err != nil {
			m.errMsg = err.Error()
		}
		return m, nil

	case "j":
		id, ok := selectedTaskID(m.list)
		if !ok {
			return m, nil
		}
		if err := m.svc.Move(id, +1); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if err := refreshItems(m.svc, &m.list, id); err != nil {
			m.errMsg = err.Error()
		}
		return m, nil
	}

	// Forward remaining keys to the list for arrow-key navigation.
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) updateAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input.Blur()
		m.input.SetValue("")
		m.mode = modeList
		return m, nil
	case "enter":
		task, err := m.svc.Add(m.input.Value())
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.input.Blur()
		m.input.SetValue("")
		m.mode = modeList
		if err := refreshItems(m.svc, &m.list, task.ID); err != nil {
			m.errMsg = err.Error()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input.Blur()
		m.input.SetValue("")
		m.mode = modeList
		return m, nil
	case "enter":
		if _, err := m.svc.Edit(m.editID, m.input.Value()); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if err := refreshItems(m.svc, &m.list, m.editID); err != nil {
			m.errMsg = err.Error()
		}
		m.input.Blur()
		m.input.SetValue("")
		m.mode = modeList
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if err := m.svc.Delete(m.deleteID); err != nil {
			m.errMsg = err.Error()
			m.mode = modeList
			return m, nil
		}
		if err := refreshItems(m.svc, &m.list, m.deleteID); err != nil {
			m.errMsg = err.Error()
		}
		m.mode = modeList
		return m, nil
	case "n", "esc":
		m.mode = modeList
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	var footer string
	switch m.mode {
	case modeList:
		footer = "a:add  e:edit  d:delete  space:done  j/k:move  q:quit"
	case modeAdd, modeEdit:
		footer = "enter: save   esc: cancel"
	case modeConfirmDelete:
		footer = "y: yes   n: no"
	}

	body := m.list.View()
	switch m.mode {
	case modeAdd, modeEdit:
		body += "\n" + m.input.View()
	case modeConfirmDelete:
		body += "\n" + errStyle.Render("Are you sure you want to delete this? (y/n)")
	}
	if m.errMsg != "" {
		body += "\n" + errStyle.Render(m.errMsg)
	}
	return body + "\n" + helpStyle.Render(footer)
}
