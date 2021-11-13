package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/superioz/aqua/internal/config"
	"github.com/superioz/aqua/pkg/env"
	"io"
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
	validMimeTypes = []string{
		"application/pdf",
		"application/json",
		"image/png",
		"image/jpeg",
		"text/csv",
		"text/plain",
	}

	authConfig *config.AuthConfig
)

func Initialize() {
	path := env.StringOrDefault("AUTH_CONFIG_PATH", "/etc/aqua/auth.yml")
	ac, err := config.FromLocalFile(path)
	if err != nil {
		// this is not good, but the system still works.
		// nobody can upload a file though.
		klog.Warningf("Could not open auth config at %s: %v", path, err)
		authConfig = config.NewEmptyAuthConfig()
		return
	} else {
		klog.Infof("Loaded %d valid tokens", len(ac.ValidTokens))
	}
	authConfig = ac
}

func Upload(c *gin.Context) {
	// get token for auth
	// empty string, if not given
	token := getToken(c)

	klog.Infof("Checking authentication for token=%s", token)
	if !authConfig.HasToken(token) {
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

	of, err := file.Open()
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not open file"})
		return
	}
	defer of.Close()

	name, err := getRandomFileName(8)
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not generate id of file"})
		return
	}

	path := env.StringOrDefault("FILE_STORAGE_PATH", "/var/lib/aqua/")

	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not create file directory"})
		return
	}

	f, err := os.Create(name)
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not create file"})
		return
	}
	defer f.Close()

	// use io.Copy so that we don't have to load all the image into the memory.
	// they get copied in smaller 32kb chunks.
	_, err = io.Copy(f, of)
	if err != nil {
		klog.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "could not copy content to file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": name})
}

func getRandomFileName(size int) (string, error) {
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
