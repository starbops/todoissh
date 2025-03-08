package integration

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"todoissh/pkg/config"
	"todoissh/pkg/ssh"
	"todoissh/pkg/todo"

	gossh "golang.org/x/crypto/ssh"
)

const (
	testPort    = 2223 // Use a different port for testing
	testHostKey = "test_host_key"
)

type testServer struct {
	server    *ssh.Server
	todoStore *todo.Store
}

func setupTestServer(t *testing.T) (*testServer, error) {
	// Create test config
	cfg := &config.Config{
		Port:    testPort,
		HostKey: testHostKey,
	}

	// Initialize todo store
	todoStore := todo.NewStore()

	// Create and start test server
	server, err := ssh.NewServer(cfg.Port, cfg.HostKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create test server: %v", err)
	}

	// Set up a basic channel handler for tests
	server.SetChannelHandler(func(ch gossh.Channel, reqs <-chan *gossh.Request) {
		// Handle requests
		go func() {
			for req := range reqs {
				switch req.Type {
				case "pty-req":
					// Accept PTY requests
					req.Reply(true, nil)
				case "shell":
					// Accept shell requests
					req.Reply(true, nil)
					// Close the channel after accepting shell to simulate exit
					go func() {
						time.Sleep(100 * time.Millisecond)
						// Send exit status before closing
						_, err := ch.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
						if err != nil {
							t.Logf("Failed to send exit status: %v", err)
						}
						ch.Close()
					}()
				case "window-change":
					// Accept window changes
					req.Reply(true, nil)
				default:
					// Reject other requests
					req.Reply(false, nil)
				}
			}
		}()

		// Keep the channel open until closed
		io.Copy(io.Discard, ch)
	})

	go func() {
		if err := server.Start(); err != nil {
			t.Errorf("server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	return &testServer{server: server, todoStore: todoStore}, nil
}

func (ts *testServer) Close() error {
	if ts.server != nil {
		return ts.server.Close()
	}
	return nil
}

func createTestClient() (*gossh.Client, error) {
	config := &gossh.ClientConfig{
		User: "test",
		Auth: []gossh.AuthMethod{
			gossh.Password("test"),
		},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}

	return gossh.Dial("tcp", fmt.Sprintf("localhost:%d", testPort), config)
}

func TestServerConnection(t *testing.T) {
	ts, err := setupTestServer(t)
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer ts.Close()
	defer os.Remove(testHostKey)

	// Test 1: Verify SSH connection works
	t.Run("SSH Connection", func(t *testing.T) {
		client, err := createTestClient()
		if err != nil {
			t.Fatalf("Failed to connect to server via SSH: %v", err)
		}
		defer client.Close()
	})

	// Test 2: Verify server is listening on TCP port
	t.Run("TCP Port", func(t *testing.T) {
		// Just verify the port is open
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", testPort), 100*time.Millisecond)
		if err != nil {
			t.Fatalf("Server not listening on port %d: %v", testPort, err)
		}
		conn.Close() // Close immediately as we only care that the port is open
	})

	// Test 3: Verify non-SSH connections are rejected
	t.Run("Reject non-SSH", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", testPort))
		if err != nil {
			t.Fatalf("Failed to establish TCP connection: %v", err)
		}
		defer conn.Close()

		// Read the initial SSH banner
		initialBuf := make([]byte, 1024)
		n, err := conn.Read(initialBuf)
		if err != nil {
			t.Fatalf("Failed to read initial SSH banner: %v", err)
		}
		t.Logf("Read initial SSH banner (%d bytes): %s", n, initialBuf[:n])

		// Write some non-SSH data
		_, err = conn.Write([]byte("NOT AN SSH CONNECTION\n"))
		if err != nil {
			// If write fails, that's fine - server might have rejected immediately
			t.Logf("Write failed (connection rejected): %v", err)
			return
		}

		// Now try to read - this should fail or return 0 bytes as the server should reject the connection
		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		responseBuffer := make([]byte, 1024)
		n, err = conn.Read(responseBuffer)

		if err == nil && n > 0 {
			t.Errorf("Expected connection to be rejected, but read %d bytes: %s", n, responseBuffer[:n])
		} else {
			// Either we got an error or read 0 bytes, both are fine
			t.Logf("Connection properly rejected: err=%v, bytes=%d", err, n)
		}
	})
}

// isConnectionResetError checks if the error is a connection reset
func isConnectionResetError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection reset by peer") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection refused")
}

func TestTodoOperations(t *testing.T) {
	ts, err := setupTestServer(t)
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer ts.Close()
	defer os.Remove(testHostKey)

	// Add a test todo item
	todo, err := ts.todoStore.Add("Test todo item")
	if err != nil {
		t.Fatalf("Failed to add todo item: %v", err)
	}
	if todo == nil {
		t.Fatal("Todo item is nil")
	}

	// Verify the todo was added
	todos := ts.todoStore.List()
	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo item, got %d", len(todos))
	}
	if todos[0].Text != "Test todo item" {
		t.Fatalf("Expected todo text 'Test todo item', got '%s'", todos[0].Text)
	}

	client, err := createTestClient()
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer client.Close()

	// Create session
	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer session.Close()

	// Request pseudo-terminal
	err = session.RequestPty("xterm", 80, 40, gossh.TerminalModes{})
	if err != nil {
		t.Fatalf("Failed to request pty: %v", err)
	}

	// Start shell
	err = session.Shell()
	if err != nil {
		t.Fatalf("Failed to start shell: %v", err)
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Session error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out")
	}
}

func TestConcurrentConnections(t *testing.T) {
	ts, err := setupTestServer(t)
	if err != nil {
		t.Fatalf("Failed to setup test server: %v", err)
	}
	defer ts.Close()
	defer os.Remove(testHostKey)

	numClients := 5
	var wg sync.WaitGroup
	errChan := make(chan error, numClients)

	// Start all clients
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientNum int) {
			defer wg.Done()

			client, err := createTestClient()
			if err != nil {
				errChan <- fmt.Errorf("client %d failed to connect: %v", clientNum, err)
				return
			}
			defer client.Close()

			session, err := client.NewSession()
			if err != nil {
				errChan <- fmt.Errorf("client %d failed to create session: %v", clientNum, err)
				return
			}
			defer session.Close()

			// Request pseudo-terminal to simulate real usage
			err = session.RequestPty("xterm", 80, 40, gossh.TerminalModes{})
			if err != nil {
				errChan <- fmt.Errorf("client %d failed to request pty: %v", clientNum, err)
				return
			}

			// Start shell
			err = session.Shell()
			if err != nil {
				errChan <- fmt.Errorf("client %d failed to start shell: %v", clientNum, err)
				return
			}

			// Wait with timeout
			done := make(chan error, 1)
			go func() {
				done <- session.Wait()
			}()

			select {
			case err := <-done:
				if err != nil {
					errChan <- fmt.Errorf("client %d session error: %v", clientNum, err)
				} else {
					errChan <- nil
				}
			case <-time.After(2 * time.Second):
				errChan <- fmt.Errorf("client %d timed out", clientNum)
			}
		}(i)
	}

	// Wait for all clients to finish
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		if err != nil {
			t.Errorf("Concurrent connection test failed: %v", err)
		}
	}
}
