package models

type RefreshToken struct {
	TokenID   string `json:"token_id"`
	UserID    string `json:"user_id"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}