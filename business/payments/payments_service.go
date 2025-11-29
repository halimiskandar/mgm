package payments

import (
	"context"
	"errors"
	"myGreenMarket/business/orders"
	"myGreenMarket/business/product"
	"myGreenMarket/business/user"
	"myGreenMarket/domain"
	"myGreenMarket/internal/repository/xendit"
	"myGreenMarket/internal/rest"
	"strconv"
	"strings"
	"time"
)

type PaymentsRepository interface {
	CreatePayment(data domain.Payments) (domain.Payments, error)
	GetAllPayments(user_id int) ([]domain.Payments, error)
	GetPayment(payment_id, user_id int) (domain.Payments, error)
	UpdatePayment(data domain.Payments) error
	DeletePayment(payment_id int) error
	GetPaymentByOrderID(order_id int) (domain.Payments, error)
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

	if !isWallet && data.OrderID == nil {
		return domain.PaymentWithLink{}, errors.New("order id is nil, please add order id")
	}

	if isWallet {
		data.PaymentMethod = "WALLET"
		data.PaymentStatus = "PAID"
		data.CreatedAt = time.Now()
		data.PaymentType = "ORDER"

		user, err := s.userRepo.FindByID(context.TODO(), user_id)
		if err != nil {
			return domain.PaymentWithLink{}, err
		}
		order, err := s.orderRepo.GetOrder(*data.OrderID, int(user_id))
		if err != nil {
			return domain.PaymentWithLink{}, err
		}

		if user.Wallet < order.Subtotal {
			return domain.PaymentWithLink{}, errors.New("insufficient wallet balance")
		}

		if order.OrderStatus == "PAID" {
			return domain.PaymentWithLink{}, errors.New("this order have already been paid")
		}

		product, err := s.productRepo.FindByID(context.TODO(), uint64(order.ProductID))
		if err != nil {
			return domain.PaymentWithLink{}, err
		}
		if product.Quantity == 0 {
			return domain.PaymentWithLink{}, errors.New("product stock is empty")
		}

		payment, err := s.paymentRepo.CreatePayment(data)
		if err != nil {
			return domain.PaymentWithLink{}, err
		}

		user.Wallet -= order.Subtotal

		err = s.userRepo.Update(context.TODO(), &user)
		if err != nil {
			return domain.PaymentWithLink{}, err
		}

		order.OrderStatus = "PAID"
		order.PaymentMethod = "WALLET"
		order.UpdatedAt = time.Now()
		err = s.orderRepo.UpdateOrder(order)
		if err != nil {
			return domain.PaymentWithLink{}, err
		}

		return domain.PaymentWithLink{
			ID:            payment.ID,
			UserID:        payment.UserID,
			OrderID:       *payment.OrderID,
			PaymentStatus: payment.PaymentStatus,
			PaymentMethod: payment.PaymentMethod,
			PaymentLink:   "",
			CreatedAt:     payment.CreatedAt,
		}, nil

	} else {
		existing, err := s.paymentRepo.GetPaymentByOrderID(*data.OrderID)
		if err == nil && existing.PaymentStatus == "PENDING" {
			return domain.PaymentWithLink{}, errors.New("pending payment already exists for this order")
		}

		data.PaymentStatus = "PENDING"
		data.CreatedAt = time.Now()
		data.PaymentType = "ORDER"

		user, err := s.userRepo.FindByID(context.TODO(), user_id)
		if err != nil {
			return domain.PaymentWithLink{}, err
		}
		order, err := s.orderRepo.GetOrder(*data.OrderID, int(user_id))
		if err != nil {
			return domain.PaymentWithLink{}, err
		}
		if order.OrderStatus == "PAID" {
			return domain.PaymentWithLink{}, errors.New("this order have already been paid")
		}

		product, err := s.productRepo.FindByID(context.TODO(), uint64(order.ProductID))
		if err != nil {
			return domain.PaymentWithLink{}, err
		}
		if product.Quantity == 0 {
			return domain.PaymentWithLink{}, errors.New("product stock is empty")
		}

		payment, err := s.paymentRepo.CreatePayment(data)
		if err != nil {
			return domain.PaymentWithLink{}, err
		}

		paymentLink, err := s.xenditRepo.XenditInvoiceUrl("TRANSFER", user.FullName, user.Email, product.ProductName, product.ProductCategory, int(user.ID), int(product.ID), order.Quantity, payment.ID, order.Subtotal, order.PriceEach)
		if err != nil {
			return domain.PaymentWithLink{}, err
		}

		order.OrderStatus = "AWAITING_PAYMENT"
		order.UpdatedAt = time.Now()
		err = s.orderRepo.UpdateOrder(order)
		if err != nil {
			return domain.PaymentWithLink{}, err
		}

		if paymentLink == "" {
			return domain.PaymentWithLink{}, errors.New("payment link doesnt generated, please try again!")
		}
		return domain.PaymentWithLink{
			ID:            payment.ID,
			UserID:        payment.UserID,
			OrderID:       *payment.OrderID,
			PaymentStatus: payment.PaymentStatus,
			PaymentMethod: payment.PaymentMethod,
			PaymentLink:   paymentLink,
			CreatedAt:     payment.CreatedAt,
		}, nil
	}
}
func (s *PaymentsService) GetAllPayments(user_id int) ([]domain.Payments, error) {
	return s.paymentRepo.GetAllPayments(user_id)
}
func (s *PaymentsService) GetPayment(payment_id, user_id int) (domain.Payments, error) {
	return s.paymentRepo.GetPayment(payment_id, user_id)
}
func (s *PaymentsService) ReceivePaymentWebhook(request rest.WebhookRequest) error {
	externalID := strings.Split(request.ExternalID, "|")
	paymentId, _ := strconv.Atoi(externalID[0])
	userId, _ := strconv.Atoi(externalID[1])
	productId, _ := strconv.Atoi(externalID[2])
	purpose := externalID[3]

	var errUpdate error
	payment, err := s.paymentRepo.GetPayment(paymentId, userId)
	if err != nil {
		return err
	}
	if payment.PaymentStatus == "PAID" {
		return nil
	}

	switch purpose {
	case "TRANSFER":
		order, err := s.orderRepo.GetOrder(*payment.OrderID, userId)
		if err != nil {
			return err
		}
		switch request.Status {
		case "PAID":

			payment.PaymentMethod = request.PaymentMethod
			payment.PaymentStatus = request.Status

			product, err := s.productRepo.FindByID(context.TODO(), uint64(productId))
			if err != nil {
				return err
			}

			if product.Quantity == 0 {
				return errors.New("product stock is empty")
			}
			if product.Quantity < float64(request.Items[0].Quantity) {
				return errors.New("insufficient stock")
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

			err = s.productRepo.Update(context.TODO(), &domain.Product{
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
			if err != nil {
				return err
			}

			errUpdate = s.paymentRepo.UpdatePayment(payment)
		case "EXPIRED":
			err = s.orderRepo.UpdateOrder(domain.Orders{
				ID:            order.ID,
				UserID:        order.UserID,
				ProductID:     order.ProductID,
				Quantity:      order.Quantity,
				PriceEach:     order.PriceEach,
				Subtotal:      order.Subtotal,
				OrderStatus:   "PENDING",
				PaymentMethod: request.PaymentMethod,
				CreatedAt:     order.CreatedAt,
				UpdatedAt:     time.Now(),
			})
			if err != nil {
				return err
			}
			payment.PaymentStatus = request.Status
			errUpdate = s.paymentRepo.UpdatePayment(payment)

		}
	case "TOPUP":
		switch request.Status {
		case "PAID":
			user, err := s.userRepo.FindByID(context.TODO(), uint(userId))
			if err != nil {
				return err
			}

			user.Wallet = user.Wallet + float64(request.Amount)
			err = s.userRepo.Update(context.TODO(), &user)
			if err != nil {
				return err
			}

			payment.PaymentMethod = request.PaymentMethod
			payment.PaymentStatus = request.Status
			errUpdate = s.paymentRepo.UpdatePayment(payment)

		case "EXPIRED":
			payment.PaymentStatus = request.Status
			errUpdate = s.paymentRepo.UpdatePayment(payment)
		}
	}

	return errUpdate
}
func (s *PaymentsService) DeletePayment(payment_id int) error {
	return s.paymentRepo.DeletePayment(payment_id)
}

func (s *PaymentsService) TopUp(user_id uint, amount float64) (domain.TopUp, error) {
	user, err := s.userRepo.FindByID(context.TODO(), user_id)
	if err != nil {
		return domain.TopUp{}, err
	}

	payment, err := s.paymentRepo.CreatePayment(domain.Payments{
		UserID:        int(user_id),
		OrderID:       nil,
		PaymentType:   "TOPUP",
		PaymentStatus: "PENDING",
		CreatedAt:     time.Now(),
	})
	if err != nil {
		return domain.TopUp{}, err
	}

	paymentLink, err := s.xenditRepo.XenditInvoiceUrl("TOPUP", user.FullName, user.Email, "Wallet", "Topup", int(user_id), 0, 1, payment.ID, amount, amount)
	if err != nil {
		return domain.TopUp{}, err
	}
	if paymentLink == "" {
		return domain.TopUp{}, errors.New("empty payment link")
	}

	return domain.TopUp{
		ID:        payment.ID,
		UserID:    user_id,
		Amount:    amount,
		TopUpLink: paymentLink,
	}, nil
}
