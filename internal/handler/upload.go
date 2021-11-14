package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/superioz/aqua/internal/config"
	"github.com/superioz/aqua/internal/storage"
	"github.com/superioz/aqua/pkg/env"
	"k8s.io/klog"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const (
	SizeMegaByte = 1 << (10 * 2)
)

var (
	validMimeTypes = []string{
		"application/pdf",
		"application/json",
		"image/png",
		"image/jpeg",
		"text/csv",
		"text/plain",
	}
)

type UploadHandler struct {
	authConfig  *config.AuthConfig
	fileMetaDb  storage.FileMetaDatabase
	fileStorage storage.FileStorage
}

func NewUploadHandler() *UploadHandler {
	handler := &UploadHandler{}

	path := env.StringOrDefault("AUTH_CONFIG_PATH", "/etc/aqua/auth.yml")
	ac, err := config.FromLocalFile(path)
	if err != nil {
		// this is not good, but the system still works.
		// nobody can upload a file though.
		klog.Warningf("Could not open auth config at %s: %v", path, err)
		handler.authConfig = config.NewEmptyAuthConfig()
	} else {
		klog.Infof("Loaded %d valid tokens", len(ac.ValidTokens))
		handler.authConfig = ac
	}

	metaDbFilePath := env.StringOrDefault("FILE_META_DB_FILE", "./files.db")
	handler.fileMetaDb = storage.NewSqliteFileMetaDatabase(metaDbFilePath)

	fileStoragePath := env.StringOrDefault("FILE_STORAGE_PATH", "/var/lib/aqua/")
	handler.fileStorage = storage.NewLocalFileStorage(fileStoragePath)
	return handler
}

func (u *UploadHandler) Upload(c *gin.Context) {
	// get token for auth
	// empty string, if not given
	token := getToken(c)

	klog.Infof("Checking authentication for token=%s", token)
	if !u.authConfig.HasToken(token) {
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
	klog.Infof("Detected content type: %s", ct)
	if !isContentTypeValid(ct) {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "content type of file is not valid"})
		return
	}

	if !u.authConfig.CanUpload(token, ct) {
		c.JSON(http.StatusForbidden, gin.H{"msg": "you can not upload a file with this content type"})
		return
	}

	of, err := file.Open()
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not open file"})
		return
	}
	defer of.Close()

	name, err := getRandomFileName(env.IntOrDefault("FILE_NAME_LENGTH", 8))
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not generate random name"})
		return
	}

	_, err = u.fileStorage.CreateFile(of, name)
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not save file"})
	}

	t := time.Now()
	sf := &storage.StoredFile{
		Id:         name,
		UploadedAt: t.Unix(),
		ExpiresAt:  t.Add(time.Duration(storage.ExpireNever)).Unix(),
	}

	// write to meta database
	u.fileMetaDb.WriteFile(sf)

	c.JSON(http.StatusOK, gin.H{"id": sf.Id})
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

// getRandomFileName returns a random string with a fixed size
// that is generated from an uuid.
// It also checks, that no file with that name already exists,
// if that is the case, it generates a new one.
func getRandomFileName(size int) (string, error) {
	if size <= 1 {
		return "", errors.New("size must be greater than 1")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	// strip '-' from uuid
	str := strings.ReplaceAll(id.String(), "-", "")
	if size >= len(str) {
		return str, nil
	}
	return str[:size], nil
}
