package middleware

import (
	"context"
	"myGreenMarket/pkg/logger"
	"myGreenMarket/pkg/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	jsonres "myGreenMarket/pkg/response"

	"github.com/labstack/echo/v4"
)

// TokenValidator interface untuk validasi token dari Redis
type TokenValidator interface {
	ValidateTokenFromRedis(ctx context.Context, token string) (string, error)
}

// AuthMiddleware basic JWT authentication tanpa Redis
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, jsonres.Error(
					"UNAUTHORIZED", "Missing authorization header", nil,
				))
			}

			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, jsonres.Error(
					"UNAUTHORIZED", "Invalid authorization format", nil,
				))
			}

			tokenString := tokenParts[1]

			claims, err := utils.ParseJWT(tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, jsonres.Error(
					"UNAUTHORIZED", "Invalid token", nil,
				))
			}

			expAt, err := claims.GetExpirationTime()
			if err != nil {
				return c.JSON(http.StatusForbidden, jsonres.Error(
					"FORBIDDEN", "Status Forbidden", nil,
				))
			}

			if time.Now().After(expAt.Time) {
				return c.JSON(http.StatusForbidden, jsonres.Error(
					"FORBIDDEN", "Status Forbidden", nil,
				))
			}

			userIDUint, err := strconv.ParseUint(claims.UserID, 10, 64)
			if err != nil {
				logger.Error("Invalid user ID in token", err)
				return c.JSON(http.StatusForbidden, jsonres.Error(
					"FORBIDDEN", "Invalid user ID in token", nil,
				))
			}

			c.Set("user_id", uint(userIDUint))
			c.Set("role", claims.Role)
			c.Set("token", tokenString)

			return next(c)
		}
	}
}

// AuthMiddlewareWithRedis JWT authentication dengan validasi Redis
func AuthMiddlewareWithRedis(tokenValidator TokenValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, jsonres.Error(
					"UNAUTHORIZED", "Missing authorization header", nil,
				))
			}

			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, jsonres.Error(
					"UNAUTHORIZED", "Invalid authorization format", nil,
				))
			}

			tokenString := tokenParts[1]

			// Parse JWT
			claims, err := utils.ParseJWT(tokenString)
			if err != nil {
				logger.Error("Failed to parse JWT", err)
				return c.JSON(http.StatusUnauthorized, jsonres.Error(
					"UNAUTHORIZED", "Invalid token", nil,
				))
			}

			// Validasi expiration
			expAt, err := claims.GetExpirationTime()
			if err != nil {
				return c.JSON(http.StatusForbidden, jsonres.Error(
					"FORBIDDEN", "Status Forbidden", nil,
				))
			}

			if time.Now().After(expAt.Time) {
				return c.JSON(http.StatusForbidden, jsonres.Error(
					"FORBIDDEN", "Token expired", nil,
				))
			}

			// Validasi token dari Redis
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			userID, err := tokenValidator.ValidateTokenFromRedis(ctx, tokenString)
			if err != nil {
				logger.Error("Token not found in Redis", err)
				return c.JSON(http.StatusUnauthorized, jsonres.Error(
					"UNAUTHORIZED", "Token expired or invalid", nil,
				))
			}

			// Validasi UserID match antara JWT dan Redis
			if userID != claims.UserID {
				logger.Error("UserID mismatch between JWT and Redis")
				return c.JSON(http.StatusUnauthorized, jsonres.Error(
					"UNAUTHORIZED", "Invalid token", nil,
				))
			}

			userIDUint, err := strconv.ParseUint(claims.UserID, 10, 64)
			if err != nil {
				logger.Error("Invalid user ID in token", err)
				return c.JSON(http.StatusForbidden, jsonres.Error(
					"FORBIDDEN", "Invalid user ID in token", nil,
				))
			}

			c.Set("user_id", uint(userIDUint))
			c.Set("role", claims.Role)
			c.Set("token", tokenString)

			return next(c)
		}
	}
}

func AdminOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role := c.Get("role")
			roleStr, ok := role.(string)
			if !ok || strings.ToUpper(roleStr) != "ADMIN" {
				return c.JSON(http.StatusForbidden, jsonres.Error(
					"FORBIDDEN", "Admin access required", nil,
				))
			}

			return next(c)
		}
	}
}

func SelfOrAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			loggedInUserID, ok := c.Get("user_id").(uint)
			if !ok {
				return c.JSON(http.StatusUnauthorized, jsonres.Error(
					"UNAUTHORIZED", "User not authenticated", nil,
				))
			}

			role := c.Get("role")
			roleStr, ok := role.(string)
			if !ok {
				return c.JSON(http.StatusForbidden, jsonres.Error(
					"FORBIDDEN", "Invalid role", nil,
				))
			}

			// Jika admin, langsung izinkan akses semua resource
			if strings.ToUpper(roleStr) == "ADMIN" {
				return next(c)
			}

			// Jika bukan admin, check apakah ID di path sama dengan ID user yang login
			requestedID := c.Param("id")
			requestedIDUint, err := strconv.ParseUint(requestedID, 10, 64)
			if err != nil {
				return c.JSON(http.StatusBadRequest, jsonres.Error(
					"BAD_REQUEST", "Invalid user ID", nil,
				))
			}

			if uint(requestedIDUint) != loggedInUserID {
				return c.JSON(http.StatusForbidden, jsonres.Error(
					"FORBIDDEN", "You can only access your own data", nil,
				))
			}

			return next(c)
		}
	}
}
