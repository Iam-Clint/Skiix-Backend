package http

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
)

const (
	maxUploadBytes = 10 << 20 // 10MB
)

type UploadHandler struct {
	cld *cloudinary.Cloudinary
}

func NewUploadHandler(cld *cloudinary.Cloudinary) *UploadHandler {
	return &UploadHandler{cld: cld}
}

type uploadResponse struct {
	URL          string `json:"url"`
	ResourceType string `json:"resource_type"` // image|video|raw
	PublicID     string `json:"public_id"`
	Bytes        int    `json:"bytes"`
}

// UploadMedia godoc
// @Summary      Upload image/video to Cloudinary
// @Description  Upload a multipart file (image/video) to Cloudinary and returns secure URL
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Security     Bearer
// @Param        file  formData  file  true  "media file (image/video)"
// @Success      200   {object}  uploadResponse
// @Failure      400   {object}  map[string]string
// @Failure      413   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /api/v1/upload [post]
func (h *UploadHandler) UploadMedia(c *gin.Context) {
	// Enforce size limit at HTTP layer (prevents large uploads from consuming memory).
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes)

	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file field"})
		return
	}
	if fh.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty file"})
		return
	}
	if fh.Size > maxUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large (max 10MB)"})
		return
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	// Basic allow-list. Cloudinary also validates; this is fast fail.
	allowed := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
		".mp4": true, ".mov": true, ".m4v": true, ".webm": true,
	}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type"})
		return
	}

	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not open uploaded file"})
		return
	}
	defer f.Close()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// ResourceType auto: Cloudinary will decide image vs video.
	// Use folder per app, and enable auto optimization for images via eager transformations at delivery time.
	res, err := h.cld.Upload.Upload(ctx, f, uploader.UploadParams{
		ResourceType: "auto",
		Folder:       "skiix",
		// These are safe defaults; you can override on delivery URLs too.
		Transformation: "f_auto,q_auto",
	})
	if err != nil {
		msg := err.Error()
		// If MaxBytesReader triggered
		if strings.Contains(strings.ToLower(msg), "request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file too large (max 10MB)"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		return
	}

	c.JSON(http.StatusOK, uploadResponse{
		URL:          res.SecureURL,
		ResourceType: res.ResourceType,
		PublicID:     res.PublicID,
		Bytes:        res.Bytes,
	})
}
