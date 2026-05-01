package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"todo/internal/todo"
)

// Run starts the full-screen TUI backed by ws. Blocks until the user quits.
func Run(ws *todo.Workspace, savePath string) error {
	p := tea.NewProgram(initialModel(ws, savePath), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type mode int

const (
	modeList mode = iota
	modeAdd
	modeEdit
	modeConfirmDelete
	modeConfirmDeleteCompleted
	modeAddTab
	modeConfirmDeleteTab
)

type tabUI struct {
	tab  *todo.Tab
	list list.Model
}

type model struct {
	workspace      *todo.Workspace
	tabs           []tabUI
	input          textinput.Model
	mode           mode
	errMsg         string
	editID         int
	deleteID       int
	completedCount int
	deleteTabID    int
	savePath       string
	width          int
	height         int
}

func buildList() list.Model {
	l := list.New(nil, itemDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	return l
}

func initialModel(ws *todo.Workspace, savePath string) model {
	ti := textinput.New()
	ti.CharLimit = 120

	wsTabs := ws.Tabs()
	tabs := make([]tabUI, len(wsTabs))
	for i, t := range wsTabs {
		l := buildList()
		_ = refreshItems(t.Service(), &l, 0)
		tabs[i] = tabUI{tab: t, list: l}
	}

	return model{
		workspace: ws,
		tabs:      tabs,
		input:     ti,
		savePath:  savePath,
	}
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

func (m model) activeSvc() *todo.Service {
	return m.workspace.Active().Service()
}

func (m model) activeList() *list.Model {
	return &m.tabs[m.workspace.ActiveIndex()].list
}

// syncTabs reconciles m.tabs with the current workspace tabs.
// Existing tabs keep their list state (cursor position etc); new tabs get a fresh list.
func (m model) syncTabs() (model, error) {
	wsTabs := m.workspace.Tabs()
	existing := make(map[int]tabUI, len(m.tabs))
	for _, t := range m.tabs {
		existing[t.tab.ID] = t
	}
	newTabs := make([]tabUI, len(wsTabs))
	for i, wsTab := range wsTabs {
		if old, ok := existing[wsTab.ID]; ok {
			old.tab = wsTab
			newTabs[i] = old
		} else {
			l := buildList()
			l.SetSize(m.width, m.listHeight())
			if err := refreshItems(wsTab.Service(), &l, 0); err != nil {
				return m, err
			}
			newTabs[i] = tabUI{tab: wsTab, list: l}
		}
	}
	m.tabs = newTabs
	return m, nil
}

func (m model) listHeight() int {
	if m.height == 0 {
		return 0
	}
	// tab bar (1) + "\n" (1) + "\n" before footer (1) + footer (1) + padding (2)
	return m.height - 6
}

// save persists the workspace and returns the (possibly error-annotated) model.
func (m model) save() model {
	if m.savePath == "" {
		return m
	}
	if err := todo.Save(m.savePath, m.workspace); err != nil {
		m.errMsg = "save failed: " + err.Error()
	}
	return m
}

func (m model) deleteTabName() string {
	for _, t := range m.workspace.Tabs() {
		if t.ID == m.deleteTabID {
			return t.Name
		}
	}
	return "tab"
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		lh := m.listHeight()
		for i := range m.tabs {
			m.tabs[i].list.SetSize(msg.Width, lh)
		}
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
		case modeConfirmDeleteCompleted:
			return m.updateConfirmDeleteCompleted(msg)
		case modeAddTab:
			return m.updateAddTab(msg)
		case modeConfirmDeleteTab:
			return m.updateConfirmDeleteTab(msg)
		}
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "tab":
		m.workspace.Cycle(+1)
		m = m.save()
		return m, nil

	case "shift+tab":
		m.workspace.Cycle(-1)
		m = m.save()
		return m, nil

	case "t":
		m.input.Placeholder = "tab name"
		m.input.Prompt = "New tab: "
		m.input.SetValue("")
		cmd := m.input.Focus()
		m.mode = modeAddTab
		return m, cmd

	case "X":
		m.deleteTabID = m.workspace.Active().ID
		m.mode = modeConfirmDeleteTab
		return m, nil

	case "a":
		m.input.Placeholder = "task description"
		m.input.Prompt = "New task: "
		m.input.SetValue("")
		cmd := m.input.Focus()
		m.mode = modeAdd
		return m, cmd

	case "e":
		id, ok := selectedTaskID(*m.activeList())
		if !ok {
			return m, nil
		}
		selected := m.activeList().SelectedItem().(taskItem)
		m.input.Placeholder = ""
		m.input.Prompt = "Edit task: "
		m.input.SetValue(selected.task.Text)
		m.input.CursorEnd()
		cmd := m.input.Focus()
		m.editID = id
		m.mode = modeEdit
		return m, cmd

	case "d":
		id, ok := selectedTaskID(*m.activeList())
		if !ok {
			return m, nil
		}
		m.deleteID = id
		m.mode = modeConfirmDelete
		return m, nil

	case "D":
		tasks, err := m.activeSvc().List()
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		count := 0
		for _, t := range tasks {
			if t.Done {
				count++
			}
		}
		if count == 0 {
			return m, nil
		}
		m.completedCount = count
		m.mode = modeConfirmDeleteCompleted
		return m, nil

	case " ":
		id, ok := selectedTaskID(*m.activeList())
		if !ok {
			return m, nil
		}
		if _, err := m.activeSvc().ToggleDone(id); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if err := refreshItems(m.activeSvc(), m.activeList(), id); err != nil {
			m.errMsg = err.Error()
		}
		m = m.save()
		return m, nil

	case "k":
		id, ok := selectedTaskID(*m.activeList())
		if !ok {
			return m, nil
		}
		if err := m.activeSvc().Move(id, -1); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if err := refreshItems(m.activeSvc(), m.activeList(), id); err != nil {
			m.errMsg = err.Error()
		}
		m = m.save()
		return m, nil

	case "j":
		id, ok := selectedTaskID(*m.activeList())
		if !ok {
			return m, nil
		}
		if err := m.activeSvc().Move(id, +1); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if err := refreshItems(m.activeSvc(), m.activeList(), id); err != nil {
			m.errMsg = err.Error()
		}
		m = m.save()
		return m, nil
	}

	// Forward remaining keys to the active list for arrow-key navigation.
	var cmd tea.Cmd
	m.tabs[m.workspace.ActiveIndex()].list, cmd = m.tabs[m.workspace.ActiveIndex()].list.Update(msg)
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
		task, err := m.activeSvc().Add(m.input.Value())
		if err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.input.Blur()
		m.input.SetValue("")
		m.mode = modeList
		if err := refreshItems(m.activeSvc(), m.activeList(), task.ID); err != nil {
			m.errMsg = err.Error()
		}
		m = m.save()
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
		if _, err := m.activeSvc().Edit(m.editID, m.input.Value()); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if err := refreshItems(m.activeSvc(), m.activeList(), m.editID); err != nil {
			m.errMsg = err.Error()
		}
		m.input.Blur()
		m.input.SetValue("")
		m.mode = modeList
		m = m.save()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if err := m.activeSvc().Delete(m.deleteID); err != nil {
			m.errMsg = err.Error()
			m.mode = modeList
			return m, nil
		}
		if err := refreshItems(m.activeSvc(), m.activeList(), m.deleteID); err != nil {
			m.errMsg = err.Error()
		}
		m.mode = modeList
		m = m.save()
		return m, nil
	case "n", "esc":
		m.mode = modeList
		return m, nil
	}
	return m, nil
}

func (m model) updateConfirmDeleteCompleted(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if _, err := m.activeSvc().DeleteCompleted(); err != nil {
			m.errMsg = err.Error()
			m.mode = modeList
			return m, nil
		}
		if err := refreshItems(m.activeSvc(), m.activeList(), 0); err != nil {
			m.errMsg = err.Error()
		}
		m.mode = modeList
		m = m.save()
		return m, nil
	case "n", "esc":
		m.mode = modeList
		return m, nil
	}
	return m, nil
}

func (m model) updateAddTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input.Blur()
		m.input.SetValue("")
		m.mode = modeList
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.input.Value())
		if name == "" {
			name = "Tasks"
		}
		m.workspace.AddTab(name)
		var err error
		m, err = m.syncTabs()
		if err != nil {
			m.errMsg = err.Error()
		}
		m.input.Blur()
		m.input.SetValue("")
		m.mode = modeList
		m = m.save()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateConfirmDeleteTab(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if err := m.workspace.DeleteTab(m.deleteTabID); err != nil {
			m.errMsg = err.Error()
			m.mode = modeList
			return m, nil
		}
		var err error
		m, err = m.syncTabs()
		if err != nil {
			m.errMsg = err.Error()
		}
		m.mode = modeList
		m = m.save()
		return m, nil
	case "n", "esc":
		m.mode = modeList
		return m, nil
	}
	return m, nil
}

func (m model) renderTabBar() string {
	tabs := m.workspace.Tabs()
	activeIdx := m.workspace.ActiveIndex()
	parts := make([]string, len(tabs))
	for i, t := range tabs {
		if i == activeIdx {
			parts[i] = activeTabStyle.Render(" " + t.Name + " ")
		} else {
			parts[i] = inactiveTabStyle.Render(" " + t.Name + " ")
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m model) View() string {
	var footer string
	switch m.mode {
	case modeList:
		footer = "a:add  e:edit  d:del  D:del done  spc:done  j/k:move  t:new tab  X:del tab  tab/S-tab:cycle  q:quit"
	case modeAdd, modeEdit:
		footer = "enter:save  esc:cancel"
	case modeAddTab:
		footer = "enter:create tab  esc:cancel"
	case modeConfirmDelete, modeConfirmDeleteCompleted, modeConfirmDeleteTab:
		footer = "y:yes  n:no"
	}

	tabBar := m.renderTabBar()
	body := m.tabs[m.workspace.ActiveIndex()].list.View()

	switch m.mode {
	case modeAdd, modeEdit, modeAddTab:
		body += "\n" + m.input.View()
	case modeConfirmDelete:
		body += "\n" + errStyle.Render("Delete this task? (y/n)")
	case modeConfirmDeleteCompleted:
		body += "\n" + errStyle.Render(fmt.Sprintf("Delete all %d completed tasks? (y/n)", m.completedCount))
	case modeConfirmDeleteTab:
		body += "\n" + errStyle.Render(fmt.Sprintf("Delete tab '%s' and all its tasks? (y/n)", m.deleteTabName()))
	}

	if m.errMsg != "" {
		body += "\n" + errStyle.Render(m.errMsg)
	}

	return tabBar + "\n" + body + "\n" + helpStyle.Render(footer)
}
