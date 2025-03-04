package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

// Server represents an SSH server instance
type Server struct {
	config   *ssh.ServerConfig
	port     int
	hostKey  string
	handler  func(ssh.Channel, <-chan *ssh.Request)
}

// NewServer creates a new SSH server instance
func NewServer(port int, hostKeyPath string) (*Server, error) {
	server := &Server{
		port:    port,
		hostKey: hostKeyPath,
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
			return nil, nil
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

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

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

		go s.handler(channel, requests)
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