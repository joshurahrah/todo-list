package todo

import "testing"

func TestService_Add_RejectsEmpty(t *testing.T) {
	svc := NewService(NewMemStore())
	if _, err := svc.Add("  "); err != ErrEmptyText {
		t.Errorf("expected ErrEmptyText, got %v", err)
	}
}

func TestService_Edit_RejectsEmpty(t *testing.T) {
	svc := NewService(NewMemStore())
	task, _ := svc.Add("hello")
	if _, err := svc.Edit(task.ID, " "); err != ErrEmptyText {
		t.Errorf("expected ErrEmptyText, got %v", err)
	}
}

func TestService_Edit_NotFound(t *testing.T) {
	svc := NewService(NewMemStore())
	if _, err := svc.Edit(99, "text"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Delete_RemovesTask(t *testing.T) {
	svc := NewService(NewMemStore())
	t1, _ := svc.Add("one")
	t2, _ := svc.Add("two")
	svc.Delete(t1.ID)
	tasks, _ := svc.List()
	if len(tasks) != 1 || tasks[0].ID != t2.ID {
		t.Errorf("expected only t2, got %v", tasks)
	}
}

func TestService_ToggleDone_BecomingDoneMovesToEnd(t *testing.T) {
	svc := NewService(NewMemStore())
	svc.Add("one")
	t2, _ := svc.Add("two")
	svc.Add("three")

	svc.ToggleDone(t2.ID)
	tasks, _ := svc.List()
	last := tasks[len(tasks)-1]
	if last.ID != t2.ID || !last.Done {
		t.Errorf("expected t2 (done) at end, got ID:%d done:%v", last.ID, last.Done)
	}
}

func TestService_ToggleDone_BecomingActiveMovesToEndOfActiveBlock(t *testing.T) {
	svc := NewService(NewMemStore())
	svc.Add("one")
	t2, _ := svc.Add("two")
	svc.Add("three")

	svc.ToggleDone(t2.ID) // → [one, three, ✓two]
	svc.ToggleDone(t2.ID) // → [one, three, two]

	tasks, _ := svc.List()
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}
	for _, task := range tasks {
		if task.Done {
			t.Errorf("expected all tasks active, but ID:%d is done", task.ID)
		}
	}
	if tasks[2].ID != t2.ID {
		t.Errorf("expected t2 at position 2 (end of active block), got ID:%d", tasks[2].ID)
	}
}

func TestService_Move_CrossRegionRejected(t *testing.T) {
	svc := NewService(NewMemStore())
	t1, _ := svc.Add("one")
	t2, _ := svc.Add("two")

	svc.ToggleDone(t2.ID) // → [one, ✓two]
	svc.Move(t1.ID, 1)    // should be no-op (would cross into done region)

	tasks, _ := svc.List()
	if tasks[0].ID != t1.ID {
		t.Errorf("expected t1 at position 0 (unchanged), got ID:%d", tasks[0].ID)
	}
}
