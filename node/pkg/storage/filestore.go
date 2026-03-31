package storage

import (
	"io"
	"mime/multipart"
	"os"
)

// FileStore defines the contract for storage operations
type FileStore interface {
	Save(file *multipart.FileHeader, path string) error
	Delete(path string) error
	Exists(path string) bool
}

// LocalFileStore implements FileStore on the local filesystem
type LocalFileStore struct{}

func NewLocalFileStore(uploadDir string) *LocalFileStore {
	os.MkdirAll(uploadDir, os.ModePerm)
	return &LocalFileStore{}
}

func (l *LocalFileStore) Save(file *multipart.FileHeader, path string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func (l *LocalFileStore) Delete(path string) error {
	return os.Remove(path)
}

func (l *LocalFileStore) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

// FS is the global FileStore instance
var FS FileStore = NewLocalFileStore("./uploads")
