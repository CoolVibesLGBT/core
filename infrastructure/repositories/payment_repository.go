package repositories

import (
	"core/application/ports"
	"core/helpers"
	"core/models"
	"core/models/payment"

	"gorm.io/gorm"
)

type PaymentRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	mediaRepo        *MediaRepository
	userRepo         *UserRepository
	notificationRepo *NotificationRepository
}

func (r *PaymentRepository) DB() *gorm.DB {
	return r.db
}

func (r *PaymentRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewPaymentRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository, userRepo *UserRepository, notificationRepo *NotificationRepository) *PaymentRepository {
	return &PaymentRepository{db: db, snowFlakeNode: snowFlakeNode, mediaRepo: mediaRepo, userRepo: userRepo, notificationRepo: notificationRepo}
}

func (r *PaymentRepository) ProcessPayment(paymentKind payment.PaymentKind, authUser models.User) error {
	return ports.ErrPaymentProcessingNotImplemented
}

func (r *PaymentRepository) GooglePay(authUser models.User) error {
	return r.ProcessPayment(payment.PaymentKind_GOOGLEPAY, authUser)
}

func (r *PaymentRepository) Crypto(authUser models.User) error {
	return r.ProcessPayment(payment.PaymentKind_CRYPTO, authUser)
}

func (r *PaymentRepository) IBAN(authUser models.User) error {
	return r.ProcessPayment(payment.PaymentKind_IBAN, authUser)
}

func (r *PaymentRepository) Deposit(authUser models.User) error {
	return r.ProcessPayment(payment.PaymentKind_GOOGLEPAY, authUser)
}

func (r *PaymentRepository) Withdraw(authUser models.User) error {
	return r.ProcessPayment(payment.PaymentKind_GOOGLEPAY, authUser)
}

func (r *PaymentRepository) Transactions(authUser models.User) error {
	return r.ProcessPayment(payment.PaymentKind_GOOGLEPAY, authUser)
}
