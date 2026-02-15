package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golangzakhireh/internal/storage"
)

// File extensions used in the Go module proxy protocol.
const (
	ExtInfo = ".info" // Module version metadata
	ExtMod  = ".mod"  // go.mod file
	ExtZip  = ".zip"  // Source code archive
)

// ProxyHandler serves Go module requests following the GOPROXY protocol.
// It checks local storage first, then falls back to the upstream proxy.
type ProxyHandler struct {
	Storage    storage.Backend // Local module cache
	Fallback   string          // Upstream proxy URL (e.g. https://proxy.golang.org)
	HttpClient *http.Client    // HTTP client for upstream requests
	Validator  *Validator      // Access control validator
}

// NewProxyHandler creates a new proxy handler with the given storage, upstream URL, and validator.
func NewProxyHandler(store storage.Backend, fallback string, validator *Validator) *ProxyHandler {
	return &ProxyHandler{
		Storage:    store,
		Fallback:   fallback,
		HttpClient: &http.Client{},
		Validator:  validator,
	}
}

// ServeHTTP handles incoming module requests.
// Supports both the standard Go proxy protocol and /proxy/<module>/@latest requests.
//
// Supported paths:
//
//	/proxy/<module>/@v/<version>.<ext>
//	/proxy/<module>/@latest
//	/proxy/<module>/@v/list
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean path and handle both /proxy/ prefixed (if not stripped) and direct requests
	reqPath := r.URL.Path
	if strings.HasPrefix(reqPath, "/proxy/") {
		reqPath = strings.TrimPrefix(reqPath, "/proxy/")
	}
	reqPath = strings.TrimPrefix(reqPath, "/")

	// Handle '@latest' requests (e.g. /proxy/github.com/labstack/echo/v4/@latest)
	if strings.HasSuffix(reqPath, "/@latest") {
		mod := strings.TrimSuffix(reqPath, "/@latest")
		if mod == "" {
			http.Error(w, "invalid module path", http.StatusBadRequest)
			return
		}

		if !h.Validator.IsAllowed(mod) {
			http.Error(w, "module access denied", http.StatusForbidden)
			return
		}

		// Try to serve .info for @latest from local storage
		if served, err := h.serveLocal(w, mod, "latest", ExtInfo); err != nil {
			http.Error(w, fmt.Sprintf("storage error: %v", err), http.StatusInternalServerError)
			return
		} else if served {
			return
		}

		// Fallback to the upstream proxy, mapping '@latest' to the upstream path and fetching .info
		h.serveFallback(w, mod, "latest", ExtInfo, "latest"+ExtInfo)
		return
	}

	// Handle /@v/list for module version listing (e.g. /proxy/github.com/labstack/echo/v4/@v/list)
	if strings.HasSuffix(reqPath, "/@v/list") {
		mod := strings.TrimSuffix(reqPath, "/@v/list")
		if mod == "" {
			http.Error(w, "invalid module path", http.StatusBadRequest)
			return
		}
		if !h.Validator.IsAllowed(mod) {
			http.Error(w, "module access denied", http.StatusForbidden)
			return
		}
		h.handleList(w, mod)
		return
	}

	// Handle standard Go proxy protocol file requests: /<module>/@v/<version>.<ext>
	parts := strings.SplitN(reqPath, "/@v/", 2)
	if len(parts) != 2 {
		http.Error(w, "invalid module request", http.StatusBadRequest)
		return
	}
	mod := parts[0]
	file := parts[1]

	if !h.Validator.IsAllowed(mod) {
		http.Error(w, "module access denied", http.StatusForbidden)
		return
	}

	// Determine extension and version
	var ext string
	switch {
	case strings.HasSuffix(file, ExtInfo):
		ext = ExtInfo
	case strings.HasSuffix(file, ExtMod):
		ext = ExtMod
	case strings.HasSuffix(file, ExtZip):
		ext = ExtZip
	case file == "list":
		h.handleList(w, mod)
		return
	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	version := strings.TrimSuffix(file, ext)

	// Try to serve module from local storage
	if served, err := h.serveLocal(w, mod, version, ext); err != nil {
		fmt.Printf("[Proxy] Error checking local storage for %s: %v\n", mod, err)
		http.Error(w, fmt.Sprintf("storage error: %v", err), http.StatusInternalServerError)
		return
	} else if served {
		fmt.Printf("[Proxy] Hit: %s %s\n", mod, version+ext)
		return
	}

	// If not found locally, try fallback
	// serveFallback will handle its own error logging if fetch fails
	if h.serveFallback(w, mod, version, ext, file) {
		fmt.Printf("[Proxy] Miss (Fetched): %s %s\n", mod, version+ext)
	} else {
		// Just log that it failed to serve, details are in serveFallback
		fmt.Printf("[Proxy] Failed to serve: %s %s\n", mod, version+ext)
	}
}

// serveFallback fetches a module file from the upstream proxy...
// Returns true if successfully served, false otherwise
// handleList responds with all cached versions of a module, one per line.
func (h *ProxyHandler) handleList(w http.ResponseWriter, mod string) {
	versions, err := h.Storage.ListVersions(mod)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list versions: %v", err), http.StatusInternalServerError)
		return
	}
	for _, v := range versions {
		fmt.Fprintln(w, v)
	}
}

// serveLocal attempts to serve a module file from local storage.
// Returns (true, nil) if found and served, (false, nil) if not found.
func (h *ProxyHandler) serveLocal(w http.ResponseWriter, mod, version, ext string) (bool, error) {
	exists, err := h.Storage.Exists(mod, version, ext)
	if err != nil {
		return false, err
	}

	if exists {
		content, err := h.Storage.Get(mod, version, ext)
		if err != nil {
			return false, fmt.Errorf("failed to read file: %w", err)
		}
		defer content.Close()
		_, err = io.Copy(w, content)
		return true, err
	}
	return false, nil
}

// serveFallback fetches a module file from the upstream proxy,
// caches it locally, and serves it to the client.
// Returns true if successfully served, false otherwise
func (h *ProxyHandler) serveFallback(w http.ResponseWriter, mod, version, ext, file string) bool {
	if h.Fallback == "" {
		http.Error(w, "module not found", http.StatusNotFound)
		return false
	}

	// Build the upstream URL
	var fallbackURL string
	// Handle @latest translation: map to /<module>/@latest for .info fetches (like upstream proxy expects)
	if version == "latest" && ext == ExtInfo {
		// Upstream expects /<mod>/@latest (no extension)
		fallbackURL = fmt.Sprintf("%s/%s/@latest", strings.TrimSuffix(h.Fallback, "/"), mod)
	} else if file == "list" {
		// /<mod>/@v/list
		fallbackURL = fmt.Sprintf("%s/%s/@v/list", strings.TrimSuffix(h.Fallback, "/"), mod)
	} else {
		// /<mod>/@v/<version>.<ext>
		fallbackURL = fmt.Sprintf("%s/%s/@v/%s", strings.TrimSuffix(h.Fallback, "/"), mod, file)
	}

	resp, err := h.HttpClient.Get(fallbackURL)
	if err != nil {
		fmt.Printf("[Proxy] Fallback error for %s: %v\n", mod, err)
		http.Error(w, fmt.Sprintf("fallback error: %v", err), http.StatusBadGateway)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("[Proxy] Upstream returned %d for %s (URL: %s)\n", resp.StatusCode, mod, fallbackURL)
		http.Error(w, fmt.Sprintf("module not found: %s", file), resp.StatusCode)
		return false
	}

	// Read body to memory to save it
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read upstream body: %v", err), http.StatusBadGateway)
		return false
	}

	// Save to local storage for future cache hits
	if err := h.Storage.Save(mod, version, ext, bytes.NewReader(bodyBytes)); err != nil {
		// Log error but try to serve anyway
		fmt.Printf("failed to save to storage: %v\n", err)
	}

	// Serve to client
	w.Write(bodyBytes)
	return true
}

// IsModuleRequest checks if a path looks like a Go module request
func IsModuleRequest(path string) bool {
	return strings.Contains(path, "/@v/") || strings.Contains(path, "/@latest")
}
