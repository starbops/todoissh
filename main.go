package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	"todoissh/todo"
)

const (
	hostKey = "id_rsa"
	port    = 2222
)

var todoStore *todo.Store

type TerminalUI struct {
	channel    ssh.Channel
	width      int
	height     int
	mutex      sync.Mutex
	todos      []*todo.Todo
	selected   int
	inputMode  bool
	inputText  string
	inputLabel string
	cursorPos  int
}

func newTerminalUI(channel ssh.Channel) *TerminalUI {
	return &TerminalUI{
		channel:    channel,
		selected:   0,
		inputMode:  false,
		inputLabel: "New todo: ",
		width:      80,
		height:    24,
		cursorPos:  0,
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

	// Header
	t.write("Todo List\r\n")
	t.write(strings.Repeat("─", t.width) + "\r\n")

	// Only show commands in input mode
	if t.inputMode {
		t.write("Commands: ←/→: Move cursor • Enter: Save • Tab: Cancel • Ctrl+C: Exit\r\n")
	} else {
		t.write("Commands: ↑/↓: Navigate • Space: Toggle • Enter: Edit • Tab: New • Delete: Remove • Ctrl+C: Exit\r\n")
	}
	t.write("\r\n")

	// Get and sort todos
	t.todos = todoStore.List()
	sort.Slice(t.todos, func(i, j int) bool {
		return t.todos[i].ID < t.todos[j].ID
	})

	// Print todos
	if len(t.todos) == 0 {
		t.write("No todos yet. Press Tab to add one.\r\n")
	} else {
		for i, todo := range t.todos {
			prefix := "  "
			if i == t.selected && !t.inputMode {
				prefix = "> "
			}
			status := "[ ]"
			if todo.Completed {
				status = "[✓]"
			}
			t.write(fmt.Sprintf("%s%s %d. %s\r\n", prefix, status, todo.ID, todo.Text))
		}
	}

	// Input field
	if t.inputMode {
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

		switch buf[0] {
		case 3: // Ctrl+C
			t.clear()
			t.showCursor()
			t.write("Goodbye!\r\n")
			return nil
		case 9: // Tab
			t.inputMode = !t.inputMode
			if t.inputMode {
				t.inputLabel = "New todo: "
				t.inputText = ""
				t.cursorPos = 0
			}
		case 13: // Enter
			if t.inputMode {
				text := strings.TrimSpace(t.inputText)
				if text != "" {
					if t.inputLabel == "New todo: " {
						todoStore.Add(text)
					} else {
						id, _ := strconv.Atoi(strings.TrimPrefix(strings.TrimSuffix(t.inputLabel, ": "), "Edit todo "))
						todoStore.Update(id, text)
					}
				}
				t.inputMode = false
				t.inputText = ""
				t.cursorPos = 0
			} else if len(t.todos) > 0 {
				t.inputMode = true
				t.inputText = t.todos[t.selected].Text
				t.inputLabel = fmt.Sprintf("Edit todo %d: ", t.todos[t.selected].ID)
				t.cursorPos = len(t.inputText)
			}
		case 127: // Backspace
			if t.inputMode && len(t.inputText) > 0 && t.cursorPos > 0 {
				t.inputText = t.inputText[:t.cursorPos-1] + t.inputText[t.cursorPos:]
				t.cursorPos--
			}
		case 32: // Space
			if !t.inputMode && len(t.todos) > 0 {
				todoStore.ToggleComplete(t.todos[t.selected].ID)
			} else if t.inputMode {
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
				if !t.inputMode && t.selected > 0 {
					t.selected--
				}
			case 66: // Down arrow
				if !t.inputMode && t.selected < len(t.todos)-1 {
					t.selected++
				}
			case 67: // Right arrow
				if t.inputMode && t.cursorPos < len(t.inputText) {
					t.cursorPos++
				}
			case 68: // Left arrow
				if t.inputMode && t.cursorPos > 0 {
					t.cursorPos--
				}
			case 51: // Delete key
				if !t.inputMode && len(t.todos) > 0 {
					t.channel.Read(make([]byte, 1))
					todoStore.Delete(t.todos[t.selected].ID)
					if t.selected >= len(t.todos)-1 {
						t.selected = len(t.todos) - 2
						if t.selected < 0 {
							t.selected = 0
						}
					}
				} else if t.inputMode && t.cursorPos < len(t.inputText) {
					t.inputText = t.inputText[:t.cursorPos] + t.inputText[t.cursorPos+1:]
				}
			}
		default:
			if t.inputMode && buf[0] >= 32 && buf[0] < 127 {
				t.inputText = t.inputText[:t.cursorPos] + string(buf[0]) + t.inputText[t.cursorPos:]
				t.cursorPos++
			}
		}

		t.refreshDisplay()
	}
}

func main() {
	// Initialize todo store
	todoStore = todo.NewStore()

	// Generate the server's private key
	if _, err := os.Stat(hostKey); os.IsNotExist(err) {
		privateKey, err := generateHostKey()
		if err != nil {
			log.Fatalf("Failed to generate host key: %v", err)
		}
		log.Printf("Generated new host key: %s", hostKey)
		if err := os.WriteFile(hostKey, privateKey, 0600); err != nil {
			log.Fatalf("Failed to write host key: %v", err)
		}
	}

	// Read the host key
	privateBytes, err := os.ReadFile(hostKey)
	if err != nil {
		log.Fatalf("Failed to read host key: %v", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// For this example, allow any username/password
			return nil, nil
		},
	}
	config.AddHostKey(private)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", port, err)
	}
	log.Printf("Listening on port %d...", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(conn, config)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to establish SSH connection: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Failed to accept channel: %v", err)
			continue
		}

		go handleChannel(channel, requests)
	}
}

func handleChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	termUI := newTerminalUI(channel)

	// Initialize terminal
	termUI.write("\x1b[?1049h") // Use alternate screen buffer
	termUI.write("\x1b[?7l")    // Disable line wrapping
	defer func() {
		termUI.write("\x1b[?25h")   // Show cursor
		termUI.write("\x1b[?7h")    // Enable line wrapping
		termUI.write("\x1b[?1049l") // Restore main screen
		termUI.write("Goodbye!\r\n") // Always show goodbye message
		channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0}) // Send exit code 0
	}()

	for req := range requests {
		switch req.Type {
		case "shell":
			if len(req.Payload) > 0 {
				req.Reply(false, nil)
				continue
			}
			req.Reply(true, nil)
			termUI.refreshDisplay()
			if err := termUI.handleInput(); err != nil {
				if err != io.EOF {
					log.Printf("Error handling input: %v", err)
					channel.SendRequest("exit-status", false, []byte{0, 0, 0, 1}) // Send exit code 1 for errors
				}
			}
			return
		case "pty-req":
			termWidth, termHeight := parsePtyRequest(req.Payload)
			termUI.setSize(termWidth, termHeight)
			req.Reply(true, nil)
		case "window-change":
			termWidth, termHeight := parseWinchRequest(req.Payload)
			termUI.setSize(termWidth, termHeight)
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

func parsePtyRequest(payload []byte) (width, height int) {
	width = int(payload[10]) + int(payload[11])<<8
	height = int(payload[12]) + int(payload[13])<<8
	return
}

func parseWinchRequest(payload []byte) (width, height int) {
	width = int(payload[2]) + int(payload[3])<<8
	height = int(payload[0]) + int(payload[1])<<8
	return
}

func generateHostKey() ([]byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	return pem.EncodeToMemory(privateKeyPEM), nil
} 