package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/milktea736/upload-http/pkg/client"
	"github.com/milktea736/upload-http/pkg/config"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "upload":
		handleUpload()
	case "download":
		handleDownload()
	case "list":
		handleList()
	case "health":
		handleHealth()
	case "config":
		handleConfig()
	case "version":
		handleVersion()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleUpload() {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	var (
		configFile = fs.String("config", "", "Path to configuration file")
		serverURL  = fs.String("server", "http://localhost:8080", "Server URL")
		verbose    = fs.Bool("verbose", false, "Verbose output")
		logLevel   = fs.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s upload [options] <local-path> [remote-path]\n", os.Args[0])
		os.Exit(1)
	}

	localPath := fs.Arg(0)
	remotePath := fs.Arg(1)
	if remotePath == "" {
		remotePath = filepath.Base(localPath)
	}

	// Load configuration
	cfg, err := config.LoadClientConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override with command line flags
	if *serverURL != "http://localhost:8080" {
		cfg.ServerURL = *serverURL
	}
	if *logLevel != "info" {
		cfg.LogLevel = *logLevel
	}
	if *verbose {
		cfg.LogLevel = "debug"
	}

	// Create client
	c := client.NewClient(cfg)

	// Check if local path exists
	info, err := os.Stat(localPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Local path does not exist: %s\n", localPath)
		os.Exit(1)
	}

	fmt.Printf("Uploading %s to %s...\n", localPath, remotePath)

	var uploadErr error
	if info.IsDir() {
		// Upload folder
		uploadErr = c.UploadFolder(localPath, remotePath, func(progress *client.TransferProgress) {
			if *verbose {
				fmt.Printf("Progress: %d/%d files, %s\n",
					progress.ProcessedFiles, progress.TotalFiles, progress.CurrentFile)
			} else {
				percentage := float64(progress.ProcessedFiles) / float64(progress.TotalFiles) * 100
				fmt.Printf("\rProgress: %.1f%% (%d/%d files)", percentage, progress.ProcessedFiles, progress.TotalFiles)
			}
		})
	} else {
		// Upload single file - use the client's UploadFile method directly
		if err := c.UploadFile(localPath, remotePath); err != nil {
			uploadErr = fmt.Errorf("failed to upload file: %w", err)
		} else {
			fmt.Printf("\rFile uploaded successfully")
		}
	}

	if uploadErr != nil {
		fmt.Fprintf(os.Stderr, "\nUpload failed: %v\n", uploadErr)
		os.Exit(1)
	}

	fmt.Printf("\nUpload completed successfully!\n")
}

func handleDownload() {
	fs := flag.NewFlagSet("download", flag.ExitOnError)
	var (
		configFile = fs.String("config", "", "Path to configuration file")
		serverURL  = fs.String("server", "http://localhost:8080", "Server URL")
		verbose    = fs.Bool("verbose", false, "Verbose output")
		logLevel   = fs.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s download [options] <remote-path> [local-path]\n", os.Args[0])
		os.Exit(1)
	}

	remotePath := fs.Arg(0)
	localPath := fs.Arg(1)
	if localPath == "" {
		localPath = filepath.Base(remotePath)
	}

	// Load configuration
	cfg, err := config.LoadClientConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override with command line flags
	if *serverURL != "http://localhost:8080" {
		cfg.ServerURL = *serverURL
	}
	if *logLevel != "info" {
		cfg.LogLevel = *logLevel
	}
	if *verbose {
		cfg.LogLevel = "debug"
	}

	// Create client
	c := client.NewClient(cfg)

	fmt.Printf("Downloading %s to %s...\n", remotePath, localPath)

	// First, check if remote path is a file or directory by listing its parent
	parentPath := filepath.Dir(remotePath)
	fileName := filepath.Base(remotePath)
	
	files, err := c.ListFiles(parentPath)
	if err != nil {
		// If we can't list the parent, try direct download as file
		downloadErr := c.DownloadFile(remotePath, localPath)
		if downloadErr != nil {
			fmt.Fprintf(os.Stderr, "\nDownload failed: %v\n", downloadErr)
			os.Exit(1)
		}
	} else {
		// Check if the target is a file or directory
		var isDirectory bool
		for _, file := range files {
			if file.Name == fileName {
				isDirectory = file.IsDir
				break
			}
		}
		
		var downloadErr error
		if isDirectory {
			// Download as folder
			downloadErr = c.DownloadFolder(remotePath, localPath, func(progress *client.TransferProgress) {
				if *verbose {
					fmt.Printf("Progress: %d files, current: %s\n", progress.ProcessedFiles, progress.CurrentFile)
				} else {
					fmt.Printf("\rProgress: %d files processed", progress.ProcessedFiles)
				}
			})
		} else {
			// Download as file
			downloadErr = c.DownloadFile(remotePath, localPath)
		}
		
		if downloadErr != nil {
			fmt.Fprintf(os.Stderr, "\nDownload failed: %v\n", downloadErr)
			os.Exit(1)
		}
	}

	fmt.Printf("\nDownload completed successfully!\n")
}

func handleList() {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	var (
		configFile = fs.String("config", "", "Path to configuration file")
		serverURL  = fs.String("server", "http://localhost:8080", "Server URL")
		logLevel   = fs.String("log-level", "info", "Log level (debug, info, warn, error)")
	)
	fs.Parse(os.Args[2:])

	remotePath := "."
	if fs.NArg() > 0 {
		remotePath = fs.Arg(0)
	}

	// Load configuration
	cfg, err := config.LoadClientConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override with command line flags
	if *serverURL != "http://localhost:8080" {
		cfg.ServerURL = *serverURL
	}
	if *logLevel != "info" {
		cfg.LogLevel = *logLevel
	}

	// Create client
	c := client.NewClient(cfg)

	files, err := c.ListFiles(remotePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Contents of %s:\n", remotePath)
	fmt.Printf("%-30s %-10s %-15s %s\n", "Name", "Type", "Size", "Modified")
	fmt.Println(strings.Repeat("-", 80))

	for _, file := range files {
		fileType := "file"
		if file.IsDir {
			fileType = "dir"
		}

		sizeStr := fmt.Sprintf("%d", file.Size)
		if file.IsDir {
			sizeStr = "-"
		}

		fmt.Printf("%-30s %-10s %-15s %s\n",
			file.Name, fileType, sizeStr, file.ModTime.Format("2006-01-02 15:04"))
	}
}

func handleHealth() {
	fs := flag.NewFlagSet("health", flag.ExitOnError)
	var (
		configFile = fs.String("config", "", "Path to configuration file")
		serverURL  = fs.String("server", "http://localhost:8080", "Server URL")
	)
	fs.Parse(os.Args[2:])

	// Load configuration
	cfg, err := config.LoadClientConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Override with command line flags
	if *serverURL != "http://localhost:8080" {
		cfg.ServerURL = *serverURL
	}

	// Create client
	c := client.NewClient(cfg)

	fmt.Printf("Checking server health at %s...\n", cfg.ServerURL)

	if err := c.CheckHealth(); err != nil {
		fmt.Fprintf(os.Stderr, "Server health check failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Server is healthy!")
}

func handleConfig() {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	var (
		generate = fs.String("generate", "", "Generate configuration file")
		show     = fs.Bool("show", false, "Show current configuration")
	)
	fs.Parse(os.Args[2:])

	if *generate != "" {
		cfg := config.DefaultClientConfig()
		if err := cfg.Save(*generate); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate config file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Configuration file generated: %s\n", *generate)
		return
	}

	if *show {
		cfg := config.DefaultClientConfig()
		fmt.Printf("Default Client Configuration:\n")
		fmt.Printf("  Server URL: %s\n", cfg.ServerURL)
		fmt.Printf("  Timeout: %d seconds\n", cfg.Timeout)
		fmt.Printf("  Concurrency: %d\n", cfg.Concurrency)
		fmt.Printf("  Log Level: %s\n", cfg.LogLevel)
		return
	}

	fmt.Fprintf(os.Stderr, "Usage: %s config --generate <file> | --show\n", os.Args[0])
	os.Exit(1)
}

func handleVersion() {
	fmt.Println("upload-http client v1.0.0")
	fmt.Println("Built with Go")
	fmt.Printf("Build time: %s\n", time.Now().Format("2006-01-02"))
}

func printUsage() {
	fmt.Printf("upload-http client - HTTP file transfer client\n\n")
	fmt.Printf("Usage: %s <command> [options] [arguments]\n\n", os.Args[0])
	fmt.Printf("Commands:\n")
	fmt.Printf("  upload <local-path> [remote-path]  Upload file or folder\n")
	fmt.Printf("  download <remote-path> [local-path] Download file or folder\n")
	fmt.Printf("  list [remote-path]                 List files and folders\n")
	fmt.Printf("  health                             Check server health\n")
	fmt.Printf("  config --generate <file>           Generate configuration file\n")
	fmt.Printf("  config --show                      Show default configuration\n")
	fmt.Printf("  version                            Show version information\n")
	fmt.Printf("  help                               Show this help\n\n")
	fmt.Printf("Global Options:\n")
	fmt.Printf("  --config <file>                    Configuration file path\n")
	fmt.Printf("  --server <url>                     Server URL (default: http://localhost:8080)\n")
	fmt.Printf("  --log-level <level>                Log level (debug, info, warn, error)\n")
	fmt.Printf("  --verbose                          Verbose output\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s upload ./documents              Upload documents folder\n", os.Args[0])
	fmt.Printf("  %s download documents ./downloads  Download documents to downloads\n", os.Args[0])
	fmt.Printf("  %s list                            List root directory\n", os.Args[0])
	fmt.Printf("  %s health                          Check if server is running\n", os.Args[0])
}