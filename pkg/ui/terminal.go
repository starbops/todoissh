package ui

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"sync"

	"todoissh/pkg/todo"
	"todoissh/pkg/user"

	"golang.org/x/crypto/ssh"
)

// UIMode represents the current UI mode
type UIMode int

const (
	ModeNormal UIMode = iota
	ModeInput
	ModeRegister
)

// TerminalUI represents a terminal user interface
type TerminalUI struct {
	channel       ssh.Channel
	width         int
	height        int
	mutex         sync.Mutex
	todos         []*todo.Todo
	selected      int
	mode          UIMode
	inputText     string
	inputLabel    string
	cursorPos     int
	todoStore     *todo.Store
	userStore     *user.Store
	username      string
	isRegistering bool
	registerStep  int
	password      string
}

// NewTerminalUI creates a new terminal UI instance
func NewTerminalUI(channel ssh.Channel, todoStore *todo.Store, userStore *user.Store, username string, isNewUser bool) *TerminalUI {
	ui := &TerminalUI{
		channel:       channel,
		selected:      0,
		mode:          ModeNormal,
		inputLabel:    "New todo: ",
		width:         80,
		height:        24,
		cursorPos:     0,
		todoStore:     todoStore,
		userStore:     userStore,
		username:      username,
		isRegistering: isNewUser,
		registerStep:  0,
	}

	// If this is a new user, start in registration mode
	if isNewUser {
		ui.mode = ModeRegister
	}

	return ui
}

// HandleChannel handles the SSH channel and requests
func (t *TerminalUI) HandleChannel(requests <-chan *ssh.Request) {
	defer t.channel.Close()

	// Initialize terminal
	t.write("\x1b[?1049h") // Use alternate screen buffer
	t.write("\x1b[?7l")    // Disable line wrapping
	defer func() {
		t.write("\x1b[?25h")                                            // Show cursor
		t.write("\x1b[?7h")                                             // Enable line wrapping
		t.write("\x1b[?1049l")                                          // Restore main screen
		t.write("Goodbye!\r\n")                                         // Always show goodbye message
		t.channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0}) // Send exit code 0
	}()

	for req := range requests {
		switch req.Type {
		case "shell":
			if len(req.Payload) > 0 {
				req.Reply(false, nil)
				continue
			}
			req.Reply(true, nil)
			t.refreshDisplay()
			if err := t.handleInput(); err != nil {
				if err != io.EOF {
					log.Printf("Error handling input: %v", err)
					t.channel.SendRequest("exit-status", false, []byte{0, 0, 0, 1}) // Send exit code 1 for errors
				}
			}
			return
		case "pty-req":
			width, height := parsePtyRequest(req.Payload)
			t.setSize(width, height)
			req.Reply(true, nil)
		case "window-change":
			width, height := parseWinchRequest(req.Payload)
			t.setSize(width, height)
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

func (t *TerminalUI) setSize(width, height int) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.width = width
	t.height = height
}

func (t *TerminalUI) write(text string) {
	t.channel.Write([]byte(text))
}

func (t *TerminalUI) clear() {
	t.write("\x1b[2J")   // Clear screen
	t.write("\x1b[H")    // Move cursor to home
	t.write("\x1b[?25l") // Hide cursor
}

func (t *TerminalUI) showCursor() {
	t.write("\x1b[?25h")
}

func (t *TerminalUI) hideCursor() {
	t.write("\x1b[?25l")
}

func (t *TerminalUI) moveTo(row, col int) {
	t.write(fmt.Sprintf("\x1b[%d;%dH", row, col))
}

func (t *TerminalUI) refreshDisplay() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.clear()
	t.moveTo(1, 1)

	if t.mode == ModeRegister {
		t.displayRegistrationScreen()
		return
	}

	// Header
	t.write(fmt.Sprintf("Todo List - User: %s\r\n", t.username))
	t.write(strings.Repeat("─", t.width) + "\r\n")

	// Only show commands in input mode
	if t.mode == ModeInput {
		t.write("Commands: ←/→: Move cursor • Enter: Save • Tab: Cancel • Ctrl+C: Exit\r\n")
	} else {
		t.write("Commands: ↑/↓: Navigate • Space: Toggle • Enter: Edit • Tab: New • Delete: Remove • Ctrl+C: Exit\r\n")
	}
	t.write("\r\n")

	// Get and sort todos
	todos, err := t.todoStore.List(t.username)
	if err != nil {
		t.write(fmt.Sprintf("Error loading todos: %v\r\n", err))
		return
	}
	t.todos = todos
	sort.Slice(t.todos, func(i, j int) bool {
		return t.todos[i].ID < t.todos[j].ID
	})

	// Print todos
	if len(t.todos) == 0 {
		t.write("No todos yet. Press Tab to add one.\r\n")
	} else {
		for i, todo := range t.todos {
			prefix := "  "
			if i == t.selected && t.mode == ModeNormal {
				prefix = "> "
			}
			status := "[ ]"
			if todo.Completed {
				status = "[✓]"
			}
			t.write(fmt.Sprintf("%s%s %d. %s\r\n", prefix, status, i+1, todo.Text))
		}
	}

	// Input field
	if t.mode == ModeInput {
		t.moveTo(t.height-2, 1)
		t.write(strings.Repeat("─", t.width) + "\r\n")
		t.moveTo(t.height-1, 1)
		t.write(fmt.Sprintf("%s%s", t.inputLabel, t.inputText))
		t.showCursor()
		t.moveTo(t.height-1, len(t.inputLabel)+t.cursorPos+1)
	} else {
		t.hideCursor()
	}
}

func (t *TerminalUI) displayRegistrationScreen() {
	// Registration header
	t.write("Welcome to TodoiSSH!\r\n")
	t.write(strings.Repeat("─", t.width) + "\r\n\r\n")

	t.write(fmt.Sprintf("Hello, %s! You need to complete registration.\r\n\r\n", t.username))

	switch t.registerStep {
	case 0: // Set password
		t.write("Please set a password for your account.\r\n")
		t.write("Password must be at least 6 characters long.\r\n\r\n")
		t.write("Password: ")
		if len(t.inputText) > 0 {
			t.write(strings.Repeat("*", len(t.inputText)))
		}
		t.showCursor()
		t.moveTo(9, 10+len(t.inputText)) // Position cursor after password
	case 1: // Confirm password
		t.write("Please set a password for your account.\r\n")
		t.write("Password: " + strings.Repeat("*", len(t.password)) + "\r\n\r\n")
		t.write("Confirm password: ")
		if len(t.inputText) > 0 {
			t.write(strings.Repeat("*", len(t.inputText)))
		}
		t.showCursor()
		t.moveTo(10, 18+len(t.inputText)) // Position cursor after confirm password
	}
}

func (t *TerminalUI) handleRegistration() bool {
	switch t.registerStep {
	case 0: // Set password
		if len(t.inputText) < 6 {
			t.clear()
			t.moveTo(1, 1)
			t.write("Password must be at least 6 characters long. Press any key to continue.\r\n")
			var buf [1]byte
			t.channel.Read(buf[:])
			t.inputText = ""
			return false
		}
		t.password = t.inputText
		t.inputText = ""
		t.registerStep = 1
		return false
	case 1: // Confirm password
		if t.inputText != t.password {
			t.clear()
			t.moveTo(1, 1)
			t.write("Passwords do not match. Press any key to start over.\r\n")
			var buf [1]byte
			t.channel.Read(buf[:])
			t.inputText = ""
			t.registerStep = 0
			return false
		}

		// Register the user
		err := t.userStore.Register(t.username, t.password)
		if err != nil {
			t.clear()
			t.moveTo(1, 1)
			t.write(fmt.Sprintf("Registration failed: %v. Press any key to exit.\r\n", err))
			var buf [1]byte
			t.channel.Read(buf[:])
			return true // Exit
		}

		// Registration successful
		t.clear()
		t.moveTo(1, 1)
		t.write("Registration successful! Press any key to continue.\r\n")
		var buf [1]byte
		t.channel.Read(buf[:])
		t.mode = ModeNormal
		t.isRegistering = false
		return false
	}
	return false
}

func (t *TerminalUI) handleInput() error {
	var buf [1]byte
	for {
		n, err := t.channel.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				t.clear()
				t.showCursor()
				t.write("Goodbye!\r\n")
				return nil
			}
			return fmt.Errorf("read error: %v", err)
		}

		if n == 0 {
			continue
		}

		// Handle registration mode
		if t.mode == ModeRegister {
			switch buf[0] {
			case 3: // Ctrl+C
				t.clear()
				t.showCursor()
				t.write("Registration cancelled. Goodbye!\r\n")
				return nil
			case 13: // Enter
				if t.handleRegistration() {
					return nil // Exit if registration failed
				}
				t.refreshDisplay()
				continue
			case 127: // Backspace
				if len(t.inputText) > 0 {
					t.inputText = t.inputText[:len(t.inputText)-1]
				}
				t.refreshDisplay()
				continue
			default:
				// Only allow printable ASCII characters for password
				if buf[0] >= 32 && buf[0] <= 126 {
					t.inputText += string(buf[0])
				}
				t.refreshDisplay()
				continue
			}
		}

		switch buf[0] {
		case 3: // Ctrl+C
			t.clear()
			t.showCursor()
			t.write("Goodbye!\r\n")
			return nil
		case 9: // Tab
			if t.mode == ModeNormal {
				t.mode = ModeInput
				t.inputLabel = "New todo: "
				t.inputText = ""
				t.cursorPos = 0
			} else {
				t.mode = ModeNormal
				t.inputText = ""
				t.cursorPos = 0
			}
		case 13: // Enter
			if t.mode == ModeInput {
				text := strings.TrimSpace(t.inputText)
				if text != "" {
					if t.inputLabel == "New todo: " {
						_, err := t.todoStore.Add(t.username, text)
						if err != nil {
							log.Printf("Error adding todo: %v", err)
						}
					} else {
						// Extract the actual todo ID from the selected todo
						id := t.todos[t.selected].ID
						_, err := t.todoStore.Update(t.username, id, text)
						if err != nil {
							log.Printf("Error updating todo: %v", err)
						}
					}
				}
				t.mode = ModeNormal
				t.inputText = ""
				t.cursorPos = 0
			} else if len(t.todos) > 0 {
				t.mode = ModeInput
				t.inputText = t.todos[t.selected].Text
				// Just show "Edit todo:" instead of showing the ID
				t.inputLabel = "Edit todo: "
				t.cursorPos = len(t.inputText)
			}
		case 127: // Backspace
			if t.mode == ModeInput && len(t.inputText) > 0 && t.cursorPos > 0 {
				t.inputText = t.inputText[:t.cursorPos-1] + t.inputText[t.cursorPos:]
				t.cursorPos--
			}
		case 32: // Space
			if t.mode == ModeNormal && len(t.todos) > 0 {
				// Use the actual ID from the selected todo
				_, err := t.todoStore.ToggleComplete(t.username, t.todos[t.selected].ID)
				if err != nil {
					log.Printf("Error toggling todo: %v", err)
				}
			} else if t.mode == ModeInput {
				t.inputText = t.inputText[:t.cursorPos] + " " + t.inputText[t.cursorPos:]
				t.cursorPos++
			}
		case 27: // Escape sequence
			seq := make([]byte, 2)
			if _, err := t.channel.Read(seq); err != nil {
				continue
			}
			if seq[0] != 91 { // Not a '[' character
				continue
			}
			switch seq[1] {
			case 65: // Up arrow
				if t.mode == ModeNormal && t.selected > 0 {
					t.selected--
				}
			case 66: // Down arrow
				if t.mode == ModeNormal && t.selected < len(t.todos)-1 {
					t.selected++
				}
			case 67: // Right arrow
				if t.mode == ModeInput && t.cursorPos < len(t.inputText) {
					t.cursorPos++
				}
			case 68: // Left arrow
				if t.mode == ModeInput && t.cursorPos > 0 {
					t.cursorPos--
				}
			case 51: // Delete key (starts with 27, 91, 51)
				extraByte := make([]byte, 1)
				if _, err := t.channel.Read(extraByte); err != nil {
					continue
				}
				if extraByte[0] != 126 { // Not a '~' character
					continue
				}
				if t.mode == ModeNormal && len(t.todos) > 0 {
					// Use the actual ID from the selected todo
					if err := t.todoStore.Delete(t.username, t.todos[t.selected].ID); err != nil {
						log.Printf("Error deleting todo: %v", err)
					}
					if t.selected >= len(t.todos)-1 {
						t.selected = max(0, len(t.todos)-2)
					}
				} else if t.mode == ModeInput && t.cursorPos < len(t.inputText) {
					t.inputText = t.inputText[:t.cursorPos] + t.inputText[t.cursorPos+1:]
				}
			}
		default:
			// Only handle printable ASCII characters in input mode
			if t.mode == ModeInput && buf[0] >= 32 && buf[0] <= 126 {
				t.inputText = t.inputText[:t.cursorPos] + string(buf[0]) + t.inputText[t.cursorPos:]
				t.cursorPos++
			}
		}

		t.refreshDisplay()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func parsePtyRequest(payload []byte) (width, height int) {
	// Simplified parsing for example
	if len(payload) >= 8 {
		return int(payload[2])<<8 | int(payload[3]), int(payload[4])<<8 | int(payload[5])
	}
	return 80, 24
}

func parseWinchRequest(payload []byte) (width, height int) {
	// Simplified parsing for example
	if len(payload) >= 8 {
		return int(payload[0])<<24 | int(payload[1])<<16 | int(payload[2])<<8 | int(payload[3]),
			int(payload[4])<<24 | int(payload[5])<<16 | int(payload[6])<<8 | int(payload[7])
	}
	return 80, 24
}
