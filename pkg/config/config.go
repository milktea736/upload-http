package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServerConfig holds server configuration
type ServerConfig struct {
	Port        int    `json:"port"`
	Host        string `json:"host"`
	StoragePath string `json:"storage_path"`
	MaxFileSize int64  `json:"max_file_size"` // in bytes
	LogLevel    string `json:"log_level"`
	EnableHTTPS bool   `json:"enable_https"`
	CertFile    string `json:"cert_file,omitempty"`
	KeyFile     string `json:"key_file,omitempty"`
}

// ClientConfig holds client configuration
type ClientConfig struct {
	ServerURL   string `json:"server_url"`
	Timeout     int    `json:"timeout"` // in seconds
	Concurrency int    `json:"concurrency"`
	LogLevel    string `json:"log_level"`
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:        8080,
		Host:        "0.0.0.0",
		StoragePath: "./uploads",
		MaxFileSize: 100 * 1024 * 1024, // 100MB
		LogLevel:    "info",
		EnableHTTPS: false,
	}
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ServerURL:   "http://localhost:8080",
		Timeout:     300, // 5 minutes
		Concurrency: 4,
		LogLevel:    "info",
	}
}

// LoadServerConfig loads server configuration from file
func LoadServerConfig(configPath string) (*ServerConfig, error) {
	config := DefaultServerConfig()
	
	if configPath == "" {
		return config, nil
	}
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Ensure storage path exists
	if err := os.MkdirAll(config.StoragePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	
	return config, nil
}

// LoadClientConfig loads client configuration from file
func LoadClientConfig(configPath string) (*ClientConfig, error) {
	config := DefaultClientConfig()
	
	if configPath == "" {
		return config, nil
	}
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	return config, nil
}

// SaveServerConfig saves server configuration to file
func (c *ServerConfig) Save(configPath string) error {
	// Ensure config directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// SaveClientConfig saves client configuration to file
func (c *ClientConfig) Save(configPath string) error {
	// Ensure config directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// Address returns the full address for the server
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}