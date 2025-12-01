package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"myGreenMarket/domain"
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

type userService struct {
	userRepo                UserRepository
	validate                *validator.Validate
	notifRepo               NotificationRepository
	appEmailVerificationKey string
	appDeploymentUrl        string
}

const (
	verificationCodeTTL      = 5
	SubjectRegisterAccount   = "Activate Your Account!"
	EmailBodyRegisterAccount = `Halo, %v, aktivasi akun anda dengan membuka tautan dibawah</br></br>%v</br>catatan: link hanya berlaku %v menit`
)

func NewUserService(
	userRepo UserRepository,
	validate *validator.Validate,
	notifRepo NotificationRepository,
	appEmailVerificationKey string,
	appDeploymentUrl string,
) *userService {
	return &userService{
		userRepo:                userRepo,
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
		logger.Fatal("error when ecnrypt")
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

func (s *userService) Login(ctx context.Context, email, password string) (string, domain.User, error) {
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

	user.Password = ""
	return token, user, nil
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

	if updateData.Email != "" {
		// Validate email format
		if err := s.validate.Var(updateData.Email, "required,email"); err != nil {
			logger.Error("Invalid email format", err)
			return domain.User{}, errors.New("invalid email format")
		}

		// Check if email already exists (excluding current user)
		userWithEmail, err := s.userRepo.FindByEmail(ctx, updateData.Email)
		if err == nil && userWithEmail.ID != id {
			logger.Error("Email already exists")
			return domain.User{}, errors.New("email already exists")
		}
		existingUser.Email = updateData.Email
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

	if updateData.Role != "" {
		existingUser.Role = updateData.Role
	}

	if updateData.Wallet >= 0 {
		existingUser.Wallet = updateData.Wallet
	}

	if updateData.Wallet < 0 {
		return domain.User{}, errors.New("wallet ballance cannot be negative")
	}

	if updateData.Role != "" && !validRoles[updateData.Role] {
		return domain.User{}, errors.New("invalid role")
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
