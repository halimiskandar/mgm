package orders

import (
	"context"
	"errors"
	"myGreenMarket/business/product"
	"myGreenMarket/domain"
	"time"
)

type OrdersRepository interface {
	CreateOrder(data domain.Orders) (domain.Orders, error)
	GetAllOrders(user_id int) ([]domain.Orders, error)
	GetOrder(order_id, user_id int) (domain.Orders, error)
	GetOrderStatus(status string, user_id int) (domain.Orders, error)
	UpdateOrder(data domain.Orders) error
	DeleteOrder(order_id, user_id int) error
}

type OrdersService struct {
	orderRepo    OrdersRepository
	productsRepo product.ProductRepository
}

func NewOrdersService(orderRepo OrdersRepository, productsRepo product.ProductRepository) *OrdersService {
	return &OrdersService{
		orderRepo:    orderRepo,
		productsRepo: productsRepo,
	}
}

func (s *OrdersService) CreateOrder(data domain.Orders) (domain.Orders, error) {
	product, err := s.productsRepo.FindByID(context.TODO(), uint64(data.ProductID))
	if err != nil {
		return domain.Orders{}, err
	}
	if product.Quantity == 0 {
		return domain.Orders{}, errors.New("product stock is empty")
	}
	if product.Quantity < float64(data.Quantity) {
		return domain.Orders{}, errors.New("insufficient stock")
	}

	data.PriceEach = product.NormalPrice
	data.Subtotal = product.NormalPrice * float64(data.Quantity)
	data.OrderStatus = "PENDING"
	data.CreatedAt = time.Now()
	data.UpdatedAt = time.Now()

	return s.orderRepo.CreateOrder(data)
}
func (s *OrdersService) GetAllOrders(user_id int) ([]domain.Orders, error) {
	return s.orderRepo.GetAllOrders(user_id)
}
func (s *OrdersService) GetOrder(order_id, user_id int) (domain.Orders, error) {
	return s.orderRepo.GetOrder(order_id, user_id)
}

func (s *OrdersService) GetOrderStatus(status string, user_id int) (domain.Orders, error) {
	return s.orderRepo.GetOrderStatus(status, user_id)
}

func (s *OrdersService) UpdateOrder(data domain.Orders) error {
	order, err := s.orderRepo.GetOrder(data.ID, data.UserID)
	if err != nil {
		return err
	}

	switch order.OrderStatus {
	case "AWAITING_PAYMENT":
		return errors.New("order is awaiting payment and cannot be updated")
	case "PAID":
		return errors.New("order have already been paid")
	case "CANCELLED":
		return errors.New("order cancelled")
	}

	if data.UserID != order.UserID {
		return errors.New("this is not your order")
	}
	product, err := s.productsRepo.FindByID(context.TODO(), uint64(order.ProductID))
	if err != nil {
		return err
	}
	if product.Quantity < float64(data.Quantity) {
		return errors.New("insufficient stock")
	}

	data.ProductID = order.ProductID
	data.OrderStatus = order.OrderStatus
	data.PaymentMethod = order.PaymentMethod
	data.PriceEach = order.PriceEach
	data.Subtotal = order.PriceEach * float64(data.Quantity)
	data.CreatedAt = order.CreatedAt
	data.UpdatedAt = time.Now()
	return s.orderRepo.UpdateOrder(data)
}
func (s *OrdersService) DeleteOrder(order_id, user_id int) error {
	order, err := s.orderRepo.GetOrder(order_id, user_id)
	if err != nil {
		return err
	}

	switch order.OrderStatus {
	case "AWAITING_PAYMENT":
		return errors.New("order is awaiting payment and cannot be deleted")
	case "PAID":
		return errors.New("order have already been paid")
	}

	return s.orderRepo.DeleteOrder(order_id, user_id)
}
