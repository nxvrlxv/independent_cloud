package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://cloud:secret@localhost:5432/cloud?sslmode=disable")
	if err != nil {
		log.Fatalf("Error to create connection pool^ %v", err)
	}

	defer pool.Close()
}
