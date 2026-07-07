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
		"INSERT INTO files (owner_id, original_name, storage_key, size_bytes, content_type) "+
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
