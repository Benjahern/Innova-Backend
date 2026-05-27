package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	connStr := "postgres://postgres:09282325@localhost:5432/gestion_turnos?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "SELECT company_id, name, config FROM companies")
	if err != nil {
		log.Fatalf("Query failed: %v\n", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		var config *string
		if err := rows.Scan(&id, &name, &config); err != nil {
			log.Fatalf("Scan failed: %v\n", err)
		}
		configStr := "NULL"
		if config != nil {
			configStr = *config
		}
		fmt.Printf("ID: %s\nName: %s\nConfig: %s\n\n", id, name, configStr)
	}
}
