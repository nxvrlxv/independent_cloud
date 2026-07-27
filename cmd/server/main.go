package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nxvrlxv/independent_cloud/internal/handler"
	"github.com/nxvrlxv/independent_cloud/internal/repository"
	"github.com/nxvrlxv/independent_cloud/internal/service"
	"github.com/nxvrlxv/independent_cloud/internal/storage"
)

const base string = "C:/Users/ElectroN1ck/mycloud"

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://cloud:secret@localhost:5432/cloud?sslmode=disable")
	if err != nil {
		log.Fatalf("Error to create connection pool^ %v", err)
	}

	defer pool.Close()
	port := ":6767"
	//создание всех сущностей
	lstorage := storage.NewLocalStorage(base)
	repo := repository.NewFileRepository(pool)
	serv := service.NewFileService(lstorage, repo)
	hand := handler.NewFileHandler(serv)
	mux := http.NewServeMux()

	//mux.HandleFunc("GET /hello", hand.FirstHandler)
	mux.HandleFunc("POST /save", hand.Upload)
	mux.HandleFunc("GET /file/{id}", hand.Download)
	mux.HandleFunc("PATCH /file/{id}", hand.Update)
	mux.HandleFunc("DELETE /file/{id}", hand.Delete)
	mux.HandleFunc("POST /folders", hand.AddFolder)
	mux.HandleFunc("GET /folders", hand.ListFolders)
	mux.HandleFunc("GET /folders/{id}/contents", hand.ListOfFilesInFolder)
	mux.HandleFunc("DELETE /folders/{id}", hand.DeleteFolder)
	mux.HandleFunc("PATCH /folders/{id}", hand.RenameFolder)

	//s := http.Server{}
	err1 := http.ListenAndServe(port, mux)
	if err1 != nil {
		fmt.Println(err1)
		return
	}

}
