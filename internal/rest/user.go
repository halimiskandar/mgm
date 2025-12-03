package rest

import (
	"context"
	"fmt"
	"myGreenMarket/domain"
	"myGreenMarket/pkg/logger"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type UserService interface {
	Register(ctx context.Context, user *domain.User) (domain.User, error)
	Login(ctx context.Context, email, password, ipAddress, userAgent string) (string, domain.User, error)
	ValidateTokenFromRedis(ctx context.Context, token string) (string, error)
	RefreshToken(ctx context.Context, oldToken, ipAddress, userAgent string) (string, domain.User, error)
	Logout(ctx context.Context, userID uint, token string) error
	VerifyEmail(ctx context.Context, verificationCodeEncrypt string) (err error)
	GetUserByID(ctx context.Context, id uint) (domain.User, error)
	GetAllUsers(ctx context.Context) ([]domain.User, error)
	UpdateUser(ctx context.Context, id uint, updateData *domain.User) (domain.User, error)
	DeleteUser(ctx context.Context, id uint) error
}

type UserHandler struct {
	userService UserService
	validator   *validator.Validate
	timeout     time.Duration
}

func NewUserHandler(userService UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		validator:   validator.New(),
		timeout:     10 * time.Second,
	}
}

type UserRegisterRequest struct {
	FullName string `json:"full_name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UserUpdateRequest struct {
	FullName string `json:"full_name,omitempty"`
	Password string `json:"password,omitempty" validate:"omitempty,min=6"`
}

type RefreshTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

// ResponseError represent the response error struct
type ResponseError struct {
	Message string `json:"message"`
}

func (h *UserHandler) Register(c echo.Context) error {
	var reqUser UserRegisterRequest

	if err := c.Bind(&reqUser); err != nil {
		logger.Error("Invalid request body", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validator.Struct(&reqUser); err != nil {
		logger.Error("Failed to validation user register", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	user, err := h.userService.Register(ctx, &domain.User{
		FullName: reqUser.FullName,
		Email:    reqUser.Email,
		Password: reqUser.Password,
	})
	if err != nil {
		logger.Error("Failed to register user", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Registration successful. Please check your email to verify your account.",
		"user":    user,
	})
}

func (h *UserHandler) Login(c echo.Context) error {
	var reqUser UserLoginRequest

	if err := c.Bind(&reqUser); err != nil {
		logger.Error("Failed to bind request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validator.Struct(&reqUser); err != nil {
		logger.Error("Failed to validate user login", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	// get ip address and user agent
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	token, user, err := h.userService.Login(ctx, reqUser.Email, reqUser.Password, ipAddress, userAgent)
	if err != nil {
		logger.Error("Failed to login with user", err)
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Login successful",
		"token":   token,
		"user":    user,
	})
}

// Logout handles user logout by invalidating token
func (h *UserHandler) Logout(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	// Get user_id from context (set by auth middleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		logger.Error("Failed to get user_id from context")
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: "unauthorized"})
	}

	// Get token from context (set by auth middleware)
	token, ok := c.Get("token").(string)
	if !ok {
		logger.Error("Failed to get token from context")
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: "unauthorized"})
	}

	err := h.userService.Logout(ctx, userID, token)
	if err != nil {
		logger.Error("Failed to logout user", err)
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Logout successful",
	})
}

// RefreshToken used for refresh user token
func (h *UserHandler) RefreshToken(c echo.Context) error {
	var req RefreshTokenRequest

	if err := c.Bind(&req); err != nil {
		logger.Error("Invalid request body", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validator.Struct(&req); err != nil {
		logger.Error("Failed to validate refresh token request", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	// get ip address and user agent
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	newToken, user, err := h.userService.RefreshToken(ctx, req.Token, ipAddress, userAgent)
	if err != nil {
		logger.Error("Failed to refresh token", err)
		return c.JSON(http.StatusUnauthorized, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Token refreshed successfully",
		"token":   newToken,
		"user":    user,
	})
}

func (h *UserHandler) VerifyEmail(c echo.Context) error {
	encCode := c.Param("code")

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	err := h.userService.VerifyEmail(ctx, encCode)
	if err != nil {
		if strings.Contains(err.Error(), "invalid or expired") {
			return c.JSON(http.StatusUnauthorized, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, "Successfully verified email")
}

// GetUserByID handles getting a user by ID
func (h *UserHandler) GetUserByID(c echo.Context) error {
	id := c.Param("id")

	// Convert string ID to uint
	var userID uint
	if _, err := fmt.Sscan(id, &userID); err != nil {
		logger.Error("Invalid user ID", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: "invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	user, err := h.userService.GetUserByID(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User retrieved successfully",
		"user":    user,
	})
}

// GetAllUsers handles getting all users
func (h *UserHandler) GetAllUsers(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	users, err := h.userService.GetAllUsers(ctx)
	if err != nil {
		logger.Error("Failed to get all users", err)
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Users retrieved successfully",
		"users":   users,
	})
}

// UpdateUser handles updating a user
func (h *UserHandler) UpdateUser(c echo.Context) error {
	id := c.Param("id")

	// Convert string ID to uint
	var userID uint
	if _, err := fmt.Sscan(id, &userID); err != nil {
		logger.Error("Invalid user ID", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: "invalid user ID"})
	}

	var reqUpdate UserUpdateRequest
	if err := c.Bind(&reqUpdate); err != nil {
		logger.Error("Invalid request body", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	if err := h.validator.Struct(&reqUpdate); err != nil {
		logger.Error("Failed to validate user update", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	updateData := &domain.User{
		FullName: reqUpdate.FullName,
		Password: reqUpdate.Password,
	}

	updatedUser, err := h.userService.UpdateUser(ctx, userID, updateData)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, ResponseError{Message: err.Error()})
		}
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "invalid") {
			return c.JSON(http.StatusBadRequest, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User updated successfully",
		"user":    updatedUser,
	})
}

// DeleteUser handles deleting a user
func (h *UserHandler) DeleteUser(c echo.Context) error {
	id := c.Param("id")

	// Convert string ID to uint
	var userID uint
	if _, err := fmt.Sscan(id, &userID); err != nil {
		logger.Error("Invalid user ID", err)
		return c.JSON(http.StatusBadRequest, ResponseError{Message: "invalid user ID"})
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), h.timeout)
	defer cancel()

	err := h.userService.DeleteUser(ctx, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.JSON(http.StatusNotFound, ResponseError{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, ResponseError{Message: err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User deleted successfully",
	})
}
