package postgres

import (
	"context"
	"errors"
	"myGreenMarket/domain"

	"gorm.io/gorm"
)

type OrdersRepository struct {
	DB *gorm.DB
}

func NewOrdersRepository(db *gorm.DB) *OrdersRepository {
	return &OrdersRepository{
		DB: db,
	}
}

func (r *OrdersRepository) CreateOrder(data domain.Orders) (domain.Orders, error) {
	ctx := context.Background()
	err := r.DB.WithContext(ctx).Create(&data).Error
	if err != nil {
		return domain.Orders{}, err
	}

	return data, nil
}

func (r *OrdersRepository) GetAllOrders() ([]domain.Orders, error) {
	ctx := context.Background()
	var orders []domain.Orders
	err := r.DB.WithContext(ctx).Find(&orders).Error
	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *OrdersRepository) GetOrder(order_id int) (domain.Orders, error) {
	ctx := context.Background()
	var order domain.Orders
	err := r.DB.WithContext(ctx).Where("id=?", order_id).First(&order).Error
	if err != nil {
		return domain.Orders{}, err
	}

	return order, nil
}

func (r *OrdersRepository) GetOrderStatus(status string, user_id int) (domain.Orders, error) {
	ctx := context.Background()
	var order domain.Orders
	err := r.DB.WithContext(ctx).Where("order_status=?", status).Where("user_id=?", user_id).First(&order).Error
	if err != nil {
		return domain.Orders{}, err
	}

	return order, nil
}

func (r *OrdersRepository) UpdateOrder(data domain.Orders) error {
	ctx := context.Background()
	row := r.DB.WithContext(ctx).Where("id=?", data.ID).Updates(&data)
	if row.RowsAffected == 0 {
		return errors.New("order_id not found")
	}
	if err := row.Error; err != nil {
		return err
	}

	return nil
}

func (r *OrdersRepository) DeleteOrder(order_id int) error {
	ctx := context.Background()
	row := r.DB.WithContext(ctx).Where("id=?", order_id).Delete(&domain.Orders{})
	if row.RowsAffected == 0 {
		return errors.New("order_id not found")
	}
	if err := row.Error; err != nil {
		return err
	}

	return nil
}
