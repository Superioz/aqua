package storage

import (
	"io"
	"os"
)

type FileSystem interface {
	// CreateFile writes the content of given reader to a file
	// with given name.
	// Always returns if the file was partly written to disk.
	CreateFile(r io.Reader, name string) (bool, error)
	
	DeleteFile(id string) error
	GetFile(id string) (*os.File, error)
	Exists(id string) (bool, error)
}

type LocalFileSystem struct {
	FolderPath string
}

func (l LocalFileSystem) CreateFile(r io.Reader, name string) (bool, error) {
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
	_, err = io.Copy(f, r)
	if err != nil {
		return true, err
	}
	return true, nil
}

func (l LocalFileSystem) DeleteFile(id string) error {
	return os.Remove(l.FolderPath + id)
}

func (l LocalFileSystem) GetFile(id string) (*os.File, error) {
	return os.Open(l.FolderPath + id)
}

func (l LocalFileSystem) Exists(id string) (bool, error) {
	_, err := os.Stat(l.FolderPath + id)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func NewLocalFileStorage(path string) *LocalFileSystem {
	return &LocalFileSystem{FolderPath: path}
}
