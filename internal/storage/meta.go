package storage

import (
	"database/sql"
	"time"
)

// FileMetaDatabase is for storing additional meta information
// on each file, e.g. the time a file has been uploaded
// or more imporantly when the file should be expired.
//
// On startup, we check for every entry that is expired
// and delete it accordingly.
type FileMetaDatabase interface {
	Connect() error
	WriteFile(sf *StoredFile) error
	GetFile(id string) (*StoredFile, error)
	GetAllFiles() ([]*StoredFile, error)
	GetAllExpired() ([]*StoredFile, error)
	DeleteFile(id string) error
}

type SqliteFileMetaDatabase struct {
	DbFilePath string
}

func NewSqliteFileMetaDatabase(filePath string) *SqliteFileMetaDatabase {
	return &SqliteFileMetaDatabase{
		DbFilePath: filePath,
	}
}

func (s *SqliteFileMetaDatabase) Connect() error {
	return nil
}

func (s *SqliteFileMetaDatabase) WriteFile(sf *StoredFile) error {
	db, err := sql.Open("sqlite", s.DbFilePath)
	if err != nil {
		return err
	}
	defer db.Close()

	err = prepareDb(db)
	if err != nil {
		return err
	}

	stmt, err := db.Prepare(`insert into files(id, uploaded_at, expires_at) values(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(sf.Id, sf.UploadedAt, sf.ExpiresAt)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqliteFileMetaDatabase) DeleteFile(id string) error {
	db, err := sql.Open("sqlite", s.DbFilePath)
	if err != nil {
		return err
	}
	defer db.Close()

	err = prepareDb(db)
	if err != nil {
		return err
	}

	stmt, err := db.Prepare(`delete from files where id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	return err
}

func (s *SqliteFileMetaDatabase) GetFile(id string) (*StoredFile, error) {
	db, err := sql.Open("sqlite", s.DbFilePath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	err = prepareDb(db)
	if err != nil {
		return nil, err
	}

	stmt, err := db.Prepare(`select * from files where id = ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var uploadedAt int
	var expiresAt int

	err = rows.Scan(&id, &uploadedAt, &expiresAt)
	if err != nil {
		return nil, err
	}
	sf := &StoredFile{
		Id:         id,
		UploadedAt: int64(uploadedAt),
		ExpiresAt:  int64(expiresAt),
	}
	return sf, nil
}

func (s *SqliteFileMetaDatabase) GetAllFiles() ([]*StoredFile, error) {
	db, err := sql.Open("sqlite", s.DbFilePath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	err = prepareDb(db)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`select * from files`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sfs []*StoredFile
	for rows.Next() {
		var id string
		var uploadedAt int
		var expiresAt int

		err = rows.Scan(&id, &uploadedAt, &expiresAt)
		if err != nil {
			return nil, err
		}
		sf := &StoredFile{
			Id:         id,
			UploadedAt: int64(uploadedAt),
			ExpiresAt:  int64(expiresAt),
		}
		sfs = append(sfs, sf)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return sfs, nil
}

func (s *SqliteFileMetaDatabase) GetAllExpired() ([]*StoredFile, error) {
	db, err := sql.Open("sqlite", s.DbFilePath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	err = prepareDb(db)
	if err != nil {
		return nil, err
	}

	stmt, err := db.Prepare(`select * from files where expires_at > 0 and expires_at <= ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	rows, err := stmt.Query(now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sfs []*StoredFile
	for rows.Next() {
		var id string
		var uploadedAt int
		var expiresAt int

		err = rows.Scan(&id, &uploadedAt, &expiresAt)
		if err != nil {
			return nil, err
		}
		sf := &StoredFile{
			Id:         id,
			UploadedAt: int64(uploadedAt),
			ExpiresAt:  int64(expiresAt),
		}
		sfs = append(sfs, sf)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return sfs, nil
}

func prepareDb(db *sql.DB) (err error) {
	_, err = db.Exec(`create table if not exists files (
		id text not null primary key, 
		uploaded_at integer,
		expires_at integer
	);`)
	if err != nil {
		return
	}
	return nil
}
