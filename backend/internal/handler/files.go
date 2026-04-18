package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"surajdrive/backend/internal/storage"
)

type FileHandler struct {
	store *storage.MinIOClient
}

func NewFileHandler(store *storage.MinIOClient) *FileHandler {
	return &FileHandler{store: store}
}

func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	bucket, err := bucketFromRequest(r, h.store)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	offset, limit, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	response, err := h.store.ListObjects(r.Context(), bucket, r.URL.Query().Get("prefix"), offset, limit)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	bucket, err := bucketFromRequest(r, h.store)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("key is required"))
		return
	}

	if err := h.store.DeleteObject(r.Context(), bucket, key); err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"deleted": key})
}

func (h *FileHandler) Copy(w http.ResponseWriter, r *http.Request) {
	bucket, err := bucketFromRequest(r, h.store)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	var body struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Src == "" || body.Dst == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("src and dst are required"))
		return
	}

	resolvedDst, err := h.store.CopyObject(r.Context(), bucket, body.Src, body.Dst)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"src": body.Src, "dst": resolvedDst})
}
