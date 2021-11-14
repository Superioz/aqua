package storage

import (
	"io"
	"mime/multipart"
	"os"
)

type FileStorage interface {
	CreateFile(mf multipart.File, name string) (bool, error)
	DeleteFile(id string) error
	GetFile(id string) (*os.File, error)
}

type LocalFileStorage struct {
	FolderPath string
}

func (l LocalFileStorage) CreateFile(mf multipart.File, name string) (bool, error) {
	err := os.MkdirAll(l.FolderPath, os.ModePerm)
	if err != nil {
		return false, err
	}

	f, err := os.Create(l.FolderPath + name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	// use io.Copy so that we don't have to load all the image into the memory.
	// they get copied in smaller 32kb chunks.
	_, err = io.Copy(f, mf)
	if err != nil {
		return true, err
	}
	return true, nil
}

func (l LocalFileStorage) DeleteFile(id string) error {
	panic("implement me")
}

func (l LocalFileStorage) GetFile(id string) (*os.File, error) {
	panic("implement me")
}

func NewLocalFileStorage(path string) *LocalFileStorage {
	return &LocalFileStorage{FolderPath: path}
}
