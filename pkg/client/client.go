package client

import (
	"archive/tar"
	"bytes"
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

// TransferProgress represents upload/download progress
type TransferProgress struct {
	TotalFiles     int
	ProcessedFiles int
	TotalSize      int64
	ProcessedSize  int64
	CurrentFile    string
}

// ProgressCallback is called during transfers to report progress
type ProgressCallback func(progress *TransferProgress)

// Client represents the HTTP client for file operations
type Client struct {
	config     *config.ClientConfig
	logger     *utils.Logger
	hasher     *hash.Hasher
	httpClient *http.Client
}

// NewClient creates a new client instance
func NewClient(config *config.ClientConfig) *Client {
	return &Client{
		config: config,
		logger: utils.NewLogger(config.LogLevel),
		hasher: hash.DefaultHasher(),
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
}

// UploadFolder uploads a folder to the server
func (c *Client) UploadFolder(localPath, remotePath string, progressCallback ProgressCallback) error {
	c.logger.Info("Starting upload of folder: %s -> %s", localPath, remotePath)
	
	// Collect all files to upload
	files, err := c.collectFiles(localPath)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}
	
	if len(files) == 0 {
		return fmt.Errorf("no files found in directory: %s", localPath)
	}
	
	progress := &TransferProgress{
		TotalFiles: len(files),
	}
	
	// Calculate total size
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		progress.TotalSize += info.Size()
	}
	
	// Process files in batches based on concurrency
	sem := make(chan struct{}, c.config.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var uploadErr error
	
	for _, filePath := range files {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore
		
		go func(fp string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore
			
			// Calculate relative path
			relPath, err := filepath.Rel(localPath, fp)
			if err != nil {
				mu.Lock()
				if uploadErr == nil {
					uploadErr = fmt.Errorf("failed to get relative path for %s: %w", fp, err)
				}
				mu.Unlock()
				return
			}
			
			// Adjust remote path
			fullRemotePath := filepath.Join(remotePath, relPath)
			
			if err := c.UploadFile(fp, fullRemotePath); err != nil {
				mu.Lock()
				if uploadErr == nil {
					uploadErr = fmt.Errorf("failed to upload %s: %w", fp, err)
				}
				mu.Unlock()
				return
			}
			
			mu.Lock()
			progress.ProcessedFiles++
			
			// Update processed size
			if info, err := os.Stat(fp); err == nil {
				progress.ProcessedSize += info.Size()
			}
			
			progress.CurrentFile = relPath
			
			if progressCallback != nil {
				progressCallback(progress)
			}
			mu.Unlock()
			
			c.logger.Debug("Uploaded: %s", relPath)
		}(filePath)
	}
	
	wg.Wait()
	
	if uploadErr != nil {
		return uploadErr
	}
	
	c.logger.Info("Upload completed: %d files", progress.ProcessedFiles)
	return nil
}

// UploadFile uploads a single file
func (c *Client) UploadFile(localPath, remotePath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	c.logger.Debug("Uploading file: local='%s', remote='%s'", localPath, remotePath)
	
	// Create multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	
	// Add the remote path as a separate field
	pathField, err := writer.CreateFormField("remote_path")
	if err != nil {
		return fmt.Errorf("failed to create remote path field: %w", err)
	}
	if _, err := pathField.Write([]byte(remotePath)); err != nil {
		return fmt.Errorf("failed to write remote path: %w", err)
	}
	
	// Add file to form (use just the base filename for the multipart filename)
	part, err := writer.CreateFormFile("files", filepath.Base(remotePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	
	writer.Close()
	
	// Create request
	url := c.config.ServerURL + "/api/upload"
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// DownloadFolder downloads a folder from the server
func (c *Client) DownloadFolder(remotePath, localPath string, progressCallback ProgressCallback) error {
	c.logger.Info("Starting download of folder: %s -> %s", remotePath, localPath)
	
	// Ensure local directory exists
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}
	
	// Download as tar.gz
	url := fmt.Sprintf("%s/api/download?path=%s", c.config.ServerURL, remotePath)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download folder: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Extract tar.gz
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	
	tarReader := tar.NewReader(gzipReader)
	
	progress := &TransferProgress{}
	
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}
		
		// Create file path
		filePath := filepath.Join(localPath, header.Name)
		
		// Ensure directory exists
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		
		// Create file
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
		
		// Copy file content
		written, err := io.Copy(file, tarReader)
		file.Close()
		
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", filePath, err)
		}
		
		// Set file permissions and modification time
		if err := os.Chmod(filePath, os.FileMode(header.Mode)); err != nil {
			c.logger.Warn("Failed to set permissions for %s: %v", filePath, err)
		}
		
		if err := os.Chtimes(filePath, header.ModTime, header.ModTime); err != nil {
			c.logger.Warn("Failed to set modification time for %s: %v", filePath, err)
		}
		
		progress.ProcessedFiles++
		progress.ProcessedSize += written
		progress.CurrentFile = header.Name
		
		if progressCallback != nil {
			progressCallback(progress)
		}
		
		c.logger.Debug("Downloaded: %s (%d bytes)", header.Name, written)
	}
	
	c.logger.Info("Download completed: %d files", progress.ProcessedFiles)
	return nil
}

// DownloadFile downloads a single file from the server
func (c *Client) DownloadFile(remotePath, localPath string) error {
	c.logger.Info("Downloading file: %s -> %s", remotePath, localPath)
	
	// Ensure local directory exists
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}
	
	// Download file
	url := fmt.Sprintf("%s/api/download?path=%s", c.config.ServerURL, remotePath)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()
	
	// Copy content
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	// Verify hash if provided
	hashHeader := resp.Header.Get("X-File-Hash")
	if hashHeader != "" {
		if err := c.verifyFileHash(localPath, hashHeader); err != nil {
			c.logger.Warn("Hash verification failed for %s: %v", localPath, err)
		} else {
			c.logger.Debug("Hash verification passed for %s", localPath)
		}
	}
	
	c.logger.Info("Downloaded file: %s (%d bytes)", localPath, written)
	return nil
}

// ListFiles lists files and directories on the server
func (c *Client) ListFiles(remotePath string) ([]FileInfo, error) {
	url := fmt.Sprintf("%s/api/list?path=%s", c.config.ServerURL, remotePath)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var files []FileInfo
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	return files, nil
}

// CheckHealth checks server health
func (c *Client) CheckHealth() error {
	url := c.config.ServerURL + "/health"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to check health: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server unhealthy: status %d", resp.StatusCode)
	}
	
	return nil
}

// collectFiles recursively collects all files in a directory
func (c *Client) collectFiles(dir string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			files = append(files, path)
		}
		
		return nil
	})
	
	return files, err
}

// verifyFileHash verifies a file against the provided hash
func (c *Client) verifyFileHash(filePath, hashStr string) error {
	parts := strings.SplitN(hashStr, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid hash format: %s", hashStr)
	}
	
	algorithm := hash.HashType(parts[0])
	expectedValue := parts[1]
	
	hasher := hash.NewHasher(algorithm)
	fileHash, err := hasher.HashFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}
	
	if fileHash.Value != expectedValue {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedValue, fileHash.Value)
	}
	
	return nil
}

// FileInfo represents file information from server
type FileInfo struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}