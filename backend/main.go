package main

import (
	"context"
	"log"
	"time"

	"github.com/LaifuLiu/kubeai-training-platform/backend/db"
	"github.com/LaifuLiu/kubeai-training-platform/backend/registory"
	"github.com/LaifuLiu/kubeai-training-platform/backend/router"
)

func main() {
	dbCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := db.OpenDatabase(dbCtx)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	if err := registory.InitializeTrainingRunsSchema(dbCtx, db); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	log.Println("PostgreSQL connected; training_runs schema initialized")

	r := router.NewRouter(db)
	r.Run("8080")
}
