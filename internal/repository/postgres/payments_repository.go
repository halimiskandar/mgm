package postgres

import (
	"context"
	"errors"
	"myGreenMarket/domain"

	"gorm.io/gorm"
)

type PaymentsRepository struct {
	DB *gorm.DB
}

func NewPaymentsRepository(db *gorm.DB) *PaymentsRepository {
	return &PaymentsRepository{
		DB: db,
	}
}

func (r *PaymentsRepository) CreatePayment(data domain.Payments) (domain.Payments, error) {
	ctx := context.Background()
	err := r.DB.WithContext(ctx).Create(&data).Error
	if err != nil {
		return domain.Payments{}, err
	}

	return data, nil
}

func (r *PaymentsRepository) GetAllPayments() ([]domain.Payments, error) {
	ctx := context.Background()
	var payments []domain.Payments
	err := r.DB.WithContext(ctx).Find(&payments).Error
	if err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *PaymentsRepository) GetPayment(payment_id int) (domain.Payments, error) {
	ctx := context.Background()
	var payment domain.Payments
	err := r.DB.WithContext(ctx).Where("id=?", payment_id).First(&payment).Error
	if err != nil {
		return domain.Payments{}, err
	}

	return payment, nil
}

func (r *PaymentsRepository) UpdatePayment(data domain.Payments) error {
	ctx := context.Background()
	row := r.DB.WithContext(ctx).Where("id=?", data.ID).Updates(data)
	if err := row.Error; err != nil {
		return err
	}

	if row.RowsAffected == 0 {
		return errors.New("payment_id not found")
	}

	return nil
}

func (r *PaymentsRepository) DeletePayment(payment_id int) error {
	ctx := context.Background()
	row := r.DB.WithContext(ctx).Where("id=?", payment_id).Delete(&domain.Payments{})

	if err := row.Error; err != nil {
		return err
	}

	if row.RowsAffected == 0 {
		return errors.New("payment_id not found")
	}

	return nil
}
