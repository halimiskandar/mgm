package payments

import (
	"context"
	"myGreenMarket/business/orders"
	"myGreenMarket/business/product"
	"myGreenMarket/business/user"
	"myGreenMarket/domain"
	"myGreenMarket/internal/repository/xendit"
	"myGreenMarket/internal/rest"
	"time"
)

type PaymentsRepository interface {
	CreatePayment(data domain.Payments) (domain.Payments, error)
	GetAllPayments() ([]domain.Payments, error)
	GetPayment(payment_id int) (domain.Payments, error)
	UpdatePayment(data domain.Payments) error
	DeletePayment(payment_id int) error
}

type PaymentsService struct {
	paymentRepo PaymentsRepository
	xenditRepo  *xendit.XenditRepository
	userRepo    user.UserRepository
	orderRepo   orders.OrdersRepository
	productRepo product.ProductRepository
}

func NewPaymentsService(paymentRepo PaymentsRepository, xenditRepo *xendit.XenditRepository, userRepo user.UserRepository, orderRepo orders.OrdersRepository, productRepo product.ProductRepository) *PaymentsService {
	return &PaymentsService{
		paymentRepo: paymentRepo,
		xenditRepo:  xenditRepo,
		userRepo:    userRepo,
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

func (s *PaymentsService) CreatePayment(data domain.Payments, isWallet bool, user_id uint) (domain.PaymentWithLink, error) {
	if isWallet {
		data.PaymentMethod = "WALLET"
	}
	data.PaymentStatus = "PENDING"
	data.CreatedAt = time.Now()

	payment, err := s.paymentRepo.CreatePayment(data)
	if err != nil {
		return domain.PaymentWithLink{}, err
	}

	user, err := s.userRepo.FindByID(context.TODO(), user_id)
	if err != nil {
		return domain.PaymentWithLink{}, err
	}
	order, err := s.orderRepo.GetOrder(data.OrderID)
	if err != nil {
		return domain.PaymentWithLink{}, err
	}
	product, err := s.productRepo.FindByID(context.TODO(), uint64(order.ProductID))
	if err != nil {
		return domain.PaymentWithLink{}, err
	}
	paymentLink, err := s.xenditRepo.XenditInvoiceUrl("TRANSFER", user.FullName, user.Email, product.ProductName, product.ProductCategory, int(user.ID), int(product.ID), order.Quantity, payment.ID, order.Subtotal)
	if err != nil {
		return domain.PaymentWithLink{}, err
	}

	return domain.PaymentWithLink{
		ID:            payment.ID,
		OrderID:       payment.OrderID,
		PaymentStatus: payment.PaymentStatus,
		PaymentMethod: payment.PaymentMethod,
		PaymentLink:   paymentLink,
		CreatedAt:     payment.CreatedAt,
	}, nil
}
func (s *PaymentsService) GetAllPayments() ([]domain.Payments, error) {
	return s.paymentRepo.GetAllPayments()
}
func (s *PaymentsService) GetPayment(payment_id int) (domain.Payments, error) {
	return s.paymentRepo.GetPayment(payment_id)
}
func (s *PaymentsService) UpdatePayment(data domain.Payments, user_id, productId int, request rest.WebhookRequest, purpose string) error {
	switch purpose {
	case "TRANSFER":
		switch request.Status {
		case "PAID":
			product, err := s.productRepo.FindByID(context.TODO(), uint64(productId))
			if err != nil {
				return err
			}

			s.productRepo.Update(context.TODO(), &domain.Product{
				ID:              product.ID,
				ProductID:       product.ProductID,
				ProductSKUID:    product.ProductSKUID,
				IsGreenTag:      product.IsGreenTag,
				ProductName:     product.ProductName,
				ProductCategory: product.ProductCategory,
				Unit:            product.Unit,
				NormalPrice:     product.NormalPrice,
				SalePrice:       product.SalePrice,
				Discount:        product.Discount,
				Quantity:        product.Quantity - float64(request.Items[0].Quantity),
				CreatedAt:       product.CreatedAt,
			})

			data.PaymentMethod = request.PaymentMethod
			data.PaymentStatus = request.Status

			order, err := s.orderRepo.GetOrder(data.OrderID)
			if err != nil {
				return err
			}
			err = s.orderRepo.UpdateOrder(domain.Orders{
				ID:            order.ID,
				UserID:        order.UserID,
				ProductID:     order.ProductID,
				Quantity:      order.Quantity,
				PriceEach:     order.PriceEach,
				Subtotal:      order.Subtotal,
				OrderStatus:   "PAID",
				PaymentMethod: request.PaymentMethod,
				CreatedAt:     order.CreatedAt,
				UpdatedAt:     time.Now(),
			})
			if err != nil {
				return err
			}

		case "EXPIRED":
			data.PaymentStatus = request.Status
		}
	}

	return s.paymentRepo.UpdatePayment(data)
}
func (s *PaymentsService) DeletePayment(payment_id int) error {
	return s.paymentRepo.DeletePayment(payment_id)
}
