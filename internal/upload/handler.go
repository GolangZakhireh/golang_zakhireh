package upload

import (
	"fmt"
	"log"
	"net/http"

	"golangzakhireh/internal/storage"
)

// UploadHandler handles POST requests to upload module files (.info, .mod, .zip).
type UploadHandler struct {
	Storage storage.Backend
}

// NewUploadHandler creates a new upload handler backed by the given storage.
func NewUploadHandler(store storage.Backend) *UploadHandler {
	return &UploadHandler{Storage: store}
}

// ServeHTTP handles a module upload request.
// Expects a multipart POST with fields: module, version, info, mod, zip.
func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	module := r.FormValue("module")
	version := r.FormValue("version")

	if module == "" || version == "" {
		http.Error(w, "module and version required", http.StatusBadRequest)
		return
	}

	saveFile := func(formKey, ext string) error {
		file, _, err := r.FormFile(formKey)
		if err != nil {
			return err
		}
		defer file.Close()

		return h.Storage.Save(module, version, ext, file)
	}

	if err := saveFile("info", ".info"); err != nil {
		http.Error(w, fmt.Sprintf("failed saving info: %v", err), http.StatusInternalServerError)
		return
	}

	if err := saveFile("mod", ".mod"); err != nil {
		http.Error(w, fmt.Sprintf("failed saving mod: %v", err), http.StatusInternalServerError)
		return
	}

	if err := saveFile("zip", ".zip"); err != nil {
		http.Error(w, fmt.Sprintf("failed saving zip: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[UPLOAD] %s@%s uploaded by %s", module, version, r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}
