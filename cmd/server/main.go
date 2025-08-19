package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/milktea736/upload-http/pkg/config"
	"github.com/milktea736/upload-http/pkg/server"
)

func main() {
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		port       = flag.Int("port", 8080, "Server port")
		host       = flag.String("host", "0.0.0.0", "Server host")
		storage    = flag.String("storage", "./uploads", "Storage directory path")
		logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		maxSize    = flag.Int64("max-size", 100*1024*1024, "Maximum file size in bytes")
		genConfig  = flag.String("generate-config", "", "Generate configuration file and exit")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		fmt.Println("upload-http server v1.0.0")
		fmt.Println("Built with Go")
		os.Exit(0)
	}

	// Generate config file if requested
	if *genConfig != "" {
		cfg := config.DefaultServerConfig()
		if err := cfg.Save(*genConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate config file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration file generated: %s\n", *genConfig)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadServerConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override config with command line flags
	if *port != 8080 {
		cfg.Port = *port
	}
	if *host != "0.0.0.0" {
		cfg.Host = *host
	}
	if *storage != "./uploads" {
		cfg.StoragePath = *storage
	}
	if *logLevel != "info" {
		cfg.LogLevel = *logLevel
	}
	if *maxSize != 100*1024*1024 {
		cfg.MaxFileSize = *maxSize
	}

	// Create and start server
	srv := server.NewServer(cfg)

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down server...")
		os.Exit(0)
	}()

	// Start server
	fmt.Printf("Starting upload-http server...\n")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Host: %s\n", cfg.Host)
	fmt.Printf("  Port: %d\n", cfg.Port)
	fmt.Printf("  Storage: %s\n", cfg.StoragePath)
	fmt.Printf("  Max File Size: %d bytes\n", cfg.MaxFileSize)
	fmt.Printf("  Log Level: %s\n", cfg.LogLevel)
	fmt.Printf("  HTTPS: %v\n", cfg.EnableHTTPS)
	fmt.Printf("\nServer URL: http://%s:%d\n", cfg.Host, cfg.Port)
	fmt.Printf("Health check: http://%s:%d/health\n", cfg.Host, cfg.Port)
	fmt.Printf("\nPress Ctrl+C to stop the server\n\n")

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server failed to start: %v\n", err)
		os.Exit(1)
	}
}