# upload-http

HTTP file upload and download server with CLI client built in Go. This repository implements a complete file transfer system supporting folder uploads/downloads, progress tracking, and resumable transfers.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Bootstrap and Setup
- Initialize Go module: `go mod init github.com/milktea736/upload-http`
- Install dependencies: `go mod tidy` (takes 5-10 seconds)
- Format code: `go fmt ./...`
- Vet code: `go vet ./...`
- Build project: `go build` (takes 1-2 seconds for simple builds)
- Run tests: `go test -v ./...` (takes 1-2 seconds for basic tests)

### Required Go Version
- Go 1.23.2 or later (repository was tested with Go 1.24.6)
- Standard Go toolchain includes all necessary build tools

### Development Workflow
- ALWAYS run `go fmt ./...` before committing code
- ALWAYS run `go vet ./...` to catch common errors
- ALWAYS run `go test ./...` to ensure tests pass
- Build timing: Basic builds complete in under 5 seconds. NEVER CANCEL builds.
- Test timing: Unit tests complete in under 10 seconds. NEVER CANCEL test runs.

## Project Structure

Based on issue #1, this project implements:

### Server Component
- HTTP server for file upload/download API
- Folder structure preservation
- Multi-file parallel processing  
- File integrity verification (hash checksums)
- Transfer status API endpoints
- Configuration file support (port, storage path)
- Logging system

### CLI Client Component
- Upload folders: `client upload <local-folder> <server-url>`
- Download folders: `client download <server-url/folder-id> <local-path>`
- List server folders: `client list <server-url>`
- Resumable transfers
- Error retry mechanism (configurable retry count)
- Detailed progress display
- Configuration file support

## Build Commands

### Development Build
- `go build -o server cmd/server/main.go` - Build server binary
- `go build -o client cmd/client/main.go` - Build client binary
- `go build ./...` - Build all packages (takes 2-5 seconds)

### Production Build
- `go build -ldflags="-s -w" -o server cmd/server/main.go` - Optimized server
- `go build -ldflags="-s -w" -o client cmd/client/main.go` - Optimized client

### Cross-platform Build
- `GOOS=windows GOARCH=amd64 go build -o client.exe cmd/client/main.go`
- `GOOS=linux GOARCH=amd64 go build -o client-linux cmd/client/main.go`
- `GOOS=darwin GOARCH=amd64 go build -o client-macos cmd/client/main.go`

## Testing

### Unit Tests
- `go test ./...` - Run all tests (completes in under 10 seconds)
- `go test -v ./...` - Verbose test output
- `go test -race ./...` - Test with race condition detection
- `go test -cover ./...` - Test with coverage report

### Integration Tests
- Start test server: `go run cmd/server/main.go -port 8080 -dir ./test-uploads`
- Test upload: `curl -F "file=@testfile.txt" http://localhost:8080/upload`
- Test client: `go run cmd/client/main.go upload ./testfolder http://localhost:8080`

## Validation Scenarios

### CRITICAL: Manual Validation Requirements
After any changes to the codebase, ALWAYS perform these validation steps:

#### Basic Server Functionality
1. Start server: `go run cmd/server/main.go`
2. Create test file: `echo "test content" > test.txt`
3. Test upload: `curl -F "file=@test.txt" http://localhost:8080/upload`
4. Verify response indicates successful upload
5. Check file exists in uploads directory
6. Stop server with Ctrl+C

#### Client CLI Functionality  
1. Start server in background: `go run cmd/server/main.go &`
2. Create test directory with files: `mkdir testdir && echo "content" > testdir/file.txt`
3. Test upload: `go run cmd/client/main.go upload testdir http://localhost:8080`
4. Test list: `go run cmd/client/main.go list http://localhost:8080`  
5. Test download: `go run cmd/client/main.go download http://localhost:8080/testdir ./downloaded`
6. Verify downloaded files match original
7. Kill background server

#### Progress Display Validation
- Verify file count progress: "X/Y files completed"
- Verify size progress: "X MB / Y MB transferred"
- Verify transfer speed display: "X.X MB/s"
- Verify ETA calculation appears

#### Error Handling Validation
- Test network interruption recovery
- Test file permission errors
- Test disk space errors
- Verify retry mechanism works
- Check error messages are user-friendly

## Common Dependencies

The project uses these common Go packages:
- `net/http` - HTTP server and client
- `mime/multipart` - File upload handling
- `path/filepath` - Cross-platform file paths
- `crypto/sha256` - File integrity checksums
- `encoding/json` - Configuration and API responses
- `github.com/gorilla/mux` - HTTP routing (install with `go get github.com/gorilla/mux`)
- `github.com/cheggaaa/pb/v3` - Progress bars (install with `go get github.com/cheggaaa/pb/v3`)

## Configuration

### Server Configuration (config.json)
```json
{
  "port": 8080,
  "upload_dir": "./uploads",
  "max_file_size": 1073741824,
  "enable_logging": true,
  "log_file": "server.log"
}
```

### Client Configuration (~/.upload-http-config.json)
```json
{
  "default_server": "http://localhost:8080",
  "retry_count": 3,
  "chunk_size": 1048576,
  "parallel_uploads": 4
}
```

## Linting and Quality

### Standard Go Tools
- `go fmt ./...` - Format code (takes 1-2 seconds)
- `go vet ./...` - Static analysis (takes 2-3 seconds)  
- `goimports -w .` - Fix imports (install with `go install golang.org/x/tools/cmd/goimports@latest`)

### Additional Linting (Optional)
- `golangci-lint run` - Comprehensive linting (install from https://golangci-lint.run/)
- `staticcheck ./...` - Advanced static analysis (install with `go install honnef.co/go/tools/cmd/staticcheck@latest`)

## Debugging

### Server Debugging
- Enable debug logging: `go run cmd/server/main.go -debug`
- Use delve debugger: `dlv debug cmd/server/main.go`
- HTTP endpoint debugging: `curl -v http://localhost:8080/status`

### Client Debugging  
- Verbose client output: `go run cmd/client/main.go -v upload ./dir http://localhost:8080`
- Network debugging: `go run cmd/client/main.go -debug upload ./dir http://localhost:8080`

## Performance Considerations

### Timing Expectations
- Go module initialization: 1-2 seconds
- Basic build (no dependencies): 1-2 seconds  
- Build with dependencies: 5-10 seconds
- Unit test suite: 5-10 seconds
- Integration tests: 15-30 seconds (includes server startup/shutdown)
- File upload (1MB): 1-2 seconds on localhost
- File upload (100MB): 10-30 seconds on localhost

### Timeout Settings
- NEVER CANCEL builds - they complete quickly (under 30 seconds)
- NEVER CANCEL tests - unit tests complete in under 30 seconds
- Set 60+ minute timeouts for large file transfer tests
- Set 30+ minute timeouts for integration test suites

## Repository Structure

```
upload-http/
├── cmd/
│   ├── server/           # Server main package
│   └── client/           # Client main package  
├── internal/
│   ├── server/           # Server implementation
│   ├── client/           # Client implementation
│   └── common/           # Shared utilities
├── pkg/                  # Public packages
├── configs/              # Configuration files
├── docs/                 # Documentation
├── scripts/              # Build and deployment scripts
└── tests/                # Integration tests
```

## Common Tasks

### Initialize New Project
```bash
go mod init github.com/milktea736/upload-http
mkdir -p cmd/server cmd/client internal/server internal/client internal/common pkg
touch cmd/server/main.go cmd/client/main.go
```

### Add Dependencies
```bash
go get github.com/gorilla/mux          # HTTP routing
go get github.com/cheggaaa/pb/v3       # Progress bars  
go get github.com/spf13/cobra          # CLI framework
go get github.com/spf13/viper          # Configuration management
go mod tidy                            # Clean up dependencies
```

### Quick Development Test
```bash
# Terminal 1: Start server
go run cmd/server/main.go

# Terminal 2: Test upload
echo "test" > test.txt
curl -F "file=@test.txt" http://localhost:8080/upload
```

## Environment Variables

- `UPLOAD_HTTP_PORT` - Server port (default: 8080)
- `UPLOAD_HTTP_DIR` - Upload directory (default: ./uploads)  
- `UPLOAD_HTTP_DEBUG` - Enable debug logging (default: false)
- `UPLOAD_HTTP_CONFIG` - Configuration file path

## Security Considerations

- ALWAYS validate file paths to prevent directory traversal
- ALWAYS check file sizes to prevent disk exhaustion
- ALWAYS sanitize uploaded filenames
- Consider implementing authentication for production use
- Use HTTPS in production environments

## Troubleshooting

### Common Issues
- "permission denied" - Check file/directory permissions
- "port already in use" - Kill existing server or use different port
- "no such file or directory" - Verify file paths are correct
- Build fails - Run `go mod tidy` to resolve dependencies

### Network Issues
- Connection refused - Verify server is running
- Timeout errors - Check firewall settings
- SSL/TLS errors - Verify certificates if using HTTPS

This repository implements a production-ready HTTP file transfer system. Always validate changes by running the complete upload/download workflow before committing code.