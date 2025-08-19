# upload-http

A high-performance HTTP-based file transfer system with folder upload/download support, built in Go.

## 📋 Features

### Server Features
- ✅ HTTP Server with RESTful API for file operations
- ✅ Folder structure preservation during transfers
- ✅ Multi-file parallel processing
- ✅ File integrity verification (SHA256 hash validation)
- ✅ Transfer status tracking via API endpoints
- ✅ Configuration file support (JSON format)
- ✅ Comprehensive logging system
- ✅ HTTPS support with TLS certificates
- ✅ Health check endpoint
- ✅ Cross-Origin Resource Sharing (CORS) support

### Client Features
- ✅ Upload entire folders with recursive directory support
- ✅ Download folders as compressed archives (tar.gz)
- ✅ Download individual files
- ✅ Configurable concurrent transfers
- ✅ Real-time progress tracking
- ✅ File integrity verification
- ✅ Comprehensive CLI interface
- ✅ Remote file/directory listing

## 🚀 Quick Start

### Installation

1. Clone the repository:
```bash
git clone https://github.com/milktea736/upload-http.git
cd upload-http
```

2. Build the binaries:
```bash
go build -o bin/server ./cmd/server
go build -o bin/client ./cmd/client
```

### Running the Server

1. Start with default settings:
```bash
./bin/server
```

2. Start with custom configuration:
```bash
./bin/server --config configs/server.json --port 9090 --storage ./files
```

3. Generate a configuration file:
```bash
./bin/server --generate-config ./my-server-config.json
```

### Using the Client

1. Upload a folder:
```bash
./bin/client upload ./documents
```

2. Download a folder:
```bash
./bin/client download documents ./downloads
```

3. List remote files:
```bash
./bin/client list
./bin/client list documents
```

4. Check server health:
```bash
./bin/client health
```

## 📖 Detailed Usage

### Server Configuration

The server can be configured via JSON configuration file or command-line flags:

```json
{
  "port": 8080,
  "host": "0.0.0.0",
  "storage_path": "./uploads",
  "max_file_size": 104857600,
  "log_level": "info",
  "enable_https": false,
  "cert_file": "",
  "key_file": ""
}
```

#### Configuration Options:
- `port`: Server listening port (default: 8080)
- `host`: Server listening host (default: 0.0.0.0)
- `storage_path`: Directory for storing uploaded files (default: ./uploads)
- `max_file_size`: Maximum file size in bytes (default: 100MB)
- `log_level`: Logging level - debug, info, warn, error (default: info)
- `enable_https`: Enable HTTPS (default: false)
- `cert_file`: Path to TLS certificate file (required for HTTPS)
- `key_file`: Path to TLS private key file (required for HTTPS)

#### Server Command Line Options:
```bash
./bin/server [options]

Options:
  --config string        Path to configuration file
  --port int            Server port (default 8080)
  --host string         Server host (default "0.0.0.0")
  --storage string      Storage directory path (default "./uploads")
  --log-level string    Log level (default "info")
  --max-size int        Maximum file size in bytes (default 104857600)
  --generate-config string  Generate configuration file and exit
  --version             Show version information
```

### Client Configuration

Client configuration can be set via JSON file or command-line flags:

```json
{
  "server_url": "http://localhost:8080",
  "timeout": 300,
  "concurrency": 4,
  "log_level": "info"
}
```

#### Configuration Options:
- `server_url`: Server URL (default: http://localhost:8080)
- `timeout`: Request timeout in seconds (default: 300)
- `concurrency`: Number of concurrent uploads (default: 4)
- `log_level`: Logging level (default: info)

#### Client Commands:

1. **Upload Files/Folders**:
```bash
./bin/client upload [options] <local-path> [remote-path]

Options:
  --config string      Path to configuration file
  --server string      Server URL (default "http://localhost:8080")
  --verbose           Verbose output
  --log-level string  Log level (default "info")

Examples:
  ./bin/client upload ./documents
  ./bin/client upload ./file.txt remote/file.txt
  ./bin/client upload ./photos --server http://myserver:8080
```

2. **Download Files/Folders**:
```bash
./bin/client download [options] <remote-path> [local-path]

Examples:
  ./bin/client download documents ./downloads
  ./bin/client download file.txt ./local-file.txt
```

3. **List Remote Files**:
```bash
./bin/client list [options] [remote-path]

Examples:
  ./bin/client list
  ./bin/client list documents
```

4. **Health Check**:
```bash
./bin/client health [options]
```

5. **Configuration Management**:
```bash
./bin/client config --generate client-config.json
./bin/client config --show
```

## 🔧 API Endpoints

### Server REST API

1. **Upload Files**: `POST /api/upload`
   - Accepts multipart form data
   - Returns transfer ID for status tracking

2. **Download File/Folder**: `GET /api/download?path=<path>`
   - Downloads single files directly
   - Downloads folders as tar.gz archives

3. **List Files**: `GET /api/list?path=<path>`
   - Returns JSON array of file information

4. **Transfer Status**: `GET /api/status/<transfer-id>`
   - Returns detailed transfer progress information

5. **Health Check**: `GET /health`
   - Returns server health status

### API Response Examples

**Upload Response**:
```json
{
  "transfer_id": "transfer_1640995200000",
  "status": "started"
}
```

**File List Response**:
```json
[
  {
    "name": "document.pdf",
    "is_dir": false,
    "size": 1024000,
    "mod_time": "2023-12-01T10:00:00Z"
  },
  {
    "name": "photos",
    "is_dir": true,
    "size": 0,
    "mod_time": "2023-12-01T09:30:00Z"
  }
]
```

**Transfer Status Response**:
```json
{
  "id": "transfer_1640995200000",
  "type": "upload",
  "status": "running",
  "progress": 0.75,
  "total_files": 100,
  "processed_files": 75,
  "total_size": 104857600,
  "processed_size": 78643200,
  "start_time": "2023-12-01T10:00:00Z"
}
```

## 🔒 Security Features

1. **Path Sanitization**: Prevents directory traversal attacks
2. **File Size Limits**: Configurable maximum file size
3. **HTTPS Support**: TLS encryption for secure transfers
4. **Hash Verification**: SHA256 integrity checking
5. **Input Validation**: Comprehensive request validation

## 🧪 Testing

Run the included tests:
```bash
go test ./...
```

Test with sample files:
```bash
# Terminal 1: Start server
./bin/server --port 8080

# Terminal 2: Test uploads
mkdir test-data
echo "Hello World" > test-data/file1.txt
echo "Hello Go" > test-data/file2.txt
./bin/client upload test-data

# Test downloads
./bin/client download test-data ./downloaded
```

## 🐛 Troubleshooting

### Common Issues

1. **Server won't start**:
   - Check if port is already in use
   - Verify storage directory permissions
   - Check configuration file syntax

2. **Upload fails**:
   - Verify file size limits
   - Check disk space on server
   - Ensure network connectivity

3. **Download fails**:
   - Verify file exists on server
   - Check local directory permissions
   - Verify server URL is correct

### Debug Mode

Enable debug logging for detailed troubleshooting:
```bash
# Server
./bin/server --log-level debug

# Client
./bin/client upload ./files --log-level debug --verbose
```

## 📊 Performance

- **Concurrent Uploads**: Configurable parallelism (default: 4 concurrent transfers)
- **Memory Efficient**: Streaming transfers for large files
- **Compressed Downloads**: Folders downloaded as compressed tar.gz
- **Progress Tracking**: Real-time transfer progress
- **Resume Support**: Planned for future versions

## 🛣️ Roadmap

- [ ] Resume interrupted transfers
- [ ] Transfer bandwidth limiting
- [ ] User authentication and authorization
- [ ] File versioning
- [ ] WebSocket-based real-time progress
- [ ] Docker containerization
- [ ] Web UI for file management

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🆘 Support

For support and questions:
- Open an issue on GitHub
- Check the troubleshooting section
- Review the API documentation

---

Built with ❤️ in Go
