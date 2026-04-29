package todo

import "testing"

func TestMemStore_SaveAssignsSequentialIDs(t *testing.T) {
	s := NewMemStore()
	t1, _ := s.Save(Task{Text: "one"})
	t2, _ := s.Save(Task{Text: "two"})
	if t1.ID != 1 || t2.ID != 2 {
		t.Fatalf("expected IDs 1,2 got %d,%d", t1.ID, t2.ID)
	}
}

func TestMemStore_Move_Slide(t *testing.T) {
	s := NewMemStore()
	for _, text := range []string{"a", "b", "c", "d", "e"} {
		s.Save(Task{Text: text})
	}
	// Move b (ID:2) down by 2 → should land at position 3
	s.Move(2, 2)
	tasks, _ := s.All()
	want := []string{"a", "c", "d", "b", "e"}
	for i, w := range want {
		if tasks[i].Text != w {
			t.Errorf("position %d: got %q want %q", i, tasks[i].Text, w)
		}
	}
}

func TestMemStore_Move_BoundaryClamped(t *testing.T) {
	s := NewMemStore()
	s.Save(Task{Text: "x"})
	s.Save(Task{Text: "y"})
	// Move first item further up — should stay at 0
	s.Move(1, -5)
	tasks, _ := s.All()
	if tasks[0].ID != 1 {
		t.Errorf("expected ID:1 at position 0, got ID:%d", tasks[0].ID)
	}
}

func TestMemStore_Update_NotFound(t *testing.T) {
	s := NewMemStore()
	if err := s.Update(Task{ID: 99}); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMemStore_Delete_NotFound(t *testing.T) {
	s := NewMemStore()
	if err := s.Delete(99); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
