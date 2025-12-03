package postgres

import (
	"context"
	"errors"
	"myGreenMarket/domain"
	"time"

	"gorm.io/gorm"
)

type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	if err := r.DB.WithContext(ctx).Create(&user).Error; err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (domain.User, error) {
	var user domain.User

	err := r.DB.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, errors.New("user not found")
		}
		return domain.User{}, err
	}

	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User

	err := r.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, errors.New("user not found")
		}
		return domain.User{}, err
	}

	return user, nil
}

func (r *UserRepository) FindAll(ctx context.Context) ([]domain.User, error) {
	var users []domain.User

	if err := r.DB.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	var existingUser domain.User
	if err := r.DB.WithContext(ctx).First(&existingUser, user.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	user.UpdatedAt = time.Now()

	if err := r.DB.WithContext(ctx).Model(&domain.User{}).Where("id = ?", user.ID).
		Select("full_name", "password", "updated_at").
		Updates(user).Error; err != nil {
		return err
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	result := r.DB.WithContext(ctx).Delete(&domain.User{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found or already deleted")
	}

	return nil
}

func (r *UserRepository) UpdateEmailVerification(ctx context.Context, id uint, isVerified bool) error {
	result := r.DB.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("is_verified", isVerified)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("user not found or status already updated")
	}

	return nil
}
