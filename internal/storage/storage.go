package storage

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/superioz/aqua/pkg/env"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	ExpireNever = -1
)

type StoredFile struct {
	Id         string
	UploadedAt int64
	ExpiresAt  int64
}

func (sf *StoredFile) String() string {
	return fmt.Sprintf("StoredFile<%s, %s>", sf.Id, time.Unix(sf.UploadedAt, 0).String())
}

func StoreFile(mf multipart.File, expiration int64) (*StoredFile, error) {
	name, err := getRandomFileName(env.IntOrDefault("FILE_NAME_LENGTH", 8))
	if err != nil {
		return nil, err
	}
	path := env.StringOrDefault("FILE_STORAGE_PATH", "/var/lib/aqua/")

	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	f, err := os.Create(path + name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// use io.Copy so that we don't have to load all the image into the memory.
	// they get copied in smaller 32kb chunks.
	_, err = io.Copy(f, mf)
	if err != nil {
		return nil, err
	}

	t := time.Now()
	sf := &StoredFile{
		Id:         name,
		UploadedAt: t.Unix(),
		ExpiresAt:  t.Add(time.Duration(expiration)).Unix(),
	}

	return sf, nil
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
