package todo

// Tab holds an independent task list with a stable ID and display name.
// ID is the persistent identity; Name is display-only and safe to rename later.
type Tab struct {
	ID   int
	Name string
	// store is kept alongside svc so the workspace can snapshot it for persistence.
	store *MemStore
	svc   *Service
}

// Service returns the task service for this tab.
func (t *Tab) Service() *Service { return t.svc }

// Workspace holds one or more tabs and tracks which is active.
type Workspace struct {
	tabs      []*Tab
	activeIdx int
	nextTabID int
}

// NewWorkspace returns a workspace with a single default "Tasks" tab.
func NewWorkspace() *Workspace {
	ws := &Workspace{nextTabID: 1}
	ws.addDefaultTab()
	return ws
}

func (ws *Workspace) addDefaultTab() {
	store := NewMemStore()
	tab := &Tab{
		ID:    ws.nextTabID,
		Name:  "Tasks",
		store: store,
		svc:   NewService(store),
	}
	ws.nextTabID++
	ws.tabs = append(ws.tabs, tab)
	ws.activeIdx = len(ws.tabs) - 1
}

// Tabs returns all tabs in display order.
func (ws *Workspace) Tabs() []*Tab { return ws.tabs }

// Active returns the currently selected tab.
func (ws *Workspace) Active() *Tab { return ws.tabs[ws.activeIdx] }

// ActiveIndex returns the index of the active tab.
func (ws *Workspace) ActiveIndex() int { return ws.activeIdx }

// AddTab creates a new tab with the given name, appends it, and makes it active.
func (ws *Workspace) AddTab(name string) *Tab {
	store := NewMemStore()
	tab := &Tab{
		ID:    ws.nextTabID,
		Name:  name,
		store: store,
		svc:   NewService(store),
	}
	ws.nextTabID++
	ws.tabs = append(ws.tabs, tab)
	ws.activeIdx = len(ws.tabs) - 1
	return tab
}

// DeleteTab removes the tab with the given ID. If the last tab is deleted a
// fresh default tab is auto-created so the workspace is never empty.
func (ws *Workspace) DeleteTab(id int) error {
	idx := -1
	for i, t := range ws.tabs {
		if t.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotFound
	}
	ws.tabs = append(ws.tabs[:idx], ws.tabs[idx+1:]...)
	if len(ws.tabs) == 0 {
		store := NewMemStore()
		tab := &Tab{
			ID:    ws.nextTabID,
			Name:  "Tasks",
			store: store,
			svc:   NewService(store),
		}
		ws.nextTabID++
		ws.tabs = []*Tab{tab}
		ws.activeIdx = 0
	} else {
		if idx < ws.activeIdx {
			ws.activeIdx--
		} else if ws.activeIdx >= len(ws.tabs) {
			ws.activeIdx = len(ws.tabs) - 1
		}
	}
	return nil
}

// Cycle moves the active tab by delta positions, wrapping at both ends.
func (ws *Workspace) Cycle(delta int) {
	n := len(ws.tabs)
	ws.activeIdx = ((ws.activeIdx + delta) % n + n) % n
}

// toSnapshot produces a serialisable snapshot of the workspace.
func (ws *Workspace) toSnapshot() workspaceSnapshot {
	snap := workspaceSnapshot{
		Version:   1,
		NextTabID: ws.nextTabID,
		ActiveIdx: ws.activeIdx,
		Tabs:      make([]tabSnapshot, len(ws.tabs)),
	}
	for i, tab := range ws.tabs {
		tasks, nextID := tab.store.snapshot()
		tsnap := tabSnapshot{
			ID:     tab.ID,
			Name:   tab.Name,
			NextID: nextID,
			Tasks:  make([]taskSnapshot, len(tasks)),
		}
		for j, t := range tasks {
			tsnap.Tasks[j] = taskSnapshot{ID: t.ID, Text: t.Text, Done: t.Done}
		}
		snap.Tabs[i] = tsnap
	}
	return snap
}

// workspaceFromSnapshot reconstructs a Workspace from a persisted snapshot.
func workspaceFromSnapshot(snap workspaceSnapshot) *Workspace {
	ws := &Workspace{
		nextTabID: snap.NextTabID,
		activeIdx: snap.ActiveIdx,
	}
	ws.tabs = make([]*Tab, len(snap.Tabs))
	for i, ts := range snap.Tabs {
		tasks := make([]Task, len(ts.Tasks))
		for j, tsk := range ts.Tasks {
			tasks[j] = Task{ID: tsk.ID, Text: tsk.Text, Done: tsk.Done}
		}
		store := newMemStoreFromSnapshot(tasks, ts.NextID)
		ws.tabs[i] = &Tab{
			ID:    ts.ID,
			Name:  ts.Name,
			store: store,
			svc:   NewService(store),
		}
	}
	if len(ws.tabs) == 0 {
		ws.addDefaultTab()
	}
	if ws.activeIdx >= len(ws.tabs) {
		ws.activeIdx = 0
	}
	return ws
}
