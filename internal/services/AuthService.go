package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"turno-papa/internal/config"
	"turno-papa/internal/models"
	"turno-papa/internal/repository"
)

type AuthService struct {
	userRepo       repository.UserRepository
	refreshRepo    repository.RefreshTokenRepository
	cfg            *config.Config
}

func NewAuthService(userRepo repository.UserRepository, refreshRepo repository.RefreshTokenRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		cfg:         cfg,
	}
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         *models.User
}

func (s *AuthService) Register(email, password, name, companyName string) (*AuthResult, error) {
	// Check if user exists
	existing, _ := s.userRepo.GetByEmail(email)
	if existing != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Create user
	user := &models.User{
		UserID:    uuid.New().String(),
		CompanyID: uuid.New().String(), // TODO: create or get company
		Name:      name,
		Email:     email,
		Password:  string(hashed),
		Rol:       "worker",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Generate tokens
	return s.generateAuthResult(user)
}

func (s *AuthService) Login(email, password string) (*AuthResult, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return s.generateAuthResult(user)
}

func (s *AuthService) Refresh(refreshToken string) (*AuthResult, error) {
	token, err := s.refreshRepo.GetByToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	expiresAt, err := time.Parse(time.RFC3339, token.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("invalid token format")
	}
	if expiresAt.Before(time.Now()) {
		s.refreshRepo.Delete(token.TokenID)
		return nil, fmt.Errorf("refresh token expired")
	}

	user, err := s.userRepo.GetByID(token.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Delete old refresh token
	s.refreshRepo.Delete(token.TokenID)

	// Generate new tokens
	return s.generateAuthResult(user)
}

func (s *AuthService) Logout(userID string) error {
	return s.refreshRepo.DeleteByUser(userID)
}

func (s *AuthService) generateAuthResult(user *models.User) (*AuthResult, error) {
	// Access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	// Refresh token
	refreshToken := uuid.New().String()
	expiresAt := time.Now().Add(s.cfg.JWTExpiration).Add(7 * 24 * time.Hour).Format(time.RFC3339)

	token := &models.RefreshToken{
		TokenID:   uuid.New().String(),
		UserID:    user.UserID,
		Token:     refreshToken,
		ExpiresAt: expiresAt,
	}

	if err := s.refreshRepo.Create(token); err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.JWTExpiration.Seconds()),
		User:         user,
	}, nil
}

func (s *AuthService) generateAccessToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":    user.UserID,
		"company_id": user.CompanyID,
		"role":       user.Rol,
		"branch_id":  user.BranchID,
		"exp":        time.Now().Add(s.cfg.JWTExpiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}