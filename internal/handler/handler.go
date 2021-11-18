package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/superioz/aqua/internal/config"
	"github.com/superioz/aqua/internal/metrics"
	"github.com/superioz/aqua/internal/mime"
	"github.com/superioz/aqua/internal/request"
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

type UploadHandler struct {
	AuthConfig  *config.AuthConfig
	FileStorage *storage.FileStorage

	// excluded as per defined by the environment variable
	// FILE_EXTENSIONS_EXCEPT
	exclMimeTypes []string
}

func NewUploadHandler() *UploadHandler {
	handler := &UploadHandler{}
	handler.ReloadAuthConfig()

	handler.FileStorage = storage.NewFileStorage()
	handler.exclMimeTypes = env.ListOrDefault("FILE_EXTENSIONS_EXCLUDED", []string{"image/png", "image/jpeg"})
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
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "no file in form"})
		return
	}
	file := files[0]

	if c.Request.Header.Get("Content-Length") == "" {
		c.Status(http.StatusLengthRequired)
		return
	}
	if c.Request.ContentLength > int64(env.IntOrDefault("FILE_MAX_SIZE", 100))*SizeMegaByte {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"msg": "content size must not exceed 50mb"})
		return
	}

	ct := getContentType(file)
	if !mime.IsValid(ct) {
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

	metadata := request.GetMetadata(form)
	rff := &request.RequestFormFile{
		File:          of,
		ContentType:   ct,
		ContentLength: c.Request.ContentLength,
	}

	sf, err := h.FileStorage.StoreFile(rff, metadata.Expiration)
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

	// add extension to file name if it is enabled and
	// the extension is not excluded
	storedName := sf.Id
	if env.BoolOrDefault("FILE_EXTENSIONS_RESPONSE", true) && !h.isExtensionExcluded(sf.MimeType) {
		storedName = fmt.Sprintf("%s.%s", storedName, mime.GetExtension(sf.MimeType))
	}

	c.JSON(http.StatusOK, gin.H{"fileName": storedName})
}

// isExtensionExcluded returns if the given mime type
// should be excluded by the extension response rule, which states if
// an extension should be appended to the file name when responding.
func (h *UploadHandler) isExtensionExcluded(mimeType string) bool {
	for _, exclMimeType := range h.exclMimeTypes {
		if exclMimeType == mimeType {
			return true
		}
	}
	return false
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
	fileStoragePath := env.StringOrDefault("FILE_STORAGE_PATH", config.EnvDefaultFileStoragePath)

	return func(c *gin.Context) {
		fileName := c.Param("file")

		// the file name could contain the extension
		// we split it and ship it.
		if strings.Contains(fileName, ".") {
			fileName = strings.Split(fileName, ".")[0]
		}

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
