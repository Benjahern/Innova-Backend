package db

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
)

type DB struct {
	Pool *pgxpool.Pool
}

func Connect(databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	
	return &DB{Pool: pool}, nil
}


func (db *DB) Close() {
	db.Pool.Close()
}
