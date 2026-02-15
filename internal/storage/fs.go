package storage

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FSBackend implements Backend using the local filesystem
type FSBackend struct {
	Root string
}

// NewFSBackend creates a new filesystem storage backend rooted at the given directory.
func NewFSBackend(root string) *FSBackend {
	return &FSBackend{Root: root}
}

// Init creates the root directory if it does not exist.
func (s *FSBackend) Init() error {
	return os.MkdirAll(s.Root, 0755)
}

// moduleDir returns the path to a module's version directory (e.g. <root>/<module>/@v).
func (s *FSBackend) moduleDir(module string) (string, error) {
	if err := validatePathSegment(module); err != nil {
		return "", err
	}
	return filepath.Join(s.Root, module, "@v"), nil
}

// filePath returns the full path to a specific module file (e.g. <root>/<module>/@v/<version>.zip).
func (s *FSBackend) filePath(module, version, ext string) (string, error) {
	if err := validatePathSegment(module); err != nil {
		return "", err
	}
	if err := validatePathSegment(version); err != nil {
		return "", err
	}
	// ext is trusted as it's hardcoded in callers, but good to check
	if strings.Contains(ext, "/") || strings.Contains(ext, "\\") {
		return "", fmt.Errorf("invalid extension")
	}

	dir, err := s.moduleDir(module)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, version+ext), nil
}

// validatePathSegment checks that a path segment is safe (no traversal, no absolute paths).
func validatePathSegment(segment string) error {
	if segment == "" {
		return fmt.Errorf("empty path segment")
	}
	if strings.Contains(segment, "..") {
		return fmt.Errorf("path traversal attempt")
	}
	if filepath.IsAbs(segment) {
		return fmt.Errorf("absolute path not allowed")
	}
	return nil
}

// Exists checks whether a specific module file exists in local storage.
func (s *FSBackend) Exists(module, version, ext string) (bool, error) {
	path, err := s.filePath(module, version, ext)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Get opens a module file for reading.
func (s *FSBackend) Get(module, version, ext string) (io.ReadCloser, error) {
	path, err := s.filePath(module, version, ext)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

// Save writes a module file to local storage, creating directories as needed.
func (s *FSBackend) Save(module, version, ext string, content io.Reader) error {
	dir, err := s.moduleDir(module)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, version+ext) // dir is already safe and joined with Root
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, content)
	if err != nil {
		fmt.Printf("[Storage] Failed to write file %s: %v\n", path, err)
	}
	return err
}

// ListVersions returns all cached version strings for a module (based on .info files).
func (s *FSBackend) ListVersions(module string) ([]string, error) {
	dir, err := s.moduleDir(module)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".info") {
			versions = append(versions, strings.TrimSuffix(name, ".info"))
		}
	}
	return versions, nil
}

// Walk iterates over all cached modules, calling fn for each one found.
func (s *FSBackend) Walk(fn func(module Module) error) error {
	return filepath.WalkDir(s.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Look for directory named "@v"
		if !d.IsDir() || d.Name() != "@v" {
			return nil
		}

		// module path is everything relative to Root up to this point
		relPath, _ := filepath.Rel(s.Root, path) // e.g. github.com/user/repo/@v
		moduleName := filepath.Dir(relPath)      // e.g. github.com/user/repo

		m := Module{
			Path: moduleName,
		}

		files, err := os.ReadDir(path)
		if err != nil {
			return nil // skip unreadable directories
		}

		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".zip") {
				version := strings.TrimSuffix(f.Name(), ".zip")
				m.Versions = append(m.Versions, version)
				info, err := f.Info()
				if err == nil {
					m.Size += info.Size()
					if info.ModTime().After(m.UpdatedAt) {
						m.UpdatedAt = info.ModTime()
					}
				}
			}
		}

		if len(m.Versions) > 0 {
			if err := fn(m); err != nil {
				return err
			}
		}

		return nil
	})
}
