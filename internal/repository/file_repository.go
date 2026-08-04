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
	ID           int64     `json:"id"`
	OwnerID      int64     `json:"-"`
	OriginalName string    `json:"original_name"`
	StorageKey   string    `json:"-"`
	SizeBytes    int64     `json:"size_bytes"`
	ContentType  string    `json:"content_type"`
	CreatedAt    time.Time `json:"created_at"`
	FolderID     *int64    `json:"folder_id"`
}

type Folder struct {
	FolderID   int64     `json:"folder_id"`
	ParentID   *int64    `json:"parent_id"`
	OwnerID    int64     `json:"-"`
	FolderName string    `json:"folder_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type User struct {
	UserID       int64     `json:"user_id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type ContentInFolder struct {
	Files   []File   `json:"files"`
	Folders []Folder `json:"folders"`
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
			"VALUES ($1, $2, $3, $4, $5, $6) RETURNING id", file.OwnerID, file.OriginalName, file.StorageKey, file.SizeBytes, file.ContentType, file.FolderID).Scan(&id)
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

func (repo *FileRepository) Update(ctx context.Context, file_id int64, newFilename string, ownerID int64) error {
	fmt.Println(newFilename)
	_, err := repo.pool.Exec(ctx,
		"UPDATE files SET original_name = $1 WHERE id = $2 AND owner_id = $3", newFilename, file_id, ownerID)
	if err != nil {
		return fmt.Errorf("Error occured while updating^ %w", err)
	}
	return nil
}

func (repo *FileRepository) GetByID(ctx context.Context, id int64) (*File, error) {
	var file File
	file_row := repo.pool.QueryRow(ctx, "select * from files where id = $1", id)
	err := file_row.Scan(&file.ID, &file.OwnerID, &file.OriginalName, &file.StorageKey, &file.SizeBytes, &file.ContentType, &file.CreatedAt, &file.FolderID)
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
		err := rows.Scan(&f.ID, &f.OwnerID, &f.OriginalName, &f.StorageKey, &f.SizeBytes, &f.ContentType, &f.CreatedAt, &f.FolderID)
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

func (repo *FileRepository) ChangeFolder(ctx context.Context, fileID int64, newfolderID *int64, ownerID int64) error {
	_, err := repo.pool.Exec(ctx, "UPDATE files SET folder_id = $1 WHERE id = $2 AND owner_id = $3", newfolderID, fileID, ownerID)
	if err != nil {
		return fmt.Errorf("Invalid parameters for update: %v", err)
	}
	return nil
}

// Создание пользователем кастомных папок
// (не влияет на структуру хранения файлов на сервере, визуал для пользака)

func (repo *FileRepository) AddFolder(ctx context.Context, folder *Folder) (int64, error) {
	var FolderID int64
	err := repo.pool.QueryRow(ctx, "INSERT INTO folders (parent_id, owner_id,folder_name) "+
		"VALUES ($1, $2, $3) RETURNING folder_id", folder.ParentID, folder.OwnerID, folder.FolderName).Scan(&FolderID)
	if err != nil {
		return 0, fmt.Errorf("U have a problem with sql query %w", err)
	}
	return FolderID, nil
}

func (repo *FileRepository) ListFolders(ctx context.Context, ownerID int64) ([]Folder, error) {
	result := []Folder{}

	rows, err := repo.pool.Query(ctx, "SELECT folder_id, parent_id, owner_id, folder_name, created_at FROM folders "+
		"WHERE owner_id = $1", ownerID)
	if err != nil {
		return []Folder{}, fmt.Errorf("there is problem with sql-query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var f Folder
		err := rows.Scan(&f.FolderID, &f.ParentID, &f.OwnerID, &f.FolderName, &f.CreatedAt)
		if err != nil {
			return []Folder{}, fmt.Errorf("Error in fetching data from cursor: %w", err)
		}
		result = append(result, f)
	}

	return result, nil
}

func (repo *FileRepository) ListOfContentsInFolder(ctx context.Context, folderID int64, ownerID int64) (*ContentInFolder, error) {
	var Content ContentInFolder

	filerows, err := repo.pool.Query(ctx, "select original_name, id, storage_key, size_bytes, content_type, created_at, folder_id "+
		"FROM files WHERE folder_id = $1 AND owner_id = $2", folderID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("some error occured in sql query %w", err)
	}

	defer filerows.Close()
	for filerows.Next() {
		var f File
		err := filerows.Scan(&f.OriginalName, &f.ID, &f.StorageKey, &f.SizeBytes, &f.ContentType, &f.CreatedAt, &f.FolderID)
		if err != nil {
			return nil, fmt.Errorf("some error while fetching files data: %w", err)
		}

		Content.Files = append(Content.Files, f)
	}

	if err := filerows.Err(); err != nil {
		return nil, fmt.Errorf("iterating files: %w", err)
	}

	folderrows, err1 := repo.pool.Query(ctx, "select * from folders where parent_id = $1 AND owner_id = $2", folderID, ownerID)
	if err1 != nil {
		return nil, fmt.Errorf("error in sql query folders: %w", err1)
	}

	defer folderrows.Close()

	for folderrows.Next() {
		var f Folder
		err := folderrows.Scan(&f.FolderID, &f.ParentID, &f.OwnerID, &f.FolderName, &f.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("some error while fetching files data: %w", err)
		}
		Content.Folders = append(Content.Folders, f)
	}

	if err := folderrows.Err(); err != nil {
		return nil, fmt.Errorf("iterating folders: %w", err)
	}

	return &Content, nil
}

func (repo *FileRepository) RenameFolder(ctx context.Context, folderID int64, newFolderName string, ownerID int64) error {
	_, err := repo.pool.Exec(ctx, " UPDATE folders SET folder_name = $1 WHERE folder_id = $2 AND owner_id = $3", newFolderName, folderID, ownerID)
	if err != nil {
		return fmt.Errorf("Impossible to update folder^ %w", err)
	}

	return nil
}

func (repo *FileRepository) DeleteFolder(ctx context.Context, folderID int64, ownerID int64) error {
	_, err1 := repo.pool.Exec(ctx, "DELETE FROM folders WHERE folder_id = $1 and owner_id = $2", folderID, ownerID)
	if err1 != nil {
		return fmt.Errorf("error, impossible to delete folder: %w", err1)
	}

	return nil
}

// пользовательское взаимодействие

func (repo *FileRepository) CreateUser(ctx context.Context, username string, passwordHash string) (int64, error) {
	var userID int64
	err := repo.pool.QueryRow(ctx, "INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id", username, passwordHash).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("Error in DB while creating user: %w", err)
	}

	return userID, nil
}

func (repo *FileRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var U User
	row := repo.pool.QueryRow(ctx, "select * from users where username = $1", username)
	err := row.Scan(&U.UserID, &U.Username, &U.PasswordHash, &U.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("Some error in query^ %w", err)
	}
	return &U, nil
}
