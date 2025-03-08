package todo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Todo represents a single todo item
type Todo struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserTodos stores todos for a single user
type UserTodos struct {
	Todos  map[int]*Todo `json:"todos"`
	NextID int           `json:"next_id"`
}

// Store manages todos for multiple users
type Store struct {
	sync.RWMutex
	userTodos map[string]*UserTodos // map[username]todos
	dataDir   string
}

// NewStore creates a new todo store with the given data directory
func NewStore(dataDir string) (*Store, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	store := &Store{
		userTodos: make(map[string]*UserTodos),
		dataDir:   dataDir,
	}

	// Create the todos directory if it doesn't exist
	todosDir := filepath.Join(dataDir, "todos")
	if err := os.MkdirAll(todosDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create todos directory: %v", err)
	}

	return store, nil
}

// getUserTodos gets or creates a user's todos
func (s *Store) getUserTodos(username string) (*UserTodos, error) {
	s.Lock()
	defer s.Unlock()

	userTodos, exists := s.userTodos[username]
	if exists {
		return userTodos, nil
	}

	// Try to load from disk
	todosPath := filepath.Join(s.dataDir, "todos", username+".json")
	if _, err := os.Stat(todosPath); err == nil {
		// File exists, load it
		data, err := os.ReadFile(todosPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read todos file: %v", err)
		}

		var userTodos UserTodos
		if err := json.Unmarshal(data, &userTodos); err != nil {
			return nil, fmt.Errorf("failed to parse todos file: %v", err)
		}

		s.userTodos[username] = &userTodos
		return &userTodos, nil
	}

	// Create new user todos
	userTodos = &UserTodos{
		Todos:  make(map[int]*Todo),
		NextID: 1,
	}

	s.userTodos[username] = userTodos
	return userTodos, nil
}

// saveTodos saves a user's todos to disk
func (s *Store) saveTodos(username string) error {
	// We assume the caller already has the lock
	userTodos, exists := s.userTodos[username]
	if !exists {
		return fmt.Errorf("no todos found for user %s", username)
	}

	data, err := json.MarshalIndent(userTodos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize todos: %v", err)
	}

	todosPath := filepath.Join(s.dataDir, "todos", username+".json")
	return os.WriteFile(todosPath, data, 0600)
}

// Add adds a new todo for the specified user
func (s *Store) Add(username, text string) (*Todo, error) {
	s.Lock()
	defer s.Unlock()

	// Get or create user todos (without locking since we already have the lock)
	userTodos, exists := s.userTodos[username]
	if !exists {
		// Create new user todos
		userTodos = &UserTodos{
			Todos:  make(map[int]*Todo),
			NextID: 1,
		}
		s.userTodos[username] = userTodos
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

	// Save to disk
	if err := s.saveTodos(username); err != nil {
		return nil, err
	}

	return todo, nil
}

// List returns all todos for the specified user
func (s *Store) List(username string) ([]*Todo, error) {
	userTodos, err := s.getUserTodos(username)
	if err != nil {
		return nil, err
	}

	s.RLock()
	defer s.RUnlock()

	todos := make([]*Todo, 0, len(userTodos.Todos))
	for _, todo := range userTodos.Todos {
		todos = append(todos, todo)
	}
	return todos, nil
}

// Get returns the todo with the specified ID for the specified user
func (s *Store) Get(username string, id int) (*Todo, error) {
	userTodos, err := s.getUserTodos(username)
	if err != nil {
		return nil, err
	}

	s.RLock()
	defer s.RUnlock()

	todo, ok := userTodos.Todos[id]
	if !ok {
		return nil, fmt.Errorf("todo with ID %d not found", id)
	}
	return todo, nil
}

// Update updates the todo with the specified ID for the specified user
func (s *Store) Update(username string, id int, text string) (*Todo, error) {
	userTodos, err := s.getUserTodos(username)
	if err != nil {
		return nil, err
	}

	s.Lock()
	defer s.Unlock()

	todo, ok := userTodos.Todos[id]
	if !ok {
		return nil, fmt.Errorf("todo with ID %d not found", id)
	}

	todo.Text = text
	todo.UpdatedAt = time.Now()

	// Save to disk
	if err := s.saveTodos(username); err != nil {
		return nil, err
	}

	return todo, nil
}

// Delete deletes the todo with the specified ID for the specified user
func (s *Store) Delete(username string, id int) error {
	userTodos, err := s.getUserTodos(username)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	if _, ok := userTodos.Todos[id]; !ok {
		return fmt.Errorf("todo with ID %d not found", id)
	}

	delete(userTodos.Todos, id)

	// Save to disk
	return s.saveTodos(username)
}

// ToggleComplete toggles the completed status of the todo with the specified ID for the specified user
func (s *Store) ToggleComplete(username string, id int) (*Todo, error) {
	userTodos, err := s.getUserTodos(username)
	if err != nil {
		return nil, err
	}

	s.Lock()
	defer s.Unlock()

	todo, ok := userTodos.Todos[id]
	if !ok {
		return nil, fmt.Errorf("todo with ID %d not found", id)
	}

	todo.Completed = !todo.Completed
	todo.UpdatedAt = time.Now()

	// Save to disk
	if err := s.saveTodos(username); err != nil {
		return nil, err
	}

	return todo, nil
}
