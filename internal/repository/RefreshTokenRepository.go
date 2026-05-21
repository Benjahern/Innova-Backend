package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type RefreshTokenRepository interface {
	Create(token *models.RefreshToken) error
	GetByToken(token string) (*models.RefreshToken, error)
	Delete(tokenID string) error
	DeleteByUser(userID string) error
}

type RefreshTokenRepositoryImpl struct {
	db *db.DB
}

func NewRefreshTokenRepository(database *db.DB) *RefreshTokenRepositoryImpl {
	return &RefreshTokenRepositoryImpl{db: database}
}

func (r *RefreshTokenRepositoryImpl) Create(token *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (token_id, user_id, token, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Pool.Exec(context.Background(), query,
		token.TokenID, token.UserID, token.Token, token.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("CreateRefreshToken: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepositoryImpl) GetByToken(token string) (*models.RefreshToken, error) {
	query := `SELECT token_id, user_id, token, expires_at FROM refresh_tokens WHERE token = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, token)

	var rt models.RefreshToken
	err := row.Scan(&rt.TokenID, &rt.UserID, &rt.Token, &rt.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("GetRefreshTokenByToken: %w", err)
	}
	return &rt, nil
}

func (r *RefreshTokenRepositoryImpl) Delete(tokenID string) error {
	query := `DELETE FROM refresh_tokens WHERE token_id = $1`
	_, err := r.db.Pool.Exec(context.Background(), query, tokenID)
	if err != nil {
		return fmt.Errorf("DeleteRefreshToken: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepositoryImpl) DeleteByUser(userID string) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.db.Pool.Exec(context.Background(), query, userID)
	if err != nil {
		return fmt.Errorf("DeleteRefreshTokensByUser: %w", err)
	}
	return nil
}