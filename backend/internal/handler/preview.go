package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"surajdrive/backend/internal/imageconv"
	"surajdrive/backend/internal/model"
	"surajdrive/backend/internal/storage"
)

const (
	previewURLTTL          = 15 * time.Minute
	previewJPEGContentType = "image/jpeg"
)

// Preview returns a presigned URL to a JPEG render of the requested object.
// For HEIC/HEIF inputs it lazily generates and caches a JPEG preview in the
// `.previews/` prefix of the user's bucket. For all other file types it
// behaves like PresignDownload (pass-through).
func (h *FileHandler) Preview(w http.ResponseWriter, r *http.Request) {
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

	if !isHEICKey(key) {
		urlValue, err := h.store.PresignedGetURL(r.Context(), bucket, key, previewURLTTL)
		if err != nil {
			writeStorageError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, model.PresignResponse{
			URL:       urlValue,
			Key:       key,
			ExpiresIn: "15m",
		})
		return
	}

	previewKey := previewKeyFor(key)

	exists, err := h.store.ObjectExists(r.Context(), bucket, previewKey)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	if !exists {
		heicBytes, err := h.store.GetObject(r.Context(), bucket, key)
		if err != nil {
			writeStorageError(w, err)
			return
		}

		jpegBytes, err := imageconv.HEICToJPEG(heicBytes)
		if err != nil {
			log.Error().Err(err).Str("key", key).Msg("heic preview conversion failed")
			writeError(w, http.StatusUnprocessableEntity, fmt.Errorf("failed to convert HEIC: %w", err))
			return
		}

		if err := h.store.PutObject(
			r.Context(),
			bucket,
			previewKey,
			previewJPEGContentType,
			bytes.NewReader(jpegBytes),
			int64(len(jpegBytes)),
		); err != nil {
			log.Error().Err(err).Str("preview_key", previewKey).Msg("failed to upload heic preview")
			writeStorageError(w, err)
			return
		}
	}

	urlValue, err := h.store.PresignedGetURL(r.Context(), bucket, previewKey, previewURLTTL)
	if err != nil {
		writeStorageError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.PresignResponse{
		URL:       urlValue,
		Key:       previewKey,
		ExpiresIn: "15m",
	})
}

func isHEICKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.HasSuffix(lower, ".heic") || strings.HasSuffix(lower, ".heif")
}

// previewKeyFor returns a deterministic, collision-resistant cache key for
// the JPEG preview of the given source object key.
func previewKeyFor(sourceKey string) string {
	sum := sha256.Sum256([]byte(sourceKey))
	return storage.PreviewsPrefix + hex.EncodeToString(sum[:]) + ".jpg"
}
