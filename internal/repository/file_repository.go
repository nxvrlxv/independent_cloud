package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type File struct {
	ID           int64
	OwnerID      int64
	OriginalName string
	StorageKey   string
	SizeBytes    int64
	ContentType  string
	CreatedAt    time.Time
	FolderID     int64
	folder_id    int64
}

type Folder struct {
	FolderID   int64
	ParentID   *int64
	OwnerID    int64
	FolderName string
	CreatedAt  time.Time
}

type FileRepository struct {
	pool *pgxpool.Pool
}

var ErrFileNotFound error = errors.New("File not Found")

func NewFileRepository(pool *pgxpool.Pool) *FileRepository {
	return &FileRepository{pool: pool}
}

func (repo *FileRepository) Save(ctx context.Context, file *File) (int64, error) {
	var id int64
	err := repo.pool.QueryRow(ctx,
		"INSERT INTO files (owner_id, original_name, storage_key, size_bytes, content_type, folder_id) "+
			"VALUES ($1, $2, $3, $4, $5) RETURNING id", file.OwnerID, file.OriginalName, file.StorageKey, file.SizeBytes, file.ContentType).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrFileNotFound
		}
		return 0, fmt.Errorf("save file: %w", err)
	}

	return id, nil
}

func (repo *FileRepository) Delete(ctx context.Context, id int64) error {
	_, err := repo.pool.Exec(ctx,
		"DELETE FROM files WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	return nil
}

func (repo *FileRepository) Update(ctx context.Context, file_id int64, newFilename string) error {
	fmt.Println(newFilename)
	_, err := repo.pool.Exec(ctx,
		"UPDATE files SET original_name = $1 WHERE id = $2", newFilename, file_id)
	if err != nil {
		return fmt.Errorf("Error occured while updating^ %w", err)
	}
	return nil
}

func (repo *FileRepository) GetByID(ctx context.Context, id int64) (*File, error) {
	var file File
	file_row := repo.pool.QueryRow(ctx, "select * from files where id = $1", id)
	err := file_row.Scan(&file.ID, &file.OwnerID, &file.OriginalName, &file.StorageKey, &file.SizeBytes, &file.ContentType, &file.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrFileNotFound
		}
		return nil, fmt.Errorf("geting file failed: %w", err)
	}
	return &file, nil
}

func (repo *FileRepository) ListByOwner(ctx context.Context, ownerID int64) ([]File, error) {
	rows, err := repo.pool.Query(ctx,
		"select * from files where owner_id = $1", ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetched data: %w", err)
	}
	defer rows.Close()
	var ListOfFiles []File
	for rows.Next() {
		f := File{}
		err := rows.Scan(&f.ID, &f.OwnerID, &f.OriginalName, &f.StorageKey, &f.SizeBytes, &f.ContentType, &f.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data: %w", err)
		}
		ListOfFiles = append(ListOfFiles, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return ListOfFiles, nil
}

// Создание пользователем кастомных папок
// (не влияет на структуру хранения файлов на сервере, визуал для пользака)
// Все хендлеры жизненного цикла папок пользователя
func (repo *FileRepository) AddFolder(ctx context.Context, folder *Folder) error {
	_, err := repo.pool.Exec(ctx, "INSERT INTO folders (parent_id, owner_id,folder_name) "+
		"VALUES ($1, $2, $3)", folder.ParentID, folder.OwnerID, folder.FolderName)
	if err != nil {
		return fmt.Errorf("U have a problem with sql query %w", err)
	}
	return nil
}

func (repo *FileRepository) ListFolders(ctx context.Context, ownerID int64) ([]Folder, error) {
	result := []Folder{}

	rows, err := repo.pool.Query(ctx, "SELECT folder_id, parent_id, owner_id, folder_name FROM folders "+
		"WHERE owner_id = $1", ownerID)
	if err != nil {
		return []Folder{}, fmt.Errorf("there is problem with sql-query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var f Folder
		err := rows.Scan(&f.FolderID, &f.ParentID, &f.OwnerID, &f.FolderName)
		if err != nil {
			return []Folder{}, fmt.Errorf("Error in fetching data from cursor: %w", err)
		}
		result = append(result, f)
	}

	return result, nil
}

func (repo *FileRepository) ListOfFilesInFolder(ctx context.Context, folderID int64, ownerID int64) ([]File, error) {
	result := make([]File, 0)
	rows, err := repo.pool.Query(ctx, "select original_name, id, storage_key "+
		"FROM files WHERE folder_id = $1 AND owner_id = $2", folderID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("some error occured in sql query %w", err)
	}

	defer rows.Close()
	for rows.Next() {
		var f File
		err := rows.Scan(&f.OriginalName, &f.ID, &f.StorageKey)
		if err != nil {
			return nil, fmt.Errorf("some error while fetching files data: %w", err)
		}

		result = append(result, f)
	}

	return result, nil
}

func (repo *FileRepository) RenameFolder(ctx context.Context, folderID int64, newFolderName string) error {
	_, err := repo.pool.Exec(ctx, " UPDATE folders SET folder_name = $1 WHERE folder_id = $2", newFolderName, folderID)
	if err != nil {
		return fmt.Errorf("Impossible to update folder^ %w", err)
	}

	return nil
}

func (repo *FileRepository) DeleteFolder(ctx context.Context, folderID int64, ownerID int64) ([]File, error) {
	//1. получение списка файлов из папки
	files, err := repo.ListOfFilesInFolder(ctx, folderID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("can't fetch files from folder: %w", err)
	}

	_, err1 := repo.pool.Exec(ctx, "DELETE FROM folders WHERE folder_id = $1 and owner_id = $2", folderID, ownerID)
	if err1 != nil {
		return nil, fmt.Errorf("error, impossible to delete folder: %w", err1)
	}

	return files, nil
}
