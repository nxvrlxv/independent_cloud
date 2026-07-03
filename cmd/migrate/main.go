package main

import (
	"log"

	"github.com/nxvrlxv/independent_cloud/internal/migrator"
)

func main() {
	dbURL := "postgres://cloud:secret@localhost:5432/cloud?sslmode=disable"
	if err := migrator.Run(dbURL); err != nil {
		log.Fatal(err)
	}
	log.Println("migrations applied")
}
