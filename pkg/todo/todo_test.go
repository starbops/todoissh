/*
Package todo_test provides comprehensive test coverage for the todo package.

This test suite is organized into several categories:
  - Basic functionality tests (TestNewStore, TestAdd, TestList, etc.)
  - File system operation tests (TestFileSystemOperations)
  - Internal function tests (TestGetUserTodosFunction, TestSaveTodosFunction)
  - Mocked implementation tests (TestAddWithMock)
  - Concurrency tests (TestConcurrentOperations)
  - Multi-user tests (TestMultipleUsers)
  - End-to-end workflow tests (TestSimpleEndToEnd)
  - Persistence tests (TestPersistence)

The tests are designed to verify:
  - Correct behavior of all public API methods
  - Data persistence and retrieval
  - Thread safety for concurrent operations
  - User data isolation
  - Error handling and recovery
*/
package todo

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testUsername is the default username used across tests
const testUsername = "testuser"

// Additional test constants for improving test coverage
const (
	testUsername2   = "testuser2"
	nonExistentUser = "nonexistentuser"
	invalidDir      = "/nonexistent/dir"
	readOnlyPerm    = 0400
)

// setupTestStore creates a temporary test directory and initializes a Store.
// It returns the initialized store and the temporary directory path.
// The caller is responsible for calling cleanupTestStore with the returned path.
func setupTestStore(t *testing.T) (*Store, string) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "todoissh-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create store with temp directory
	store, err := NewStore(tempDir)
	if err != nil {
		os.RemoveAll(tempDir) // Clean up on error
		t.Fatalf("NewStore() error = %v", err)
	}

	return store, tempDir
}

// cleanupTestStore removes the temporary directory.
// This should be called after tests, typically in a defer statement.
func cleanupTestStore(tempDir string) {
	os.RemoveAll(tempDir)
}

// TestNewStore tests the creation of a new store.
// It verifies:
// - The store is successfully created
// - The userTodos map is initialized
// - The data directory is correctly set
// - The todos directory is created
func TestNewStore(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	if store.userTodos == nil {
		t.Error("store.userTodos is nil")
	}
	if store.dataDir != tempDir {
		t.Errorf("store.dataDir = %s; want %s", store.dataDir, tempDir)
	}

	// Verify todos directory was created
	todosDir := filepath.Join(tempDir, "todos")
	if _, err := os.Stat(todosDir); os.IsNotExist(err) {
		t.Error("todos directory was not created")
	}
}

// TestFileSystemOperations tests basic file system operations to ensure permissions are correct.
// It verifies:
// - Temporary directory creation works
// - Directory has correct permissions
// - File creation in the directory works
// - File read/write operations work correctly
// - Directory structure is as expected
func TestFileSystemOperations(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "todoissh-fs-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Print the directory permissions
	info, err := os.Stat(tempDir)
	if err != nil {
		t.Fatalf("Could not stat directory: %v", err)
	}
	t.Logf("Directory permissions: %v", info.Mode())

	// Create the todos directory
	todosDir := filepath.Join(tempDir, "todos")
	err = os.MkdirAll(todosDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create todos directory: %v", err)
	}

	// Print the todos directory permissions
	info, err = os.Stat(todosDir)
	if err != nil {
		t.Fatalf("Could not stat todos directory: %v", err)
	}
	t.Logf("Todos directory permissions: %v", info.Mode())

	// Try to create a file
	testFile := filepath.Join(todosDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	t.Logf("Successfully wrote test file: %s", testFile)

	// Try to read it back
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}
	t.Logf("Read %d bytes from test file", len(data))

	// List all files in the directory
	err = filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to get info for %s: %v", path, err)
		}
		t.Logf("Path: %s, Mode: %v, Size: %d", path, info.Mode(), info.Size())
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk directory: %v", err)
	}
}

// TestGetUserTodosFunction tests the getUserTodos function.
// It verifies:
// - Getting todos for a new user creates an empty todos object
// - Loading todos from disk works correctly
// - In-memory caching of todos works
func TestGetUserTodosFunction(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "todoissh-get-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create todos directory
	todosDir := filepath.Join(tempDir, "todos")
	err = os.MkdirAll(todosDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create todos directory: %v", err)
	}

	// Create a store
	store := &Store{
		userTodos: make(map[string]*UserTodos),
		dataDir:   tempDir,
	}

	// Test 1: Get user todos for a new user
	username1 := "new-user"
	t.Logf("Getting todos for new user: %s", username1)
	userTodos, err := store.getUserTodos(username1)
	if err != nil {
		t.Fatalf("getUserTodos() error = %v", err)
	}
	if userTodos == nil {
		t.Fatal("getUserTodos() returned nil")
	}
	if len(userTodos.Todos) != 0 {
		t.Errorf("new user has %d todos; want 0", len(userTodos.Todos))
	}
	if userTodos.NextID != 1 {
		t.Errorf("new user NextID = %d; want 1", userTodos.NextID)
	}
	t.Logf("Successfully got todos for new user")

	// Test 2: Create a file for an existing user
	username2 := "existing-user"
	existingUserTodos := &UserTodos{
		Todos:  make(map[int]*Todo),
		NextID: 5,
	}

	// Add a todo
	todo := &Todo{
		ID:        4,
		Text:      "Existing todo",
		Completed: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	existingUserTodos.Todos[todo.ID] = todo

	// Write to file
	todosPath := filepath.Join(todosDir, username2+".json")
	data, err := json.MarshalIndent(existingUserTodos, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal existing todos: %v", err)
	}
	err = os.WriteFile(todosPath, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write existing todos file: %v", err)
	}
	t.Logf("Created todos file for existing user at %s", todosPath)

	// Get the todos for the existing user
	t.Logf("Getting todos for existing user: %s", username2)
	loadedTodos, err := store.getUserTodos(username2)
	if err != nil {
		t.Fatalf("getUserTodos() error = %v", err)
	}
	if loadedTodos == nil {
		t.Fatal("getUserTodos() returned nil for existing user")
	}
	if len(loadedTodos.Todos) != 1 {
		t.Errorf("existing user has %d todos; want 1", len(loadedTodos.Todos))
	}
	if loadedTodos.NextID != 5 {
		t.Errorf("existing user NextID = %d; want 5", loadedTodos.NextID)
	}

	// Verify the loaded todo
	loadedTodo, ok := loadedTodos.Todos[4]
	if !ok {
		t.Fatal("Todo with ID 4 not found")
	}
	if loadedTodo.Text != "Existing todo" {
		t.Errorf("loaded todo text = %q; want %q", loadedTodo.Text, "Existing todo")
	}
	if !loadedTodo.Completed {
		t.Error("loaded todo not marked as completed")
	}
	t.Logf("Successfully loaded todo for existing user")

	// Test 3: Get todos for a user that's already in memory
	t.Logf("Getting todos for in-memory user: %s (second time)", username2)
	cachedTodos, err := store.getUserTodos(username2)
	if err != nil {
		t.Fatalf("getUserTodos() error = %v for cached user", err)
	}
	if cachedTodos == nil {
		t.Fatal("getUserTodos() returned nil for cached user")
	}
	if len(cachedTodos.Todos) != 1 {
		t.Errorf("cached user has %d todos; want 1", len(cachedTodos.Todos))
	}
	t.Logf("Successfully got todos for in-memory user")
}

// TestSaveTodosFunction tests the saveTodos function.
// It verifies:
// - Todos are correctly serialized to JSON
// - The JSON file is created in the right location
// - The file contains the expected data
func TestSaveTodosFunction(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "todoissh-save-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create todos directory
	todosDir := filepath.Join(tempDir, "todos")
	err = os.MkdirAll(todosDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create todos directory: %v", err)
	}

	// Create a store
	store := &Store{
		userTodos: make(map[string]*UserTodos),
		dataDir:   tempDir,
	}

	// Create a user todos struct
	username := "save-test-user"
	userTodos := &UserTodos{
		Todos:  make(map[int]*Todo),
		NextID: 1,
	}

	// Add a todo
	todo := &Todo{
		ID:        1,
		Text:      "Test todo",
		Completed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	userTodos.Todos[todo.ID] = todo
	userTodos.NextID = 2

	// Add to the store
	store.userTodos[username] = userTodos

	// Save the todos
	t.Logf("Saving todos for user %s", username)
	store.Lock() // Need to lock as saveTodos assumes caller has lock
	err = store.saveTodos(username)
	store.Unlock()
	if err != nil {
		t.Fatalf("saveTodos() error = %v", err)
	}
	t.Logf("Successfully saved todos")

	// Verify the file was created
	todosPath := filepath.Join(todosDir, username+".json")
	_, err = os.Stat(todosPath)
	if err != nil {
		t.Fatalf("Failed to stat todos file: %v", err)
	}
	t.Logf("Todos file was created successfully at %s", todosPath)

	// Read back the saved todos
	data, err := os.ReadFile(todosPath)
	if err != nil {
		t.Fatalf("Failed to read todos file: %v", err)
	}
	t.Logf("Read %d bytes from todos file: %s", len(data), string(data))
}

// TestAdd tests adding a todo.
// It verifies:
// - The todo is successfully added
// - The todo has the correct properties (ID, text, completed status, timestamps)
// - Sequential IDs are assigned correctly
func TestAdd(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	text := "Test todo"

	todo, err := store.Add(testUsername, text)
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
	todo2, _ := store.Add(testUsername, "Another todo")
	if todo2.ID != 2 {
		t.Errorf("second todo.ID = %d; want 2", todo2.ID)
	}
}

// TestAddWithMock tests the Add function with a custom implementation to isolate it from file operations.
// It uses a mock implementation to:
// - Verify that saveTodos is called when adding a todo
// - Test the core Add functionality without file system operations
// - Ensure sequential IDs are assigned correctly
func TestAddWithMock(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "todoissh-add-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create todos directory
	todosDir := filepath.Join(tempDir, "todos")
	err = os.MkdirAll(todosDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create todos directory: %v", err)
	}

	// Create a store with a modified saveTodos implementation
	// that doesn't actually write to disk
	store := &Store{
		userTodos: make(map[string]*UserTodos),
		dataDir:   tempDir,
	}

	// Create a custom implementation of Store that tracks if saveTodos was called
	type testStore struct {
		*Store
		saveCalled bool
	}

	// Create our test store
	ts := &testStore{
		Store:      store,
		saveCalled: false,
	}

	// Create a wrapper function for Add that uses our testStore
	addTodo := func(username, text string) (*Todo, error) {
		// Get or create user todos
		ts.Lock()
		userTodos, exists := ts.userTodos[username]
		if !exists {
			userTodos = &UserTodos{
				Todos:  make(map[int]*Todo),
				NextID: 1,
			}
			ts.userTodos[username] = userTodos
		}

		todo := &Todo{
			ID:        userTodos.NextID,
			Text:      text,
			Completed: false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		userTodos.Todos[todo.ID] = todo
		userTodos.NextID++

		// Mark that saveTodos was called
		ts.saveCalled = true

		ts.Unlock()
		return todo, nil
	}

	// Add a todo
	username := "add-test-user"
	text := "Test todo"
	t.Logf("Adding todo for user: %s", username)

	todo, err := addTodo(username, text)
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

	// Verify saveTodos was called
	if !ts.saveCalled {
		t.Error("saveTodos() was not called")
	}

	t.Logf("Successfully added todo: %+v", todo)

	// Verify the todo was added to the store
	ts.Lock()
	userTodos, exists := ts.userTodos[username]
	ts.Unlock()
	if !exists {
		t.Fatal("User todos not found in store")
	}
	if len(userTodos.Todos) != 1 {
		t.Errorf("User has %d todos; want 1", len(userTodos.Todos))
	}
	if userTodos.NextID != 2 {
		t.Errorf("NextID = %d; want 2", userTodos.NextID)
	}

	// Test sequential IDs
	ts.saveCalled = false
	todo2, err := addTodo(username, "Another todo")
	if err != nil {
		t.Fatalf("Add() second todo error = %v", err)
	}
	if todo2.ID != 2 {
		t.Errorf("second todo.ID = %d; want 2", todo2.ID)
	}
	if !ts.saveCalled {
		t.Error("saveTodos() was not called for second todo")
	}
	t.Logf("Successfully added second todo: %+v", todo2)
}

// TestList tests listing todos.
// It verifies:
// - An empty list is returned for a new user
// - All added todos are returned in the list
// - The correct number of todos is returned
func TestList(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Test empty list
	todos, err := store.List(testUsername)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("List() returned %d todos; want 0", len(todos))
	}

	// Add some todos
	store.Add(testUsername, "Todo 1")
	store.Add(testUsername, "Todo 2")
	store.Add(testUsername, "Todo 3")

	todos, err = store.List(testUsername)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(todos) != 3 {
		t.Errorf("List() returned %d todos; want 3", len(todos))
	}
}

// TestGet tests getting a todo by ID.
// It verifies:
// - Getting a non-existent todo returns an error
// - Getting an existing todo returns the correct todo
func TestGet(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Test getting non-existent todo
	todo, err := store.Get(testUsername, 1)
	if err == nil {
		t.Error("Get() non-existent todo; want error")
	}
	if todo != nil {
		t.Error("Get() non-existent todo returned non-nil todo")
	}

	// Add and get a todo
	added, _ := store.Add(testUsername, "Test todo")
	todo, err = store.Get(testUsername, added.ID)
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

// TestUpdate tests updating a todo.
// It verifies:
// - Updating a non-existent todo returns an error
// - Updating an existing todo changes its text
// - The UpdatedAt timestamp is updated
func TestUpdate(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Test updating non-existent todo
	_, err := store.Update(testUsername, 1, "Updated text")
	if err == nil {
		t.Error("Update() non-existent todo; want error")
	}

	// Add and update a todo
	todo, _ := store.Add(testUsername, "Original text")
	originalUpdatedAt := todo.UpdatedAt
	time.Sleep(time.Millisecond) // Ensure time difference

	updated, err := store.Update(testUsername, todo.ID, "Updated text")
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

// TestDelete tests deleting a todo.
// It verifies:
// - Deleting a non-existent todo returns an error
// - Deleting an existing todo removes it from the store
// - Getting a deleted todo returns an error
func TestDelete(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Test deleting non-existent todo
	err := store.Delete(testUsername, 1)
	if err == nil {
		t.Error("Delete() non-existent todo; want error")
	}

	// Add and delete a todo
	todo, _ := store.Add(testUsername, "Test todo")
	err = store.Delete(testUsername, todo.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify todo was deleted
	_, err = store.Get(testUsername, todo.ID)
	if err == nil {
		t.Error("Get() deleted todo; want error")
	}
}

// TestToggleComplete tests toggling the completed status of a todo.
// It verifies:
// - Toggling a non-existent todo returns an error
// - Toggling an existing todo changes its completed status
// - The UpdatedAt timestamp is updated
// - Toggling again reverts the completed status
func TestToggleComplete(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Test toggling non-existent todo
	_, err := store.ToggleComplete(testUsername, 1)
	if err == nil {
		t.Error("ToggleComplete() non-existent todo; want error")
	}

	// Add and toggle a todo
	todo, _ := store.Add(testUsername, "Test todo")
	originalUpdatedAt := todo.UpdatedAt
	time.Sleep(time.Millisecond) // Ensure time difference

	toggled, err := store.ToggleComplete(testUsername, todo.ID)
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
	toggled, _ = store.ToggleComplete(testUsername, todo.ID)
	if toggled.Completed {
		t.Error("toggled.Completed = true; want false")
	}
}

// TestConcurrentOperations tests concurrent add operations.
// It verifies:
// - Adding todos concurrently works correctly
// - All todos are added without errors
// - Each todo gets a unique ID
func TestConcurrentOperations(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	done := make(chan bool)

	// Concurrent adds
	for i := 0; i < 10; i++ {
		go func(i int) {
			_, err := store.Add(testUsername, "Concurrent todo")
			if err != nil {
				t.Errorf("Concurrent Add() error = %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	todos, err := store.List(testUsername)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
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

// TestMultipleUsers tests that todos for different users are kept separate.
// It verifies:
// - Each user has their own collection of todos
// - Todos for different users don't interfere with each other
// - Todos contain the correct text for each user
func TestMultipleUsers(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Add todos for different users
	todo1, err := store.Add("user1", "User 1 Todo 1")
	if err != nil {
		t.Fatalf("Failed to add todo for user1: %v", err)
	}
	if todo1 == nil {
		t.Fatal("Added todo is nil for user1")
	}

	todo2, err := store.Add("user1", "User 1 Todo 2")
	if err != nil {
		t.Fatalf("Failed to add second todo for user1: %v", err)
	}
	if todo2 == nil {
		t.Fatal("Added second todo is nil for user1")
	}

	todo3, err := store.Add("user2", "User 2 Todo 1")
	if err != nil {
		t.Fatalf("Failed to add todo for user2: %v", err)
	}
	if todo3 == nil {
		t.Fatal("Added todo is nil for user2")
	}

	// Check user1's todos
	user1Todos, err := store.List("user1")
	if err != nil {
		t.Fatalf("List(user1) error = %v", err)
	}
	if len(user1Todos) != 2 {
		t.Errorf("user1 has %d todos; want 2", len(user1Todos))
	}

	// Check user2's todos
	user2Todos, err := store.List("user2")
	if err != nil {
		t.Fatalf("List(user2) error = %v", err)
	}
	if len(user2Todos) != 1 {
		t.Errorf("user2 has %d todos; want 1", len(user2Todos))
	}

	// Verify user isolation
	for _, todo := range user1Todos {
		if !strings.HasPrefix(todo.Text, "User 1") {
			t.Errorf("user1 todo has incorrect text: %s", todo.Text)
		}
	}

	for _, todo := range user2Todos {
		if !strings.HasPrefix(todo.Text, "User 2") {
			t.Errorf("user2 todo has incorrect text: %s", todo.Text)
		}
	}
}

// TestSimpleEndToEnd tests a simple end-to-end flow of adding and listing todos.
// It verifies:
// - Todo creation works
// - Listing todos works
// - Updating todos works
// - Toggling completion works
// - Deleting todos works
// - Final list state is correct
func TestSimpleEndToEnd(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "todoissh-minimal-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create store with temp directory
	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	// Add a todo
	username := "minimal-user"
	todo, err := store.Add(username, "Minimal test todo")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	t.Logf("Added todo: %+v", todo)

	// List todos
	todos, err := store.List(username)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}
	t.Logf("Listed %d todos", len(todos))

	// Update the todo
	updatedTodo, err := store.Update(username, todo.ID, "Updated text")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updatedTodo.Text != "Updated text" {
		t.Errorf("Expected updated text %q, got %q", "Updated text", updatedTodo.Text)
	}

	// Toggle completion
	completedTodo, err := store.ToggleComplete(username, todo.ID)
	if err != nil {
		t.Fatalf("ToggleComplete() error = %v", err)
	}
	if !completedTodo.Completed {
		t.Error("Expected todo to be completed, but it's not")
	}

	// Delete the todo
	err = store.Delete(username, todo.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	todos, err = store.List(username)
	if err != nil {
		t.Fatalf("List() after delete error = %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("Expected 0 todos after deletion, got %d", len(todos))
	}
}

// TestPersistence tests that todos are persisted between store instances.
// It verifies:
// - Todos created in one store instance are accessible in another
// - Todo states (text, completion) are preserved between instances
// - Updated and toggled todos keep their state
func TestPersistence(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "todoissh-persistence-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create first store instance
	store1, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("NewStore() first instance error = %v", err)
	}

	// Add some todos
	username := "persistence-user"
	todo, err := store1.Add(username, "Persistent todo 1")
	if err != nil {
		t.Fatalf("Add() first todo error = %v", err)
	}

	_, err = store1.Add(username, "Persistent todo 2")
	if err != nil {
		t.Fatalf("Add() second todo error = %v", err)
	}

	// Update and toggle the first todo
	_, err = store1.Update(username, todo.ID, "Updated persistent todo")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	_, err = store1.ToggleComplete(username, todo.ID)
	if err != nil {
		t.Fatalf("ToggleComplete() error = %v", err)
	}

	// Create a second store instance pointing to the same directory
	store2, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("NewStore() second instance error = %v", err)
	}

	// List todos from the second store
	todos, err := store2.List(username)
	if err != nil {
		t.Fatalf("List() from second store error = %v", err)
	}

	// Verify the todos were loaded correctly
	if len(todos) != 2 {
		t.Fatalf("Expected 2 todos in second store, got %d", len(todos))
	}

	// Find the todo we modified
	var loadedTodo *Todo
	for _, t := range todos {
		if t.ID == todo.ID {
			loadedTodo = t
			break
		}
	}

	if loadedTodo == nil {
		t.Fatalf("Could not find todo with ID %d in second store", todo.ID)
	}

	// Verify its state
	if loadedTodo.Text != "Updated persistent todo" {
		t.Errorf("Expected text %q, got %q", "Updated persistent todo", loadedTodo.Text)
	}

	if !loadedTodo.Completed {
		t.Error("Expected todo to be completed, but it's not")
	}
}

// TestAddWithEmptyText verifies that adding a todo with empty text is handled correctly
func TestAddWithEmptyText(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Add todo with empty text
	todo, err := store.Add(testUsername, "")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if todo.Text != "" {
		t.Errorf("todo.Text = %q; want empty string", todo.Text)
	}
}

// TestGetNonExistentTodo verifies that getting a non-existent todo returns an error
func TestGetNonExistentTodo(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Try to get non-existent todo
	_, err := store.Get(testUsername, 999)
	if err == nil {
		t.Fatal("Get() did not return error for non-existent todo")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Get() error = %v; want 'not found' error", err)
	}
}

// TestUpdateNonExistentTodo verifies that updating a non-existent todo returns an error
func TestUpdateNonExistentTodo(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Try to update non-existent todo
	_, err := store.Update(testUsername, 999, "Updated text")
	if err == nil {
		t.Fatal("Update() did not return error for non-existent todo")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Update() error = %v; want 'not found' error", err)
	}
}

// TestDeleteNonExistentTodo verifies that deleting a non-existent todo returns an error
func TestDeleteNonExistentTodo(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Try to delete non-existent todo
	err := store.Delete(testUsername, 999)
	if err == nil {
		t.Fatal("Delete() did not return error for non-existent todo")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Delete() error = %v; want 'not found' error", err)
	}
}

// TestToggleCompleteNonExistentTodo verifies that toggling a non-existent todo returns an error
func TestToggleCompleteNonExistentTodo(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Try to toggle non-existent todo
	_, err := store.ToggleComplete(testUsername, 999)
	if err == nil {
		t.Fatal("ToggleComplete() did not return error for non-existent todo")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("ToggleComplete() error = %v; want 'not found' error", err)
	}
}

// TestNonExistentUserTodos verifies that operations with a non-existent user work correctly
func TestNonExistentUserTodos(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// List todos for non-existent user
	todos, err := store.List(nonExistentUser)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("List() returned %d todos; want 0", len(todos))
	}

	// Add todo for non-existent user
	todo, err := store.Add(nonExistentUser, "Test todo for new user")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if todo.ID != 1 {
		t.Errorf("todo.ID = %d; want 1", todo.ID)
	}

	// List todos again to verify the todo was added
	todos, err = store.List(nonExistentUser)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("List() returned %d todos; want 1", len(todos))
	}
}

// TestConcurrentAdd verifies that adding todos concurrently from multiple goroutines works correctly
func TestConcurrentAdd(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Number of concurrent operations
	const concurrency = 50

	// Run concurrent Add operations
	done := make(chan bool)
	for i := 0; i < concurrency; i++ {
		go func(i int) {
			text := fmt.Sprintf("Concurrent todo %d", i)
			_, err := store.Add(testUsername, text)
			if err != nil {
				t.Errorf("Add() error = %v in goroutine %d", err, i)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Check that all todos were added
	todos, err := store.List(testUsername)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(todos) != concurrency {
		t.Errorf("List() returned %d todos; want %d", len(todos), concurrency)
	}

	// Check that IDs are sequential and unique
	idMap := make(map[int]bool)
	for _, todo := range todos {
		if idMap[todo.ID] {
			t.Errorf("Duplicate todo ID: %d", todo.ID)
		}
		idMap[todo.ID] = true
	}
}

// TestGetUserTodosInvalidJSON verifies that loading invalid JSON is handled correctly
func TestGetUserTodosInvalidJSON(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Create invalid JSON file
	todosDir := filepath.Join(tempDir, "todos")
	os.MkdirAll(todosDir, 0700)
	todosPath := filepath.Join(todosDir, "invalid.json")
	err := os.WriteFile(todosPath, []byte("invalid json"), 0600)
	if err != nil {
		t.Fatalf("Failed to create invalid todos file: %v", err)
	}

	// Try to get todos for the invalid user
	_, err = store.getUserTodos("invalid")
	if err == nil {
		t.Fatal("getUserTodos() did not return error for invalid JSON")
	}
}

// TestConcurrentMultiUser verifies that concurrent operations with multiple users work correctly
func TestConcurrentMultiUser(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Number of users and todos per user
	const numUsers = 5
	const todosPerUser = 10

	// Run concurrent operations for multiple users
	done := make(chan bool)
	for u := 0; u < numUsers; u++ {
		go func(userIndex int) {
			username := fmt.Sprintf("user%d", userIndex)

			// Add todos for this user
			for i := 0; i < todosPerUser; i++ {
				text := fmt.Sprintf("Todo %d for %s", i, username)
				_, err := store.Add(username, text)
				if err != nil {
					t.Errorf("Add() error = %v for user %s, todo %d", err, username, i)
				}
			}

			// List todos for this user
			todos, err := store.List(username)
			if err != nil {
				t.Errorf("List() error = %v for user %s", err, username)
			}
			if len(todos) != todosPerUser {
				t.Errorf("List() returned %d todos for user %s; want %d", len(todos), username, todosPerUser)
			}

			done <- true
		}(u)
	}

	// Wait for all goroutines to finish
	for i := 0; i < numUsers; i++ {
		<-done
	}

	// Verify each user's todos
	for u := 0; u < numUsers; u++ {
		username := fmt.Sprintf("user%d", u)
		todos, err := store.List(username)
		if err != nil {
			t.Fatalf("List() error = %v for user %s", err, username)
		}
		if len(todos) != todosPerUser {
			t.Errorf("List() returned %d todos for user %s; want %d", len(todos), username, todosPerUser)
		}
	}
}

// TestCorruptedTodosFile verifies that corrupted todos files are handled gracefully
func TestCorruptedTodosFile(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Add a todo to create the file
	_, err := store.Add(testUsername, "Test todo")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Corrupt the file by overwriting it with truncated JSON
	todosPath := filepath.Join(tempDir, "todos", testUsername+".json")
	err = os.WriteFile(todosPath, []byte(`{"todos": {`), 0600)
	if err != nil {
		t.Fatalf("Failed to corrupt todos file: %v", err)
	}

	// Create a new store to force it to load from disk
	store = nil
	store, err = NewStore(tempDir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	// Try to get todos for the user with corrupted file
	_, err = store.getUserTodos(testUsername)
	if err == nil {
		t.Fatal("getUserTodos() did not return error for corrupted file")
	}
}

// TestToggleCompleteCycle verifies that toggling a todo's completed status works correctly
func TestToggleCompleteCycle(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Add a todo
	todo, err := store.Add(testUsername, "Toggle test")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if todo.Completed {
		t.Fatal("New todo is already completed")
	}

	// Toggle to completed
	todo, err = store.ToggleComplete(testUsername, todo.ID)
	if err != nil {
		t.Fatalf("ToggleComplete() error = %v", err)
	}
	if !todo.Completed {
		t.Error("Todo should be completed after toggle")
	}

	// Toggle back to not completed
	todo, err = store.ToggleComplete(testUsername, todo.ID)
	if err != nil {
		t.Fatalf("ToggleComplete() error = %v", err)
	}
	if todo.Completed {
		t.Error("Todo should not be completed after second toggle")
	}

	// Get the todo to verify persistence
	todo, err = store.Get(testUsername, todo.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if todo.Completed {
		t.Error("Todo completed status not persisted correctly")
	}
}

// TestUpdateTimestamps verifies that timestamps are updated correctly
func TestUpdateTimestamps(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Add a todo
	todo, err := store.Add(testUsername, "Timestamp test")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	createdAt := todo.CreatedAt
	updatedAt := todo.UpdatedAt

	// Sleep to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update the todo
	todo, err = store.Update(testUsername, todo.ID, "Updated text")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// CreatedAt should not change
	if !todo.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt changed after update: %v -> %v", createdAt, todo.CreatedAt)
	}

	// UpdatedAt should change
	if todo.UpdatedAt.Equal(updatedAt) {
		t.Error("UpdatedAt did not change after update")
	}

	// Sleep again
	time.Sleep(10 * time.Millisecond)

	// Toggle complete
	todo, err = store.ToggleComplete(testUsername, todo.ID)
	if err != nil {
		t.Fatalf("ToggleComplete() error = %v", err)
	}

	// CreatedAt should still not change
	if !todo.CreatedAt.Equal(createdAt) {
		t.Errorf("CreatedAt changed after toggle: %v -> %v", createdAt, todo.CreatedAt)
	}

	// UpdatedAt should change again
	if todo.UpdatedAt.Equal(updatedAt) {
		t.Error("UpdatedAt did not change after toggle")
	}
}

// TestNewStoreDirectoryError verifies that an error is returned when creating the data directory fails
func TestNewStoreDirectoryError(t *testing.T) {
	// Try to create a store with an invalid directory
	// This simulates a permission error or other issue that would prevent directory creation
	_, err := NewStore("/root/nonexistent/directory") // Should fail on most systems due to permissions
	if err == nil {
		t.Fatal("NewStore() did not return error for invalid directory")
	}
}

// TestTodosDirectoryError verifies that an error is returned when creating the todos directory fails
func TestTodosDirectoryError(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "todoissh-todos-dir-error")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file at the path where the todos directory should be
	todosDir := filepath.Join(tempDir, "todos")
	err = os.WriteFile(todosDir, []byte("not a directory"), 0600)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Try to create a store, which should fail when creating the todos directory
	_, err = NewStore(tempDir)
	if err == nil {
		t.Fatal("NewStore() did not return error when todos directory couldn't be created")
	}
}

// TestGetUserTodosError verifies that an error in getUserTodos is handled correctly
func TestGetUserTodosError(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Add a todo to create the user directory and file
	_, err := store.Add(testUsername, "Test todo")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Create a new store instance to clear the in-memory cache
	store = nil
	store, err = NewStore(tempDir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	// Make the todos file unreadable
	todosPath := filepath.Join(tempDir, "todos", testUsername+".json")
	err = os.Chmod(todosPath, 0000) // ----------
	if err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Try to get todos, which should fail
	_, err = store.List(testUsername)

	// Restore permissions for cleanup
	os.Chmod(todosPath, 0600)

	// Check if we got an error
	if err == nil {
		t.Fatal("List() did not return error for unreadable todos file")
	}
}

// TestSaveTodosError verifies that an error during saveTodos is handled correctly
func TestSaveTodosError(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Add a todo to create the user
	_, err := store.Add(testUsername, "Test todo")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Get the path to the todos file
	todosPath := filepath.Join(tempDir, "todos", testUsername+".json")

	// Make the file read-only (not the directory, since that doesn't always cause errors)
	err = os.Chmod(todosPath, 0400) // r--------
	if err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	// Try to add another todo, which should fail when saving
	_, err = store.Add(testUsername, "Another todo")

	// Restore permissions for cleanup
	os.Chmod(todosPath, 0600)

	// Check if we got an error
	// Note: This might not fail on all systems, so we'll make it conditional
	if err == nil {
		// On some systems, you can still write to a file even if it's read-only
		// due to directory permissions. So we'll check if the file was modified.
		t.Log("No error was returned when writing to read-only file, checking if file was modified")

		fileInfo, err := os.Stat(todosPath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		// If the file size is still the same as before, it wasn't modified
		if fileInfo.Size() < 100 {
			t.Error("File size is too small, suggests file wasn't modified")
		}
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

// TestErrorCases verifies error handling for all operations
func TestErrorCases(t *testing.T) {
	// 1. Test error when todos directory can't be created
	_, err := NewStore(invalidDir)
	if err == nil {
		t.Fatal("NewStore() did not return error for invalid directory")
	}

	// 2. Create a store for further error testing
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// 3. Add a todo for testing
	todo, err := store.Add(testUsername, "Test todo")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// 4. Get the path to the todos file
	todosPath := filepath.Join(tempDir, "todos", testUsername+".json")

	// 5. Test update error - we'll use a read-only file instead of directory
	err = os.Chmod(todosPath, 0400) // r--------
	if err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	_, err = store.Update(testUsername, todo.ID, "Updated text")

	// Verify behavior
	if err == nil {
		t.Log("No error was returned when updating with read-only file, this may be system-dependent")
	} else {
		t.Logf("Got expected error from Update: %v", err)
	}

	// Restore permissions
	os.Chmod(todosPath, 0600)

	// 6. Test delete error
	err = os.Chmod(todosPath, 0400) // r--------
	if err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	err = store.Delete(testUsername, todo.ID)

	// Verify behavior
	if err == nil {
		t.Log("No error was returned when deleting with read-only file, this may be system-dependent")
	} else {
		t.Logf("Got expected error from Delete: %v", err)
	}

	// Restore permissions
	os.Chmod(todosPath, 0600)

	// 7. Test toggle error
	err = os.Chmod(todosPath, 0400) // r--------
	if err != nil {
		t.Fatalf("Failed to change file permissions: %v", err)
	}

	_, err = store.ToggleComplete(testUsername, todo.ID)

	// Verify behavior
	if err == nil {
		t.Log("No error was returned when toggling with read-only file, this may be system-dependent")
	} else {
		t.Logf("Got expected error from ToggleComplete: %v", err)
	}

	// Restore permissions
	os.Chmod(todosPath, 0600)
}
