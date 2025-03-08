package ssh

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Server represents an SSH server instance
type Server struct {
	config   *ssh.ServerConfig
	port     int
	hostKey  string
	handler  func(ssh.Channel, <-chan *ssh.Request)
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	conns    map[net.Conn]struct{}
}

// NewServer creates a new SSH server instance
func NewServer(port int, hostKeyPath string) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		port:    port,
		hostKey: hostKeyPath,
		ctx:     ctx,
		cancel:  cancel,
		conns:   make(map[net.Conn]struct{}),
	}

	// Generate the server's private key if it doesn't exist
	if _, err := os.Stat(hostKeyPath); os.IsNotExist(err) {
		privateKey, err := generateHostKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate host key: %v", err)
		}
		log.Printf("Generated new host key: %s", hostKeyPath)
		if err := os.WriteFile(hostKeyPath, privateKey, 0600); err != nil {
			return nil, fmt.Errorf("failed to write host key: %v", err)
		}
	}

	// Read the host key
	privateBytes, err := os.ReadFile(hostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read host key: %v", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// For this example, allow any username/password
			return &ssh.Permissions{}, nil
		},
	}
	config.AddHostKey(private)
	server.config = config

	return server, nil
}

// SetChannelHandler sets the handler for new SSH channels
func (s *Server) SetChannelHandler(handler func(ssh.Channel, <-chan *ssh.Request)) {
	s.handler = handler
}

// Start starts the SSH server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", s.port, err)
	}
	log.Printf("Listening on port %d...", s.port)

	s.listener = listener

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-s.ctx.Done():
					return
				default:
					log.Printf("Failed to accept connection: %v", err)
					continue
				}
			}
			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}()

	return nil
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	// Track connection
	s.mu.Lock()
	s.conns[conn] = struct{}{}
	s.mu.Unlock()

	// Cleanup connection tracking on exit
	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
	}()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
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

		if s.handler != nil {
			go s.handler(channel, requests)
		} else {
			channel.Close()
		}
	}
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

// Close shuts down the SSH server and cleans up resources
func (s *Server) Close() error {
	s.cancel() // Signal shutdown

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Close all active connections
	s.mu.Lock()
	for conn := range s.conns {
		conn.Close()
	}
	s.mu.Unlock()

	// Wait for all goroutines to finish
	s.wg.Wait()

	return nil
}
