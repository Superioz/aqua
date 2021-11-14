package storage

import (
	"fmt"
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
