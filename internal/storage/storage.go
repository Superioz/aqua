package storage

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/superioz/aqua/pkg/env"
	"k8s.io/klog"
	"mime/multipart"
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

// FileStorage is the abstraction layer for storing uploaded files.
// It consists of a file system where the physical files are written to
// and a seperate database, where it stores metadata for each file.
type FileStorage struct {
	fileMetaDb FileMetaDatabase
	fileSystem FileSystem
}

func NewFileStorage() *FileStorage {
	metaDbFilePath := env.StringOrDefault("FILE_META_DB_FILE", "./files.db")
	fileMetaDb := NewSqliteFileMetaDatabase(metaDbFilePath)

	fileStoragePath := env.StringOrDefault("FILE_STORAGE_PATH", "/var/lib/aqua/")
	fileSystem := NewLocalFileStorage(fileStoragePath)

	return &FileStorage{
		fileMetaDb: fileMetaDb,
		fileSystem: fileSystem,
	}
}

// Cleanup uses the meta database to check for all files
// that have expired and deletes them accordingly.
func (fs *FileStorage) Cleanup() error {
	klog.Infoln("Cleanup expired files")
	expiredFiles, err := fs.fileMetaDb.GetAllExpired()
	if err != nil {
		return err
	}
	if len(expiredFiles) == 0 {
		klog.Infoln("No expired files found.")
		return nil
	}

	for _, file := range expiredFiles {
		if file.ExpiresAt < 0 {
			continue
		}

		klog.Infof("Delete file: %s", file)
		// delete this file
		err = fs.fileSystem.DeleteFile(file.Id)
		if err != nil {
			return errors.New(fmt.Sprintf("Could not delete file with id=%s: %v", file.Id, err))
		}

		err = fs.fileMetaDb.DeleteFile(file.Id)
		if err != nil {
			return errors.New(fmt.Sprintf("Could not delete file with id=%s: %v", file.Id, err))
		}
	}
	return nil
}

func (fs *FileStorage) StoreFile(of multipart.File, expiration int64) (*StoredFile, error) {
	name, err := getRandomFileName(env.IntOrDefault("FILE_NAME_LENGTH", 8))
	if err != nil {
		return nil, errors.New("could not generate random name")
	}

	_, err = fs.fileSystem.CreateFile(of, name)
	if err != nil {
		klog.Error(err)
		return nil, errors.New("could not save file to system")
	}

	t := time.Now()
	expAt := t.Add(time.Duration(expiration)).Unix()
	if expiration == ExpireNever {
		expAt = ExpireNever
	}

	sf := &StoredFile{
		Id:         name,
		UploadedAt: t.Unix(),
		ExpiresAt:  expAt,
	}

	// write to meta database
	fs.fileMetaDb.WriteFile(sf)

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
