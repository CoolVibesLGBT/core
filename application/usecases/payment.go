package usecases

import (
	"core/application/ports"
	"core/models"
)

type PaymentService struct {
	paymentRepo ports.PaymentRepository
	mediaRepo   ports.MediaRepository
	userRepo    ports.UserRepository
	postRepo    ports.PostRepository
}

func NewPaymentService(
	paymentRepo ports.PaymentRepository,
	userRepo ports.UserRepository,
	postRepo ports.PostRepository,
	mediaRepo ports.MediaRepository) *PaymentService {
	return &PaymentService{paymentRepo: paymentRepo, postRepo: postRepo, mediaRepo: mediaRepo, userRepo: userRepo}
}

func (s *PaymentService) Deposit(author models.User) error {
	return s.paymentRepo.Deposit(author)
}

func (s *PaymentService) Withdraw(author models.User) error {
	return s.paymentRepo.Withdraw(author)
}

func (s *PaymentService) Transactions(author models.User) error {
	return s.paymentRepo.Transactions(author)
}
