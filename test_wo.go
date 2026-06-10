package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/usecase"
)

func main() {
	dbURL := "postgres://postgres:postgres@localhost:15432/permatatex_inventory?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	repo := entity.New(pool)
	auditLogUC, err := usecase.NewAuditLogUseCase(repo)
	if err != nil {
		log.Fatalf("Failed to create audit log usecase: %v\n", err)
	}
	uc, err := usecase.NewWorkOrderProductionUseCase(repo, pool, auditLogUC)
	if err != nil {
		log.Fatalf("Failed to create usecase: %v\n", err)
	}

	header, err := uc.GetWorkOrderDetail(context.Background(), 4, nil)
	if err != nil {
		fmt.Printf("Error getting WO 4: %v\n", err)
	} else {
		fmt.Printf("WO 4 found: %+v\n", header)
	}
}
