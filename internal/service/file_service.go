package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

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

func (service *FileService) Upload(ctx context.Context, r io.Reader, ownerID int64, filename string, contentType string, FolderID *int64) (int64, error) {
	storage_key := uuid.New().String()
	size_bytes, err := service.storage.Save(ctx, storage_key, r)
	if err != nil {
		return 0, fmt.Errorf("Unexpected error while saving file: %w", err)
	}

	file := repository.File{OwnerID: ownerID,
		OriginalName: filename,
		SizeBytes:    size_bytes,
		StorageKey:   storage_key,
		ContentType:  contentType,
		FolderID:     FolderID}

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

func (service *FileService) ChangeFolder(ctx context.Context, fileID int64, newFolderID *int64) error {
	err := service.repo.ChangeFolder(ctx, fileID, newFolderID)
	if err != nil {
		return fmt.Errorf("Some error occured: %v", err)
	}

	return nil
}

// folders life cycle
func (service *FileService) AddFolder(ctx context.Context, ownerID int64, folderName string, parentID *int64) (int64, error) {
	folder := repository.Folder{
		ParentID:   parentID,
		OwnerID:    ownerID,
		FolderName: folderName}
	FolderID, err := service.repo.AddFolder(ctx, &folder)
	if err != nil {
		return 0, fmt.Errorf("error on service while adding floder: %w", err)
	}
	return FolderID, nil
}

func (service *FileService) ListOfFolders(ctx context.Context, ownerID int64) ([]repository.Folder, error) {
	result, err := service.repo.ListFolders(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("some error when listing in service: %w", err)
	}

	return result, nil
}

func (service *FileService) ListOfFilesInFolder(ctx context.Context, folderID int64, ownerID int64) ([]repository.File, error) {
	result, err := service.repo.ListOfFilesInFolder(ctx, folderID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("error list of files service: %w", err)
	}

	return result, nil
}

func (service *FileService) RenameFolder(ctx context.Context, folderID int64, newFolderName string) error {
	err := service.repo.RenameFolder(ctx, folderID, newFolderName)
	if err != nil {
		return fmt.Errorf("error rename service: %w", err)
	}
	return nil
}

func (service *FileService) DeleteFolder(ctx context.Context, folderID int64, ownerID int64) error {
	var wg sync.WaitGroup
	var mux sync.Mutex
	wPool := make(chan struct{}, 50)
	files, err := service.ListOfFilesInFolder(ctx, folderID, ownerID)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("something went wrong while getting files from deleted folder: %v", err)
	}

	err1 := service.repo.DeleteFolder(ctx, folderID, ownerID)
	if err1 != nil {
		fmt.Println(err1)
		return fmt.Errorf("error select files service: %v", err1)
	}

	errSlice := make([]error, 0, len(files))
	//параллельно удаляем файлы из storage

	for _, value := range files {
		wg.Add(1)
		go func(value repository.File) {
			defer wg.Done()
			wPool <- struct{}{}
			err := service.storage.Delete(ctx, value.StorageKey)
			if err != nil {
				mux.Lock()
				defer mux.Unlock()
				errSlice = append(errSlice, err)
				return
			}
			<-wPool
		}(value)
	}

	wg.Wait()

	return errors.Join(errSlice...)
}
