package config

// Config holds the application configuration
type Config struct {
	Port     int
	HostKey  string
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		Port:    2222,
		HostKey: "id_rsa",
	}
} 