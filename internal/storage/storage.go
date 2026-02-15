package storage

import (
	"io"
	"time"
)

// Module represents a cached module with statistics
type Module struct {
	Path      string
	Versions  []string
	Size      int64
	UpdatedAt time.Time
}

// Backend defines the storage interface for Go modules
type Backend interface {
	// Initialize the storage (e.g. ensure directories exist)
	Init() error

	// Exists checks if a specific file exists
	// ext should include the dot, e.g. ".zip", ".mod", ".info"
	Exists(module, version, ext string) (bool, error)

	// Get opens a file for reading
	Get(module, version, ext string) (io.ReadCloser, error)

	// Save writes content to a file
	Save(module, version, ext string, content io.Reader) error

	// ListVersions returns all known versions for a module
	ListVersions(module string) ([]string, error)

	// Walk iterates over all stored modules
	Walk(fn func(module Module) error) error
}
