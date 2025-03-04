package todo

import (
	"fmt"
	"sync"
	"time"
)

type Todo struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	sync.RWMutex
	todos  map[int]*Todo
	nextID int
}

func NewStore() *Store {
	return &Store{
		todos:  make(map[int]*Todo),
		nextID: 1,
	}
}

func (s *Store) Add(text string) (*Todo, error) {
	s.Lock()
	defer s.Unlock()

	todo := &Todo{
		ID:        s.nextID,
		Text:      text,
		Completed: false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.todos[todo.ID] = todo
	s.nextID++

	return todo, nil
}

func (s *Store) List() []*Todo {
	s.RLock()
	defer s.RUnlock()

	todos := make([]*Todo, 0, len(s.todos))
	for _, todo := range s.todos {
		todos = append(todos, todo)
	}
	return todos
}

func (s *Store) Get(id int) (*Todo, error) {
	s.RLock()
	defer s.RUnlock()

	todo, ok := s.todos[id]
	if !ok {
		return nil, fmt.Errorf("todo with ID %d not found", id)
	}
	return todo, nil
}

func (s *Store) Update(id int, text string) (*Todo, error) {
	s.Lock()
	defer s.Unlock()

	todo, ok := s.todos[id]
	if !ok {
		return nil, fmt.Errorf("todo with ID %d not found", id)
	}

	todo.Text = text
	todo.UpdatedAt = time.Now()
	return todo, nil
}

func (s *Store) Delete(id int) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.todos[id]; !ok {
		return fmt.Errorf("todo with ID %d not found", id)
	}

	delete(s.todos, id)
	return nil
}

func (s *Store) ToggleComplete(id int) (*Todo, error) {
	s.Lock()
	defer s.Unlock()

	todo, ok := s.todos[id]
	if !ok {
		return nil, fmt.Errorf("todo with ID %d not found", id)
	}

	todo.Completed = !todo.Completed
	todo.UpdatedAt = time.Now()
	return todo, nil
} 