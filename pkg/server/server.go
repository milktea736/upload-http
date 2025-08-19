package server

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/milktea736/upload-http/internal/utils"
	"github.com/milktea736/upload-http/pkg/config"
	"github.com/milktea736/upload-http/pkg/hash"
)

// TransferStatus represents the status of a transfer operation
type TransferStatus struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // "upload" or "download"
	Status      string    `json:"status"` // "running", "completed", "failed"
	Progress    float64   `json:"progress"` // 0.0 to 1.0
	TotalFiles  int       `json:"total_files"`
	ProcessedFiles int    `json:"processed_files"`
	TotalSize   int64     `json:"total_size"`
	ProcessedSize int64   `json:"processed_size"`
	StartTime   time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// Server represents the HTTP file server
type Server struct {
	config      *config.ServerConfig
	logger      *utils.Logger
	hasher      *hash.Hasher
	transfers   map[string]*TransferStatus
	transfersMu sync.RWMutex
}

// NewServer creates a new server instance
func NewServer(config *config.ServerConfig) *Server {
	return &Server{
		config:    config,
		logger:    utils.NewLogger(config.LogLevel),
		hasher:    hash.DefaultHasher(),
		transfers: make(map[string]*TransferStatus),
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	
	// API routes
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/download", s.handleDownload)
	mux.HandleFunc("/api/status/", s.handleStatus)
	mux.HandleFunc("/api/list", s.handleList)
	mux.HandleFunc("/health", s.handleHealth)
	
	server := &http.Server{
		Addr:    s.config.Address(),
		Handler: s.corsMiddleware(mux),
	}
	
	s.logger.Info("Starting server on %s", s.config.Address())
	s.logger.Info("Storage path: %s", s.config.StoragePath)
	
	if s.config.EnableHTTPS {
		if s.config.CertFile == "" || s.config.KeyFile == "" {
			return fmt.Errorf("HTTPS enabled but cert_file or key_file not specified")
		}
		return server.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
	}
	
	return server.ListenAndServe()
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now(),
		"storage_path": s.config.StoragePath,
	})
}

// handleUpload handles file upload requests
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse multipart form
	err := r.ParseMultipartForm(s.config.MaxFileSize)
	if err != nil {
		s.logger.Error("Failed to parse multipart form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	
	transferID := generateTransferID()
	status := &TransferStatus{
		ID:        transferID,
		Type:      "upload",
		Status:    "running",
		StartTime: time.Now(),
	}
	
	s.transfersMu.Lock()
	s.transfers[transferID] = status
	s.transfersMu.Unlock()
	
	go s.processUpload(transferID, r.MultipartForm)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"transfer_id": transferID,
		"status": "started",
	})
}

// processUpload processes uploaded files
func (s *Server) processUpload(transferID string, form *multipart.Form) {
	s.transfersMu.RLock()
	status := s.transfers[transferID]
	s.transfersMu.RUnlock()
	
	defer func() {
		endTime := time.Now()
		status.EndTime = &endTime
	}()
	
	files := form.File["files"]
	status.TotalFiles = len(files)
	
	// Calculate total size
	for _, fileHeader := range files {
		status.TotalSize += fileHeader.Size
	}
	
	for i, fileHeader := range files {
		if err := s.processUploadedFile(fileHeader, status); err != nil {
			s.logger.Error("Failed to process file %s: %v", fileHeader.Filename, err)
			status.Status = "failed"
			status.Error = err.Error()
			return
		}
		
		status.ProcessedFiles = i + 1
		status.Progress = float64(status.ProcessedFiles) / float64(status.TotalFiles)
	}
	
	status.Status = "completed"
	s.logger.Info("Upload completed: %s (%d files)", transferID, status.TotalFiles)
}

// processUploadedFile processes a single uploaded file
func (s *Server) processUploadedFile(fileHeader *multipart.FileHeader, status *TransferStatus) error {
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer file.Close()
	
	// Create destination path
	destPath := filepath.Join(s.config.StoragePath, fileHeader.Filename)
	destDir := filepath.Dir(destPath)
	
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()
	
	// Copy file with progress tracking
	written, err := io.Copy(destFile, file)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	
	status.ProcessedSize += written
	
	// Calculate hash for verification
	if _, err := s.hasher.HashFile(destPath); err != nil {
		s.logger.Warn("Failed to calculate hash for %s: %v", destPath, err)
	}
	
	s.logger.Debug("Uploaded file: %s (%d bytes)", destPath, written)
	return nil
}

// handleDownload handles file download requests
func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Path parameter required", http.StatusBadRequest)
		return
	}
	
	// Sanitize path to prevent directory traversal
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	fullPath := filepath.Join(s.config.StoragePath, cleanPath)
	
	// Check if path exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		http.Error(w, "File or directory not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.logger.Error("Failed to stat file: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	if info.IsDir() {
		s.handleDirectoryDownload(w, r, fullPath, cleanPath)
	} else {
		s.handleFileDownload(w, r, fullPath, cleanPath)
	}
}

// handleFileDownload handles single file download
func (s *Server) handleFileDownload(w http.ResponseWriter, r *http.Request, fullPath, cleanPath string) {
	file, err := os.Open(fullPath)
	if err != nil {
		s.logger.Error("Failed to open file: %v", err)
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	
	// Set headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(cleanPath)))
	
	// Calculate and set hash header
	if fileHash, err := s.hasher.HashFile(fullPath); err == nil {
		w.Header().Set("X-File-Hash", fileHash.String())
	}
	
	// Copy file to response
	if _, err := io.Copy(w, file); err != nil {
		s.logger.Error("Failed to write file to response: %v", err)
	}
	
	s.logger.Info("Downloaded file: %s", cleanPath)
}

// handleDirectoryDownload handles directory download as tar.gz
func (s *Server) handleDirectoryDownload(w http.ResponseWriter, r *http.Request, fullPath, cleanPath string) {
	transferID := generateTransferID()
	
	// Set headers for tar.gz download
	filename := filepath.Base(cleanPath) + ".tar.gz"
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("X-Transfer-ID", transferID)
	
	// Create gzip writer
	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()
	
	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	
	// Walk directory and add files to tar
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Get relative path
		relPath, err := filepath.Rel(fullPath, path)
		if err != nil {
			return err
		}
		
		// Create tar header
		header := &tar.Header{
			Name: relPath,
			Size: info.Size(),
			Mode: int64(info.Mode()),
			ModTime: info.ModTime(),
		}
		
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		
		// Open and copy file
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		
		_, err = io.Copy(tarWriter, file)
		return err
	})
	
	if err != nil {
		s.logger.Error("Failed to create tar archive: %v", err)
		return
	}
	
	s.logger.Info("Downloaded directory: %s as %s", cleanPath, filename)
}

// handleStatus returns transfer status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	transferID := strings.TrimPrefix(r.URL.Path, "/api/status/")
	if transferID == "" {
		http.Error(w, "Transfer ID required", http.StatusBadRequest)
		return
	}
	
	s.transfersMu.RLock()
	status, exists := s.transfers[transferID]
	s.transfersMu.RUnlock()
	
	if !exists {
		http.Error(w, "Transfer not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleList returns list of files and directories
func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}
	
	// Sanitize path
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	fullPath := filepath.Join(s.config.StoragePath, cleanPath)
	
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		s.logger.Error("Failed to read directory: %v", err)
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}
	
	type FileInfo struct {
		Name    string    `json:"name"`
		IsDir   bool      `json:"is_dir"`
		Size    int64     `json:"size"`
		ModTime time.Time `json:"mod_time"`
	}
	
	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		
		files = append(files, FileInfo{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

// generateTransferID generates a unique transfer ID
func generateTransferID() string {
	return fmt.Sprintf("transfer_%d", time.Now().UnixNano())
}