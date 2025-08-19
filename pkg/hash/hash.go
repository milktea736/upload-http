package hash

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
)

// HashType represents different hash algorithms
type HashType string

const (
	MD5    HashType = "md5"
	SHA256 HashType = "sha256"
)

// FileHash represents a file hash with its algorithm
type FileHash struct {
	Algorithm HashType `json:"algorithm"`
	Value     string   `json:"value"`
}

// Hasher provides file hashing functionality
type Hasher struct {
	hashType HashType
}

// NewHasher creates a new hasher with the specified algorithm
func NewHasher(hashType HashType) *Hasher {
	return &Hasher{
		hashType: hashType,
	}
}

// HashFile calculates hash for a file
func (h *Hasher) HashFile(filePath string) (*FileHash, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return h.HashReader(file)
}

// HashReader calculates hash for an io.Reader
func (h *Hasher) HashReader(reader io.Reader) (*FileHash, error) {
	var hasher hash.Hash
	
	switch h.hashType {
	case MD5:
		hasher = md5.New()
	case SHA256:
		hasher = sha256.New()
	default:
		return nil, fmt.Errorf("unsupported hash type: %s", h.hashType)
	}
	
	if _, err := io.Copy(hasher, reader); err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}
	
	return &FileHash{
		Algorithm: h.hashType,
		Value:     fmt.Sprintf("%x", hasher.Sum(nil)),
	}, nil
}

// VerifyFile verifies a file against an expected hash
func (h *Hasher) VerifyFile(filePath string, expectedHash *FileHash) (bool, error) {
	if expectedHash.Algorithm != h.hashType {
		return false, fmt.Errorf("hash algorithm mismatch: expected %s, got %s", expectedHash.Algorithm, h.hashType)
	}
	
	actualHash, err := h.HashFile(filePath)
	if err != nil {
		return false, err
	}
	
	return actualHash.Value == expectedHash.Value, nil
}

// VerifyReader verifies a reader against an expected hash
func (h *Hasher) VerifyReader(reader io.Reader, expectedHash *FileHash) (bool, error) {
	if expectedHash.Algorithm != h.hashType {
		return false, fmt.Errorf("hash algorithm mismatch: expected %s, got %s", expectedHash.Algorithm, h.hashType)
	}
	
	actualHash, err := h.HashReader(reader)
	if err != nil {
		return false, err
	}
	
	return actualHash.Value == expectedHash.Value, nil
}

// String returns the string representation of a FileHash
func (fh *FileHash) String() string {
	return fmt.Sprintf("%s:%s", fh.Algorithm, fh.Value)
}

// DefaultHasher returns a hasher with SHA256 algorithm
func DefaultHasher() *Hasher {
	return NewHasher(SHA256)
}