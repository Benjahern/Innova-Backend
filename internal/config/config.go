package config

import (
	"os"
	"time"
	"strconv"
)

type Config struct {
	// Add your configuration fields here
	ServerPort int
	ServerHost string
	DatabaseURL string
	JWTSecret string
	JWTExpiration time.Duration
	JWRefreshExpiration time.Duration

}

func LoadConfig() (*Config, error) {

	c := &Config{}

	c.ServerHost = os.Getenv("SERVER_HOST")
	serverPort, err := strconv.Atoi(os.Getenv("SERVER_PORT"))
	if err != nil {
		return nil, err
	}
	c.ServerPort = serverPort
	c.DatabaseURL = os.Getenv("DATABASE_URL")
	c.JWTSecret = os.Getenv("JWT_SECRET")
	jwtExp, err := time.ParseDuration(os.Getenv("JWT_ACCESS_EXPIRY"))
	if err != nil {
		return nil, err
	}
	c.JWTExpiration = jwtExp

	refreshExp, err := time.ParseDuration(os.Getenv("JWT_REFRESH_EXPIRATION"))
	if err != nil {
		return nil, err
	}
	c.JWRefreshExpiration = refreshExp

	return c, nil
}