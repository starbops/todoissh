package main

import (
	"log"
	"os"

	"todoissh/pkg/config"
	sshpkg "todoissh/pkg/ssh"
	"todoissh/pkg/todo"
	"todoissh/pkg/ui"

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

	// Initialize todo store
	todoStore := todo.NewStore()

	// Create and start SSH server
	log.Printf("Starting server on port %d...", cfg.Port)
	server, err := sshpkg.NewServer(cfg.Port, cfg.HostKey)
	if err != nil {
		log.Fatalf("Failed to create SSH server: %v", err)
	}

	// Set channel handler
	server.SetChannelHandler(func(channel ssh.Channel, requests <-chan *ssh.Request) {
		termUI := ui.NewTerminalUI(channel, todoStore)
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
