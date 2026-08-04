package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nxvrlxv/independent_cloud/internal/auth"
	"github.com/nxvrlxv/independent_cloud/internal/handler"
	"github.com/nxvrlxv/independent_cloud/internal/repository"
	"github.com/nxvrlxv/independent_cloud/internal/service"
	"github.com/nxvrlxv/independent_cloud/internal/storage"
)

func main() {
	ctx := context.Background()

	err1 := godotenv.Load()
	if err1 != nil {
		log.Fatalf("Error to read env variables! %v", err1)
	}

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error to create connection pool^ %v", err)
	}

	defer pool.Close()

	port := os.Getenv("PORT")
	base := os.Getenv("STORAGE_PATH")
	jwt_sec := os.Getenv("JWT_SECRET")
	if jwt_sec == "" {
		log.Fatalf("Jwt secret is not set")
	}
	//создание всех сущностей
	lstorage := storage.NewLocalStorage(base)
	repo := repository.NewFileRepository(pool)
	serv := service.NewFileService(lstorage, repo)
	authenticator := auth.NewAuthService(repo, []byte(jwt_sec))
	hand := handler.NewFileHandler(serv, authenticator)
	mux := http.NewServeMux()

	//mux.HandleFunc("GET /hello", hand.FirstHandler)
	mux.HandleFunc("POST /save", authenticator.AuthMiddleware(hand.Upload))
	mux.HandleFunc("GET /file/{id}", authenticator.AuthMiddleware(hand.Download))
	mux.HandleFunc("PATCH /file/{id}", authenticator.AuthMiddleware(hand.Update))
	mux.HandleFunc("DELETE /file/{id}", authenticator.AuthMiddleware(hand.Delete))
	mux.HandleFunc("POST /folders", authenticator.AuthMiddleware(hand.AddFolder))
	mux.HandleFunc("GET /folders", authenticator.AuthMiddleware(hand.ListFolders))
	mux.HandleFunc("GET /userfiles", authenticator.AuthMiddleware(hand.ListUserFiles))
	mux.HandleFunc("GET /folders/{id}/contents", authenticator.AuthMiddleware(hand.ListOfContentsInFolder))
	mux.HandleFunc("DELETE /folders/{id}", authenticator.AuthMiddleware(hand.DeleteFolder))
	mux.HandleFunc("PATCH /folders/{id}", authenticator.AuthMiddleware(hand.RenameFolder))
	mux.HandleFunc("POST /register", hand.RegisterUser)
	mux.HandleFunc("POST /login", hand.LoginUser)
	mux.HandleFunc("POST /logout", hand.Logout)

	//s := http.Server{}
	err12 := http.ListenAndServe(port, mux)
	if err12 != nil {
		fmt.Println(err12)
		return
	}

}
