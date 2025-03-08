package user

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	IsNew        bool   `json:"-"` // Not stored, used for first-time login detection
}

// Store manages users and their authentication
type Store struct {
	users map[string]*User
	mutex sync.RWMutex
	path  string
}

// NewStore creates a new user store
func NewStore(dataDir string) (*Store, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	path := filepath.Join(dataDir, "users.json")
	store := &Store{
		users: make(map[string]*User),
		path:  path,
	}

	// Load existing users if the file exists
	if _, err := os.Stat(path); err == nil {
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("failed to load users: %v", err)
		}
	}

	return store, nil
}

// Authenticate verifies the username and password
// Returns a user object and a boolean indicating if authentication was successful
func (s *Store) Authenticate(username, password string) (*User, bool) {
	s.mutex.RLock()
	user, exists := s.users[username]
	s.mutex.RUnlock()

	if !exists {
		// User doesn't exist, mark as new for registration
		return &User{Username: username, IsNew: true}, false
	}

	// Verify password
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return user, err == nil
}

// Register creates a new user or updates an existing user's password
func (s *Store) Register(username, password string) error {
	// Generate password hash
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create or update user
	s.users[username] = &User{
		Username:     username,
		PasswordHash: string(hash),
	}

	// Save changes
	return s.save()
}

// GetUser retrieves a user by username
func (s *Store) GetUser(username string) *User {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if user, exists := s.users[username]; exists {
		return user
	}
	return nil
}

// load reads users from disk
func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		// Empty file, no users yet
		return nil
	}

	var users map[string]*User
	if err := json.Unmarshal(data, &users); err != nil {
		return err
	}

	s.users = users
	return nil
}

// save writes users to disk
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0600)
}
