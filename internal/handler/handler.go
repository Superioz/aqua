package handler

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/superioz/aqua/internal/config"
	"github.com/superioz/aqua/internal/metrics"
	"github.com/superioz/aqua/internal/storage"
	"github.com/superioz/aqua/pkg/env"
	"k8s.io/klog"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

const (
	SizeMegaByte = 1 << (10 * 2)
)

var (
	// validMimeTypes is a whitelist of all supported
	// mime types. Taken from https://developer.mozilla.org/
	validMimeTypes = []string{
		"application/pdf",
		"application/json",
		"application/gzip",
		"application/vnd.rar",
		"application/zip",
		"application/x-7z-compressed",
		"image/png",
		"image/jpeg",
		"image/gif",
		"image/svg+xml",
		"text/csv",
		"text/plain",
		"audio/mpeg",
		"audio/ogg",
		"audio/opus",
		"audio/webm",
		"video/mp4",
		"video/mpeg",
		"video/webm",
	}

	emptyRequestMetadata = &RequestMetadata{Expiration: storage.ExpireNever}
)

type RequestMetadata struct {
	Expiration int64 `json:"expiration"`
}

type UploadHandler struct {
	AuthConfig  *config.AuthConfig
	FileStorage *storage.FileStorage
}

func NewUploadHandler() *UploadHandler {
	handler := &UploadHandler{}
	handler.ReloadAuthConfig()

	handler.FileStorage = storage.NewFileStorage()
	return handler
}

// ReloadAuthConfig reloads the auth.yml config from the local file system.
func (h *UploadHandler) ReloadAuthConfig() {
	path := env.StringOrDefault("AUTH_CONFIG_PATH", "/etc/aqua/auth.yml")
	ac, err := config.FromLocalFile(path)
	if err != nil {
		// this is not good, but the system still works.
		// nobody can upload a file though.
		klog.Warningf("Could not open auth config at %s: %v", path, err)
		h.AuthConfig = config.NewEmptyAuthConfig()
	} else {
		klog.Infof("Loaded %d valid tokens", len(ac.ValidTokens))
		h.AuthConfig = ac
	}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	// get token for auth
	// empty string, if not given
	token := getToken(c)

	if !h.AuthConfig.HasToken(token) {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "the token is not valid"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		_ = c.Error(err)
		return
	}

	files := form.File["file"]
	if len(files) > 1 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "too many files in form"})
		return
	}
	file := files[0]

	if c.Request.Header.Get("Content-Length") == "" {
		c.Status(http.StatusLengthRequired)
		return
	}
	if c.Request.ContentLength > 50*SizeMegaByte {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"msg": "content size must not exceed 50mb"})
		return
	}

	ct := getContentType(file)
	if !isContentTypeValid(ct) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "content type of file is not valid"})
		return
	}

	if !h.AuthConfig.CanUpload(token, ct) {
		c.JSON(http.StatusForbidden, gin.H{"msg": "you can not upload a file with this content type"})
		return
	}

	mb := float64(c.Request.ContentLength) / 1024 / 1024
	klog.Infof("Received valid upload request (type: %s, size: %.3fmb)", ct, mb)

	of, err := file.Open()
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not open file"})
		return
	}
	defer of.Close()

	metadata := getMetadata(form)
	sf, err := h.FileStorage.StoreFile(of, metadata.Expiration)
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not store file"})
	}

	expiresIn := "never"
	if metadata.Expiration >= 0 {
		expiresIn = fmt.Sprintf("%ds", metadata.Expiration)
	}

	klog.Infof("Stored file %s (expiresIn: %s)", sf.Id, expiresIn)
	metrics.IncFilesUploaded()

	c.JSON(http.StatusOK, gin.H{"id": sf.Id})
}

func getMetadata(form *multipart.Form) *RequestMetadata {
	metaRawList := form.Value["metadata"]
	if len(metaRawList) == 0 {
		return emptyRequestMetadata
	}
	metaRaw := metaRawList[0]

	var metadata *RequestMetadata
	err := json.Unmarshal([]byte(metaRaw), &metadata)
	if err != nil {
		return emptyRequestMetadata
	}
	return metadata
}

// workaround for file Content-Type headers
// which contain multiple values such as "...; charset=utf-8"
func getContentType(f *multipart.FileHeader) string {
	c := f.Header.Get("Content-Type")
	if strings.Contains(c, ";") {
		c = strings.Split(c, ";")[0]
	} else if strings.Contains(c, ",") {
		c = strings.Split(c, ",")[0]
	}
	return c
}

func isContentTypeValid(ct string) bool {
	for _, mt := range validMimeTypes {
		if mt == ct {
			return true
		}
	}
	return false
}

func getToken(c *gin.Context) string {
	// try to get the Bearer token, because it's the standard
	// for authorization
	bearerToken := c.Request.Header.Get("Authorization")
	if !strings.HasPrefix(bearerToken, "Bearer ") {
		// is not a Bearer token, so we don't want it
		return ""
	}

	spl := strings.Split(bearerToken, "Bearer ")
	return spl[1]
}

// HandleStaticFiles takes the files inside the configured file storage
// path and serves them to the client.
func HandleStaticFiles() gin.HandlerFunc {
	fileStoragePath := env.StringOrDefault("FILE_STORAGE_PATH", storage.EnvDefaultFileStoragePath)

	return func(c *gin.Context) {
		fileName := c.Param("file")
		fullPath := fileStoragePath + fileName

		f, err := os.Open(fullPath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		f.Close()

		c.File(fullPath)
	}
}
