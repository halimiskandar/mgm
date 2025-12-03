package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"myGreenMarket/domain"
	"myGreenMarket/internal/repository/redis"
	"myGreenMarket/pkg/logger"
	"myGreenMarket/pkg/utils"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pobyzaarif/goshortcute"
)

// UserRepository contract interface
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id uint) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindAll(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uint) error
	UpdateEmailVerification(ctx context.Context, id uint, isVerified bool) error
}

// NotificationRepository contract interface
type NotificationRepository interface {
	SendEmail(toName, toEmail, subject, message string) (err error)
}

type TokenRepository interface {
	StoreToken(ctx context.Context, userID, token string, data redis.TokenData, ttl time.Duration) error
	GetTokenData(ctx context.Context, userID string) (*redis.TokenData, error)
	ValidateToken(ctx context.Context, token string) (string, error)
	RefreshTokenTTL(ctx context.Context, userID string, newTTL time.Duration) error
	BlaclistToken(ctx context.Context, token string, ttl time.Duration) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
	DeleteToken(ctx context.Context, userID, token string) error
}

type userService struct {
	userRepo                UserRepository
	tokenRepo               TokenRepository
	validate                *validator.Validate
	notifRepo               NotificationRepository
	appEmailVerificationKey string
	appDeploymentUrl        string
}

const (
	verificationCodeTTL      = 5
	TokenExpiryDuration      = 24 * time.Hour
	SubjectRegisterAccount   = "Activate Your Account!"
	EmailBodyRegisterAccount = `Halo, %v, aktivasi akun anda dengan membuka tautan dibawah</br></br>%v</br>catatan: link hanya berlaku %v menit`
)

func NewUserService(
	userRepo UserRepository,
	tokenRepo TokenRepository,
	validate *validator.Validate,
	notifRepo NotificationRepository,
	appEmailVerificationKey string,
	appDeploymentUrl string,
) *userService {
	return &userService{
		userRepo:                userRepo,
		tokenRepo:               tokenRepo,
		validate:                validate,
		notifRepo:               notifRepo,
		appEmailVerificationKey: appEmailVerificationKey,
		appDeploymentUrl:        appDeploymentUrl,
	}
}

const (
	RoleCustomer = "customer"
	RoleAdmin    = "admin"
)

var validRoles = map[string]bool{
	RoleCustomer: true,
	RoleAdmin:    true,
}

func (s *userService) Register(ctx context.Context, user *domain.User) (domain.User, error) {
	if err := s.validate.Var(user.Email, "required,email"); err != nil {
		logger.Error("Invalid email format", err)
		return domain.User{}, errors.New("invalid email format")
	}

	if err := s.validate.Var(user.Password, "required,min=6"); err != nil {
		logger.Error("Invalid user password", err)
		return domain.User{}, errors.New("password must be at least 6 characters")
	}

	// Check if email already exists
	existingUser, err := s.userRepo.FindByEmail(ctx, user.Email)
	if err == nil && existingUser.ID > 0 {
		logger.Error("Email already exists")
		return domain.User{}, errors.New("email already exists")
	}

	passwordHash, err := utils.HashPassword(user.Password)
	if err != nil {
		logger.Error("Failed to hash password", err)
		return domain.User{}, errors.New("failed to hash password")
	}

	newUser := domain.User{
		FullName:   user.FullName,
		Email:      user.Email,
		Password:   string(passwordHash),
		IsVerified: false,
		Role:       "customer",
	}

	if err := s.userRepo.Create(ctx, &newUser); err != nil {
		logger.Error("Failed to create new user")
		return domain.User{}, err
	}

	timeNow := time.Now()
	expAt := timeNow.Add(time.Duration(time.Minute * verificationCodeTTL)).Unix()

	verificationCode := fmt.Sprintf("%v|%v", newUser.Email, expAt)
	verificationCodeEncrypt, err := goshortcute.AESCBCEncrypt([]byte(verificationCode), []byte(s.appEmailVerificationKey))
	if err != nil {
		logger.Fatal("error when encrypt")
	}
	strEncode := goshortcute.StringtoBase64Encode(verificationCodeEncrypt)
	activationLink := s.appDeploymentUrl + "/api/v1/users/email-verification/" + strEncode

	err = s.notifRepo.SendEmail(newUser.FullName, newUser.Email, SubjectRegisterAccount, fmt.Sprintf(EmailBodyRegisterAccount, newUser.FullName, activationLink, verificationCodeTTL))
	if err != nil {
		logger.Warn("Failed to send verification email", err)
	}

	newUser.Password = ""
	return newUser, nil
}

func (s *userService) Login(ctx context.Context, email, password, ipAddress, userAgent string) (string, domain.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		logger.Error("Invalid user credentials", err)
		return "", domain.User{}, err
	}

	ok := utils.CheckPassword(password, user.Password)
	if !ok {
		logger.Error("User password incorrect", err)
		return "", domain.User{}, errors.New("incorrect password")
	}

	if !user.IsVerified {
		logger.Error("Email address has not been verified", err)
		return "", domain.User{}, errors.New("email address has not been verified")
	}

	userIdStr := strconv.FormatUint(uint64(user.ID), 10)
	token, err := utils.GenerateJWT(userIdStr, user.Role)
	if err != nil {
		logger.Error("Failed to generated token", err)
		return "", domain.User{}, errors.New("failed to generate token")
	}

	// Store token in Redis
	tokenData := redis.TokenData{
		UserID:    userIdStr,
		Role:      user.Role,
		Token:     token,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(TokenExpiryDuration),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	if err := s.tokenRepo.StoreToken(ctx, userIdStr, token, tokenData, TokenExpiryDuration); err != nil {
		logger.Error("Failed to store token in Redis", err)
		// Log the error if fail
		logger.Warn("Continuing without Redis token storage")
	}

	user.Password = ""
	return token, user, nil
}

func (s *userService) ValidateTokenFromRedis(ctx context.Context, token string) (string, error) {
	// Check if token is blacklisted
	isBlacklisted, err := s.tokenRepo.IsTokenBlacklisted(ctx, token)
	if err != nil {
		logger.Error("Failed to check token blacklist", err)
		return "", errors.New("failed to validate token")
	}

	if isBlacklisted {
		return "", errors.New("token has been invalidated")
	}

	// validate token exists in redis
	userID, err := s.tokenRepo.ValidateToken(ctx, token)
	if err != nil {
		logger.Error("Token not found in Redis", err)
		return "", errors.New("invalid or expired token")
	}

	return userID, nil
}

func (s *userService) RefreshToken(ctx context.Context, oldToken, ipAddress, userAgent string) (string, domain.User, error) {
	// validate old token
	userID, err := s.ValidateTokenFromRedis(ctx, oldToken)
	if err != nil {
		return "", domain.User{}, err
	}

	// get user data
	userIDUint, _ := strconv.ParseUint(userID, 10, 64)
	user, err := s.userRepo.FindByID(ctx, uint(userIDUint))
	if err != nil {
		logger.Error("User not found", err)
		return "", domain.User{}, errors.New("user not found")
	}

	// generate new token
	newToken, err := utils.GenerateJWT(userID, user.Role)
	if err != nil {
		logger.Error("Failed to generate new token", err)
		return "", domain.User{}, errors.New("failed to generate token")
	}

	// Blacklist old token first to prevent reuse
	if err := s.tokenRepo.BlaclistToken(ctx, oldToken, TokenExpiryDuration); err != nil {
		logger.Warn("Failed to blacklist old token", err)
	}

	// Then delete old token from Redis
	if err := s.tokenRepo.DeleteToken(ctx, userID, oldToken); err != nil {
		logger.Warn("Failed to delete old token", err)
	}

	// store new token
	tokenData := redis.TokenData{
		UserID:    userID,
		Role:      user.Role,
		Token:     newToken,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(TokenExpiryDuration),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	if err := s.tokenRepo.StoreToken(ctx, userID, newToken, tokenData, TokenExpiryDuration); err != nil {
		logger.Error("Failed to store new token", err)
		return "", domain.User{}, errors.New("failed to refresh token")
	}

	user.Password = ""
	return newToken, user, nil
}

// Logout invalidates user token and removes it from Redis
func (s *userService) Logout(ctx context.Context, userID uint, token string) error {
	userIDStr := strconv.FormatUint(uint64(userID), 10)

	// Blacklist the token to prevent reuse
	if err := s.tokenRepo.BlaclistToken(ctx, token, TokenExpiryDuration); err != nil {
		logger.Error("Failed to blacklist token during logout", err)
		return errors.New("failed to logout")
	}

	// Delete token from Redis
	if err := s.tokenRepo.DeleteToken(ctx, userIDStr, token); err != nil {
		logger.Error("Failed to delete token during logout", err)
		// Don't return error here, token is already blacklisted
		logger.Warn("Token blacklisted but not deleted from Redis")
	}

	return nil
}

func (s *userService) VerifyEmail(ctx context.Context, verificationCodeEncrypt string) error {
	strDecode := goshortcute.StringtoBase64Decode(verificationCodeEncrypt)
	verificationCodeDecrypt, err := goshortcute.AESCBCDecrypt([]byte(strDecode), []byte(s.appEmailVerificationKey))
	if err != nil {
		logger.Error("Verifying email error", err)
		return errors.New("invalid or expired url")
	}

	verificationCode := strings.Split(verificationCodeDecrypt, "|")
	if len(verificationCode) != 2 {
		logger.Error("Verifying email error", verificationCodeDecrypt)
		return errors.New("invalid or expired url")
	}

	email := verificationCode[0]
	expAtStr := verificationCode[1]

	ts, err := strconv.ParseInt(expAtStr, 10, 64)
	if err != nil {
		logger.Error("Verifying email error", verificationCodeDecrypt)
		return errors.New("invalid or expired url")
	}
	expAt := time.Unix(ts, 0)
	if time.Now().After(expAt) {
		return errors.New("invalid or expired url")
	}

	getUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		logger.Error("Verifying email error", err)
		return errors.New("failed to get user by email")
	}

	if getUser.IsVerified {
		logger.Warn("verify email err", slog.Any("err", "email verified already"))
		return errors.New("invalid or expired url")
	}

	getUser.IsVerified = true

	if err := s.userRepo.UpdateEmailVerification(ctx, getUser.ID, true); err != nil {
		logger.Error("Verify email err", err)
		return err
	}

	return nil
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(ctx context.Context, id uint) (domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		logger.Error("Failed to get user by ID", err)
		return domain.User{}, err
	}

	user.Password = ""
	return user, nil
}

// GetAllUsers retrieves all users
func (s *userService) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	users, err := s.userRepo.FindAll(ctx)
	if err != nil {
		logger.Error("Failed to get all users", err)
		return nil, err
	}

	for i := range users {
		users[i].Password = ""
	}

	return users, nil
}

// UpdateUser updates user information
func (s *userService) UpdateUser(ctx context.Context, id uint, updateData *domain.User) (domain.User, error) {
	existingUser, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		logger.Error("User not found for update", err)
		return domain.User{}, err
	}

	if updateData.FullName != "" {
		existingUser.FullName = updateData.FullName
	}

	if updateData.Password != "" {
		// Validate password
		if err := s.validate.Var(updateData.Password, "required,min=6"); err != nil {
			logger.Error("Invalid password", err)
			return domain.User{}, errors.New("password must be at least 6 characters")
		}

		// Hash new password
		passwordHash, err := utils.HashPassword(updateData.Password)
		if err != nil {
			logger.Error("Failed to hash password", err)
			return domain.User{}, errors.New("failed to hash password")
		}
		existingUser.Password = string(passwordHash)
	}

	// Update in database
	if err := s.userRepo.Update(ctx, &existingUser); err != nil {
		logger.Error("Failed to update user", err)
		return domain.User{}, err
	}

	existingUser.Password = ""
	return existingUser, nil
}

// DeleteUser soft deletes a user
func (s *userService) DeleteUser(ctx context.Context, id uint) error {
	_, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		logger.Error("User not found for deletion", err)
		return err
	}

	// Delete user
	if err := s.userRepo.Delete(ctx, id); err != nil {
		logger.Error("Failed to delete user", err)
		return err
	}

	return nil
}
