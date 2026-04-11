package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-wego/wego/pkg/response"
	"github.com/google/uuid"
)

const (
	maxFileSize = 5 << 20 // 5 MB
	uploadDir   = "./uploads"
)

// UploadHandler handles file upload operations.
type UploadHandler struct {
	uploadPath string
}

// NewUploadHandler constructs an UploadHandler.
func NewUploadHandler(uploadPath string) *UploadHandler {
	return &UploadHandler{uploadPath: uploadPath}
}

// UploadImage godoc
// @Summary      Upload a profile or event image
// @Tags         upload
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Image file (max 5MB, JPG/PNG/WEBP)"
// @Success      200   {object}  map[string]string
// @Router       /api/v1/upload/image [post]
func (h *UploadHandler) UploadImage(c *gin.Context) {
	// Limit the request body size to prevent OOM attacks
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required (max 5MB)")
		return
	}
	defer file.Close()

	// Validate MIME type
	if !isAllowedImage(header.Filename) {
		response.BadRequest(c, "only JPG, PNG, and WEBP images are allowed")
		return
	}

	// Generate a unique filename to prevent path traversal / name collisions
	ext := strings.ToLower(filepath.Ext(header.Filename))
	newName := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	destPath := filepath.Join(h.uploadPath, newName)

	// Ensure the upload directory exists
	if err := os.MkdirAll(h.uploadPath, 0o755); err != nil {
		response.InternalError(c, "storage error")
		return
	}

	dst, err := os.Create(destPath)
	if err != nil {
		response.InternalError(c, "failed to create file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		response.InternalError(c, "failed to save file")
		return
	}

	// Return a URL path that clients can use (in production, replace with CDN URL)
	fileURL := fmt.Sprintf("/static/%s", newName)
	response.OK(c, gin.H{"url": fileURL, "filename": newName})
}

// isAllowedImage checks the file extension against an allowlist.
func isAllowedImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	return allowed[ext]
}
