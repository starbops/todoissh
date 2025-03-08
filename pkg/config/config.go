package config

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

// Version information
const (
	Version = "0.1.0"
	AppName = "TodoiSSH"
)

// LogLevel defines the verbosity of logging
type LogLevel int

const (
	LogLevelNormal LogLevel = iota
	LogLevelVerbose
	LogLevelDebug
)

// Config holds the application configuration
type Config struct {
	Port     int
	HostKey  string
	ShowHelp bool
	ShowVer  bool
	LogLevel LogLevel
}

// ParseFlags parses command-line flags and updates the configuration
func ParseFlags() *Config {
	cfg := &Config{
		Port:     2222,
		HostKey:  "id_rsa",
		LogLevel: LogLevelNormal,
	}

	// Define command-line flags
	pflag.IntVarP(&cfg.Port, "port", "p", cfg.Port, "Port number for the SSH server")
	pflag.StringVar(&cfg.HostKey, "hostkey", cfg.HostKey, "Path to the host key file")

	// Help and version flags
	pflag.BoolVarP(&cfg.ShowHelp, "help", "h", false, "Show help information")
	pflag.BoolVarP(&cfg.ShowVer, "version", "V", false, "Show version information")

	// Verbosity flags
	verbose := pflag.BoolP("verbose", "v", false, "Enable verbose logging")
	debug := pflag.Bool("debug", false, "Enable debug logging (implies verbose)")

	// Parse flags
	pflag.Parse()

	// Set log level based on verbosity flags
	switch {
	case *debug:
		cfg.LogLevel = LogLevelDebug
	case *verbose:
		cfg.LogLevel = LogLevelVerbose
	default:
		cfg.LogLevel = LogLevelNormal
	}

	return cfg
}

// PrintVersion prints the version information
func PrintVersion() {
	fmt.Printf("%s v%s\n", AppName, Version)
}

// PrintHelp prints the help information
func PrintHelp() {
	fmt.Printf("Usage: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Printf("A terminal-based todo list application accessible via SSH.\n\n")
	fmt.Println("Options:")
	pflag.PrintDefaults()
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return ParseFlags()
}
