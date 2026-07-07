package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepo(t *testing.T) {
	ctx := context.Background()
	dbURL := "postgres://cloud:secret@localhost:5432/cloud?sslmode=disable"
	pgpool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to create a pool: %v", err)
	}

	defer pgpool.Close()

	TestFileRepository := NewFileRepository(pgpool)
	storage_key := uuid.New().String()
	timing := time.Now()
	my_file := &File{OwnerID: 1, OriginalName: "otchet.pdf", StorageKey: storage_key, SizeBytes: 666666, ContentType: "something", CreatedAt: timing}

	id, err1 := TestFileRepository.Save(ctx, my_file)
	if err1 != nil {
		t.Fatalf("Error when saving file: %v", err1)
	}
	if id > 0 {
		my_file.ID = id
	} else {
		t.Fatalf("Expected id got 0")
	}
	t.Cleanup(func() {
		TestFileRepository.Delete(context.Background(), my_file.ID)
	})

	listFiles, err := TestFileRepository.ListByOwner(ctx, 1)
	if err != nil {
		t.Fatalf("Error list of files: %v", err)
	}
	fmt.Println("There is a list of files that owned by this user:", listFiles)

	newFile, err := TestFileRepository.GetByID(ctx, my_file.ID)
	if err != nil {
		t.Fatalf("error while geting file %v", err)
	}
	fmt.Println("My File: ", my_file)
	fmt.Println("New file: ", newFile)

	err2 := TestFileRepository.Delete(ctx, newFile.ID)
	if err2 != nil {
		t.Fatalf("Error %v", err2)
	}
	_, err = TestFileRepository.GetByID(ctx, my_file.ID)
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("expected ErrFileNotFound after delete, got %v", err)
	}
	fmt.Println("Successfully deleted a file")

}
