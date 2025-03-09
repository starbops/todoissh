package main

import (
	"log"
	"os"
	"path/filepath"

	"todoissh/pkg/config"
	sshpkg "todoissh/pkg/ssh"
	"todoissh/pkg/todo"
	"todoissh/pkg/ui"
	"todoissh/pkg/user"

	"golang.org/x/crypto/ssh"
)

func main() {
	// Load configuration with command-line flags
	cfg := config.NewConfig()

	// Handle special flags
	if cfg.ShowVer {
		config.PrintVersion()
		return
	}

	if cfg.ShowHelp {
		config.PrintHelp()
		return
	}

	// Configure logging based on verbosity level
	setupLogging(cfg.LogLevel)

	// Use DATA_DIR environment variable if set, otherwise use default "data"
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	log.Printf("Using data directory: %s", dataDir)

	// Create data directory
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Set the host key path to be in the data directory
	hostKeyPath := filepath.Join(dataDir, "id_rsa")
	if cfg.HostKey == "id_rsa" {
		// Only override if it's the default value
		cfg.HostKey = hostKeyPath
	}

	// Initialize user store
	userStore, err := user.NewStore(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize user store: %v", err)
	}

	// Initialize todo store
	todoStore, err := todo.NewStore(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize todo store: %v", err)
	}

	// Create and start SSH server
	log.Printf("Starting server on port %d...", cfg.Port)
	server, err := sshpkg.NewServer(cfg.Port, cfg.HostKey, userStore)
	if err != nil {
		log.Fatalf("Failed to create SSH server: %v", err)
	}

	// Set channel handler
	server.SetChannelHandler(func(username string, channel ssh.Channel, requests <-chan *ssh.Request) {
		// Check if this is a new user
		isNewUser := userStore.GetUser(username) == nil

		// Create terminal UI with user information
		termUI := ui.NewTerminalUI(channel, todoStore, userStore, username, isNewUser)
		termUI.HandleChannel(requests)
	})

	// Start server
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	// Keep the main function running
	log.Printf("Server running on port %d. Press Ctrl+C to exit...", cfg.Port)
	select {} // Block forever
}

// setupLogging configures the logging based on the verbosity level
func setupLogging(level config.LogLevel) {
	// Default logger settings
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags)

	switch level {
	case config.LogLevelNormal:
		// For normal mode, use minimal logging
		log.SetFlags(log.LstdFlags)
	case config.LogLevelVerbose:
		// For verbose mode, include file and line numbers
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Verbose logging enabled")
	case config.LogLevelDebug:
		// For debug mode, include all details
		log.SetFlags(log.LstdFlags | log.Llongfile | log.Lmicroseconds)
		log.Println("Debug logging enabled")
	}
}
