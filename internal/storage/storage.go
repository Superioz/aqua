package storage

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/superioz/aqua/internal/metrics"
	"github.com/superioz/aqua/pkg/env"
	"k8s.io/klog"
	"mime/multipart"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	ExpireNever = -1

	EnvDefaultFileStoragePath = "/var/lib/aqua/files/"
	EnvDefaultMetaDbPath      = "/var/lib/aqua/"
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
	metaDbFilePath := env.StringOrDefault("FILE_META_DB_PATH", EnvDefaultMetaDbPath)
	fileMetaDb := NewSqliteFileMetaDatabase(metaDbFilePath)
	err := fileMetaDb.Connect()
	if err != nil {
		klog.Errorf("Could not connect to file meta db: %v", err)
	}

	fileStoragePath := env.StringOrDefault("FILE_STORAGE_PATH", EnvDefaultFileStoragePath)
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

		// check if file doesn't exist anymore
		ok, err := fs.fileSystem.Exists(file.Id)
		if err != nil {
			return fmt.Errorf("could not check if file exists with id=%s: %v", file.Id, err)
		}
		if ok {
			// file exists
			// delete this file
			err = fs.fileSystem.DeleteFile(file.Id)
			if err != nil {
				return fmt.Errorf("could not delete file with id=%s: %v", file.Id, err)
			}
		}

		err = fs.fileMetaDb.DeleteFile(file.Id)
		if err != nil {
			return fmt.Errorf("could not delete file with id=%s: %v", file.Id, err)
		}

		klog.Infof("Delete file %s (expired at %s)", file.Id, time.Unix(file.ExpiresAt, 0).String())
		metrics.IncFilesExpired()
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

	currentTime := time.Now().Unix()
	expAt := currentTime + expiration
	if expiration == ExpireNever {
		expAt = ExpireNever
	}

	sf := &StoredFile{
		Id:         name,
		UploadedAt: currentTime,
		ExpiresAt:  expAt,
	}

	// write to meta database
	err = fs.fileMetaDb.WriteFile(sf)
	if err != nil {
		return nil, err
	}

	return sf, nil
}

var escaper = strings.NewReplacer("9", "99", "-", "90", "_", "91")

// getRandomFileName returns a random string with a fixed size
// that is generated from an uuid.
func getRandomFileName(size int) (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	str := escaper.Replace(base64.RawURLEncoding.EncodeToString([]byte(id.String())))
	if size >= len(str) {
		return str, nil
	}
	return str[:size], nil
}
