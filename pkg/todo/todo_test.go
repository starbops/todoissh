package todo

import (
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	if store.todos == nil {
		t.Error("store.todos is nil")
	}
	if store.nextID != 1 {
		t.Errorf("store.nextID = %d; want 1", store.nextID)
	}
}

func TestAdd(t *testing.T) {
	store := NewStore()
	text := "Test todo"

	todo, err := store.Add(text)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if todo == nil {
		t.Fatal("Add() returned nil todo")
	}
	if todo.ID != 1 {
		t.Errorf("todo.ID = %d; want 1", todo.ID)
	}
	if todo.Text != text {
		t.Errorf("todo.Text = %q; want %q", todo.Text, text)
	}
	if todo.Completed {
		t.Error("todo.Completed = true; want false")
	}
	if todo.CreatedAt.IsZero() {
		t.Error("todo.CreatedAt is zero")
	}
	if todo.UpdatedAt.IsZero() {
		t.Error("todo.UpdatedAt is zero")
	}

	// Test sequential IDs
	todo2, _ := store.Add("Another todo")
	if todo2.ID != 2 {
		t.Errorf("second todo.ID = %d; want 2", todo2.ID)
	}
}

func TestList(t *testing.T) {
	store := NewStore()
	
	// Test empty list
	todos := store.List()
	if len(todos) != 0 {
		t.Errorf("List() returned %d todos; want 0", len(todos))
	}

	// Add some todos
	store.Add("Todo 1")
	store.Add("Todo 2")
	store.Add("Todo 3")

	todos = store.List()
	if len(todos) != 3 {
		t.Errorf("List() returned %d todos; want 3", len(todos))
	}
}

func TestGet(t *testing.T) {
	store := NewStore()
	
	// Test getting non-existent todo
	todo, err := store.Get(1)
	if err == nil {
		t.Error("Get() non-existent todo; want error")
	}
	if todo != nil {
		t.Error("Get() non-existent todo returned non-nil todo")
	}

	// Add and get a todo
	added, _ := store.Add("Test todo")
	todo, err = store.Get(added.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if todo == nil {
		t.Fatal("Get() returned nil todo")
	}
	if todo.ID != added.ID {
		t.Errorf("todo.ID = %d; want %d", todo.ID, added.ID)
	}
}

func TestUpdate(t *testing.T) {
	store := NewStore()
	
	// Test updating non-existent todo
	_, err := store.Update(1, "Updated text")
	if err == nil {
		t.Error("Update() non-existent todo; want error")
	}

	// Add and update a todo
	todo, _ := store.Add("Original text")
	originalUpdatedAt := todo.UpdatedAt
	time.Sleep(time.Millisecond) // Ensure time difference

	updated, err := store.Update(todo.ID, "Updated text")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Text != "Updated text" {
		t.Errorf("updated.Text = %q; want %q", updated.Text, "Updated text")
	}
	if !updated.UpdatedAt.After(originalUpdatedAt) {
		t.Error("updated.UpdatedAt was not updated")
	}
}

func TestDelete(t *testing.T) {
	store := NewStore()
	
	// Test deleting non-existent todo
	err := store.Delete(1)
	if err == nil {
		t.Error("Delete() non-existent todo; want error")
	}

	// Add and delete a todo
	todo, _ := store.Add("Test todo")
	err = store.Delete(todo.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify todo was deleted
	_, err = store.Get(todo.ID)
	if err == nil {
		t.Error("Get() deleted todo; want error")
	}
}

func TestToggleComplete(t *testing.T) {
	store := NewStore()
	
	// Test toggling non-existent todo
	_, err := store.ToggleComplete(1)
	if err == nil {
		t.Error("ToggleComplete() non-existent todo; want error")
	}

	// Add and toggle a todo
	todo, _ := store.Add("Test todo")
	originalUpdatedAt := todo.UpdatedAt
	time.Sleep(time.Millisecond) // Ensure time difference

	toggled, err := store.ToggleComplete(todo.ID)
	if err != nil {
		t.Fatalf("ToggleComplete() error = %v", err)
	}
	if !toggled.Completed {
		t.Error("toggled.Completed = false; want true")
	}
	if !toggled.UpdatedAt.After(originalUpdatedAt) {
		t.Error("toggled.UpdatedAt was not updated")
	}

	// Toggle back
	toggled, _ = store.ToggleComplete(todo.ID)
	if toggled.Completed {
		t.Error("toggled.Completed = true; want false")
	}
}

func TestConcurrentOperations(t *testing.T) {
	store := NewStore()
	done := make(chan bool)
	
	// Concurrent adds
	for i := 0; i < 10; i++ {
		go func(i int) {
			store.Add("Concurrent todo")
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	todos := store.List()
	if len(todos) != 10 {
		t.Errorf("got %d todos after concurrent adds; want 10", len(todos))
	}

	// Verify IDs are unique
	idMap := make(map[int]bool)
	for _, todo := range todos {
		if idMap[todo.ID] {
			t.Errorf("duplicate ID found: %d", todo.ID)
		}
		idMap[todo.ID] = true
	}
} 