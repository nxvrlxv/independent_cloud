package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	basePath string
}

// конструктор локальных хранилищ
func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{basePath: basePath}
}

func (s *LocalStorage) buildPath(key string) string {
	return filepath.Join(s.basePath, key[0:2], key[2:4], key)
}

func (s *LocalStorage) Save(ctx context.Context, key string, r io.Reader) error {
	fullPath := s.buildPath(key)
	err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
	if err != nil {
		return err
	}
	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}

	defer file.Close()

	nbyte, err := io.Copy(file, r)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully wrote %v bytes!", nbyte)
	return nil
}

func (s *LocalStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := s.buildPath(key)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	fullPath := s.buildPath(key)
	err := os.Remove(fullPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Проверка на совпадение ВСЕГО (буквально)
var _ Storage = (*LocalStorage)(nil)
