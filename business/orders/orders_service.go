package orders

import (
	"context"
	"myGreenMarket/business/product"
	"myGreenMarket/domain"
	"time"
)

type OrdersRepository interface {
	CreateOrder(data domain.Orders) (domain.Orders, error)
	GetAllOrders() ([]domain.Orders, error)
	GetOrder(order_id int) (domain.Orders, error)
	GetOrderStatus(status string, user_id int) (domain.Orders, error)
	UpdateOrder(data domain.Orders) error
	DeleteOrder(order_id int) error
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

	data.PriceEach = product.NormalPrice
	data.Subtotal = product.NormalPrice * float64(data.Quantity)
	data.OrderStatus = "PENDING"
	data.CreatedAt = time.Now()
	data.UpdatedAt = time.Now()

	return s.orderRepo.CreateOrder(data)
}
func (s *OrdersService) GetAllOrders() ([]domain.Orders, error) {
	return s.orderRepo.GetAllOrders()
}
func (s *OrdersService) GetOrder(order_id int) (domain.Orders, error) {
	return s.orderRepo.GetOrder(order_id)
}

func (s *OrdersService) GetOrderStatus(status string, user_id int) (domain.Orders, error) {
	return s.orderRepo.GetOrderStatus(status, user_id)
}

func (s *OrdersService) UpdateOrder(data domain.Orders) error {
	return s.orderRepo.UpdateOrder(data)
}
func (s *OrdersService) DeleteOrder(order_id int) error {
	return s.orderRepo.DeleteOrder(order_id)
}
