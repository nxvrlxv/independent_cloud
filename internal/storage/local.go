package storage

import "path/filepath"

// import (
// 	"filepath"
// )

type LocalStorage struct {
	basePath string
}

//конструктор локальных хранилищ
func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{basePath: basePath}
}

func (s *LocalStorage) buildPath(key string) string {
	return filepath.Join(s.basePath, key[0:2], key[2:4], key)
}

//Проверка на совпадение ВСЕГО (буквально)
var _ Storage = (*LocalStorage)(nil)
