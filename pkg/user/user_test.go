package user

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Test constants
const (
	testUsername = "testuser"
	testPassword = "test-password123"
)

// setupTestStore creates a temporary test directory and initializes a Store.
// It returns the initialized store and the temporary directory path.
// The caller is responsible for calling cleanupTestStore with the returned path.
func setupTestStore(t *testing.T) (*Store, string) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "todoissh-user-test")
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

// TestNewStore verifies that a new store is created correctly
func TestNewStore(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	if store.users == nil {
		t.Error("store.users is nil")
	}
	if store.path != filepath.Join(tempDir, "users.json") {
		t.Errorf("store.path = %s; want %s", store.path, filepath.Join(tempDir, "users.json"))
	}

	// Verify data directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Data directory was not created")
	}
}

// TestRegisterAndAuthenticate verifies that user registration and authentication work correctly
func TestRegisterAndAuthenticate(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Register a new user
	err := store.Register(testUsername, testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Verify the user was added to the store
	if _, exists := store.users[testUsername]; !exists {
		t.Errorf("User %s was not added to the store", testUsername)
	}

	// Authenticate with correct password
	user, ok := store.Authenticate(testUsername, testPassword)
	if !ok {
		t.Error("Authenticate() failed with correct password")
	}
	if user == nil {
		t.Fatal("Authenticate() returned nil user")
	}
	if user.Username != testUsername {
		t.Errorf("user.Username = %s; want %s", user.Username, testUsername)
	}
	if user.IsNew {
		t.Error("user.IsNew = true; want false for existing user")
	}

	// Authenticate with incorrect password
	_, ok = store.Authenticate(testUsername, "wrong-password")
	if ok {
		t.Error("Authenticate() succeeded with incorrect password")
	}

	// Authenticate with non-existent user
	user, ok = store.Authenticate("nonexistent", "password")
	if ok {
		t.Error("Authenticate() succeeded with non-existent user")
	}
	if user == nil {
		t.Fatal("Authenticate() returned nil user for non-existent user")
	}
	if !user.IsNew {
		t.Error("user.IsNew = false; want true for non-existent user")
	}
}

// TestPasswordHashing verifies that passwords are properly hashed
func TestPasswordHashing(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Register a new user
	err := store.Register(testUsername, testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Verify the password is hashed
	if store.users[testUsername].PasswordHash == testPassword {
		t.Error("Password was not hashed")
	}

	// Verify the hash works with bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(store.users[testUsername].PasswordHash), []byte(testPassword))
	if err != nil {
		t.Errorf("bcrypt.CompareHashAndPassword() error = %v", err)
	}
}

// TestGetUser verifies that GetUser works correctly
func TestGetUser(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Register a new user
	err := store.Register(testUsername, testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Get existing user
	user := store.GetUser(testUsername)
	if user == nil {
		t.Fatal("GetUser() returned nil for existing user")
	}
	if user.Username != testUsername {
		t.Errorf("user.Username = %s; want %s", user.Username, testUsername)
	}

	// Get non-existent user
	user = store.GetUser("nonexistent")
	if user != nil {
		t.Errorf("GetUser() returned non-nil for non-existent user: %+v", user)
	}
}

// TestPersistence verifies that users are persisted to disk and can be loaded
func TestPersistence(t *testing.T) {
	// Create first store and register user
	store1, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	err := store1.Register(testUsername, testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Create second store with same directory to load from disk
	store2, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	// Verify user was loaded
	user := store2.GetUser(testUsername)
	if user == nil {
		t.Fatal("User was not loaded from disk")
	}
	if user.Username != testUsername {
		t.Errorf("user.Username = %s; want %s", user.Username, testUsername)
	}

	// Verify authentication works
	_, ok := store2.Authenticate(testUsername, testPassword)
	if !ok {
		t.Error("Authenticate() failed after loading from disk")
	}
}

// TestUpdateUser verifies that updating a user's password works
func TestUpdateUser(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Register a new user
	err := store.Register(testUsername, testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Update password
	newPassword := "new-password456"
	err = store.Register(testUsername, newPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Verify old password no longer works
	_, ok := store.Authenticate(testUsername, testPassword)
	if ok {
		t.Error("Old password still works after update")
	}

	// Verify new password works
	_, ok = store.Authenticate(testUsername, newPassword)
	if !ok {
		t.Error("New password doesn't work after update")
	}
}

// TestConcurrentOperations verifies that concurrent operations work correctly
func TestConcurrentOperations(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Register initial user
	err := store.Register(testUsername, testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			// Alternate between reading and writing operations
			if i%2 == 0 {
				// Authenticate (read operation)
				_, ok := store.Authenticate(testUsername, testPassword)
				if !ok {
					t.Errorf("Authenticate() failed in goroutine %d", i)
				}
			} else {
				// Update password (write operation)
				tempPass := testPassword + "-" + string(rune(i+'0'))
				err := store.Register(testUsername, tempPass)
				if err != nil {
					t.Errorf("Register() error = %v in goroutine %d", err, i)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestEmptyFile verifies that an empty users file is handled correctly
func TestEmptyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "todoissh-empty-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create empty users file
	usersPath := filepath.Join(tempDir, "users.json")
	err = os.WriteFile(usersPath, []byte{}, 0600)
	if err != nil {
		t.Fatalf("Failed to create empty users file: %v", err)
	}

	// Create store with existing empty file
	store, err := NewStore(tempDir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	// Verify store is empty but initialized
	if len(store.users) != 0 {
		t.Errorf("store.users has %d entries; want 0", len(store.users))
	}
}

// TestInvalidJSON verifies that invalid JSON in the users file is handled correctly
func TestInvalidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "todoissh-invalid-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create invalid JSON file
	usersPath := filepath.Join(tempDir, "users.json")
	err = os.WriteFile(usersPath, []byte("invalid json"), 0600)
	if err != nil {
		t.Fatalf("Failed to create invalid users file: %v", err)
	}

	// Create store with existing invalid file
	_, err = NewStore(tempDir)
	if err == nil {
		t.Fatal("NewStore() did not return error with invalid JSON")
	}
}

// TestNewStoreDirectoryError verifies that an error is returned when the data directory can't be created
func TestNewStoreDirectoryError(t *testing.T) {
	// Try to create a store with an invalid directory
	// This simulates a permission error or other issue that would prevent directory creation
	_, err := NewStore("/root/nonexistent/directory") // Should fail on most systems due to permissions
	if err == nil {
		t.Fatal("NewStore() did not return error for invalid directory")
	}
}

// TestRegisterError verifies that an error in password hashing is handled correctly
func TestRegisterError(t *testing.T) {
	// This test is intentionally skipped because we can't easily mock bcrypt functions
	// The code below shows how you would test this error path if it were possible
	t.Skip("Skipping test that requires mocking bcrypt.GenerateFromPassword")

	// We would mock the function to return an error if we could
	// bcrypt.GenerateFromPassword = func(password []byte, cost int) ([]byte, error) {
	//     return nil, fmt.Errorf("simulated bcrypt error")
	// }
}

// TestSaveError verifies that an error during saving is handled correctly
func TestSaveError(t *testing.T) {
	store, tempDir := setupTestStore(t)
	defer cleanupTestStore(tempDir)

	// Register a user
	err := store.Register(testUsername, testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Make the directory read-only to force a write error
	// Note: Skip this test on Windows as permissions work differently
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping test on Windows")
	}

	// Get the path to the users.json file
	usersFile := filepath.Join(tempDir, "users.json")

	// Delete the file so we can test write failure
	err = os.Remove(usersFile)
	if err != nil {
		t.Fatalf("Failed to remove users file: %v", err)
	}

	// Make the directory read-only
	err = os.Chmod(tempDir, 0500) // r-x------
	if err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}

	// Attempt to register another user, which should fail when saving
	err = store.Register("another", "password")

	// Restore permissions so cleanup can work
	os.Chmod(tempDir, 0700)

	// Check if we got an error
	if err == nil {
		t.Fatal("save() did not return error when writing to read-only directory")
	}
}

// TestLoadError verifies that an error during loading is handled correctly
func TestLoadError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "todoissh-user-load-error")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file that's not readable
	usersFile := filepath.Join(tempDir, "users.json")
	err = os.WriteFile(usersFile, []byte(`{"testuser":{"username":"testuser","password_hash":"hash"}}`), 0200) // --w-------
	if err != nil {
		t.Fatalf("Failed to create users file: %v", err)
	}

	// Try to create a store with the existing unreadable file
	// This should fail when trying to load the file
	_, err = NewStore(tempDir)

	// Make file readable so cleanup can work
	os.Chmod(usersFile, 0600)

	// Check if we got an error
	if err == nil {
		t.Fatal("load() did not return error when reading unreadable file")
	}
}
