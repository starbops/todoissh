package main

import (
	"log"

	"golang.org/x/crypto/ssh"
	"todoissh/pkg/config"
	sshpkg "todoissh/pkg/ssh"
	"todoissh/pkg/todo"
	"todoissh/pkg/ui"
)

func main() {
	// Load configuration
	cfg := config.NewConfig()

	// Initialize todo store
	todoStore := todo.NewStore()

	// Create and start SSH server
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
} 