package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenData struct {
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	Token     string    `json:"token"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}

type TokenRepository struct {
	client *redis.Client
}

func NewTokenRepository(client *redis.Client) *TokenRepository {
	return &TokenRepository{
		client: client,
	}
}

func (r *TokenRepository) StoreToken(ctx context.Context, userID, token string, data TokenData, ttl time.Duration) error {
	// key format: "token:{user_id}:{token_id}"
	key := fmt.Sprintf("token:user:%s", userID)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	err = r.client.Set(ctx, key, jsonData, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store token in Redis: %w", err)
	}

	// store a reverse lookup token -> user_id for quick validation
	tokenKey := fmt.Sprintf("token:lookup:%s", token)
	err = r.client.Set(ctx, tokenKey, userID, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store token lookup: %w", err)
	}

	return nil
}

// GetTokenData retrieve token data by user ID
func (r *TokenRepository) GetTokenData(ctx context.Context, userID string) (*TokenData, error) {
	key := fmt.Sprintf("token:user:%s", userID)

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("token not found")
		}
		return nil, fmt.Errorf("failed to get token from Redis: %w", err)
	}

	var tokenData TokenData
	err = json.Unmarshal([]byte(val), &tokenData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	return &tokenData, nil
}

// ValidateToken checks if a token exists and is valid
func (r *TokenRepository) ValidateToken(ctx context.Context, token string) (string, error) {
	tokenKey := fmt.Sprintf("token:lookup:%s", token)

	userID, err := r.client.Get(ctx, tokenKey).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errors.New("token not found or expired")
		}
		return "", fmt.Errorf("failed to validate token: %w", err)
	}

	return userID, nil
}

// RefreshTokenTTL extends the token expiration time
func (r *TokenRepository) RefreshTokenTTL(ctx context.Context, userID string, newTTL time.Duration) error {
	key := fmt.Sprintf("token:user:%s", userID)

	// check if exists
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check token existence: %w", err)
	}

	if exists == 0 {
		return errors.New("token not found")
	}

	// update TTL
	err = r.client.Expire(ctx, key, newTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to refresh token TTL: %w", err)
	}

	// update the lookup key TTL
	tokenData, err := r.GetTokenData(ctx, userID)
	if err != nil {
		return err
	}

	tokenKey := fmt.Sprintf("token:lookup:%s", tokenData.Token)
	err = r.client.Expire(ctx, tokenKey, newTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to refresh lookup TTL: %w", err)
	}

	return nil
}
