package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/nxvrlxv/independent_cloud/internal/repository"
	"github.com/nxvrlxv/independent_cloud/internal/storage"
)

type FileService struct {
	storage storage.Storage
	repo    *repository.FileRepository
}

var ErrAccessDenied error = errors.New("Access denied, u are not an owner")

func NewFileService(storage storage.Storage, repo *repository.FileRepository) *FileService {
	return &FileService{storage: storage, repo: repo}
}

func (service *FileService) Upload(ctx context.Context, r io.Reader, ownerID int64, filename string, contentType string) (int64, error) {
	storage_key := uuid.New().String()
	size_bytes, err := service.storage.Save(ctx, storage_key, r)
	if err != nil {
		return 0, fmt.Errorf("Unexpected error while saving file: %w", err)
	}

	file := repository.File{OwnerID: ownerID,
		OriginalName: filename,
		SizeBytes:    size_bytes,
		StorageKey:   storage_key,
		ContentType:  contentType}

	id, err := service.repo.Save(ctx, &file)
	if err != nil {
		if delErr := service.storage.Delete(ctx, storage_key); delErr != nil {
			return 0, fmt.Errorf("save metadata failed: %w; orphan cleanup also failed: %v", err, delErr)
		}
		return 0, fmt.Errorf("Unexpected error while saving metadata from file: %w", err)
	}

	return id, nil
}

func (service *FileService) Download(ctx context.Context, file_id int64, user_id int64) (io.ReadCloser, *repository.File, error) {
	file, err := service.repo.GetByID(ctx, file_id)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, nil, repository.ErrFileNotFound
		}
		return nil, nil, fmt.Errorf("get file: %w", err)
	}

	if user_id != file.OwnerID {
		return nil, nil, ErrAccessDenied
	}

	r, err := service.storage.Open(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("error to open potok: %w", err)
	}

	return r, file, nil
}

func (service *FileService) Delete(ctx context.Context, file_id int64, user_id int64) error {
	file, err := service.repo.GetByID(ctx, file_id)
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}

	if user_id != file.OwnerID {
		return ErrAccessDenied
	}

	err1 := service.repo.Delete(ctx, file_id)
	if err1 != nil {
		if errors.Is(err1, repository.ErrFileNotFound) {
			return repository.ErrFileNotFound
		}
		return fmt.Errorf("unexpected error while deleting: %w", err1)
	}

	err2 := service.storage.Delete(ctx, file.StorageKey)
	if err2 != nil {

	}

	return nil
}

func (service *FileService) List(ctx context.Context, ownerID int64) ([]repository.File, error) {
	files, err := service.repo.ListByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("some error occured: %w", err)
	}

	return files, nil
}

func (service *FileService) Update(ctx context.Context, file_id int64, newFilename string) error {
	err := service.repo.Update(ctx, file_id, newFilename)
	if err != nil {
		return fmt.Errorf("some error occured while updating filename: %w", err)
	}

	return nil
}
