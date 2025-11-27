package services

import (
	"coolvibes/models"

	"coolvibes/repositories"
)

type PaymentService struct {
	paymentRepo *repositories.PaymentRepository
	mediaRepo   *repositories.MediaRepository
	userRepo    *repositories.UserRepository
	postRepo    *repositories.PostRepository
}

func NewPaymentService(
	paymentRepo *repositories.PaymentRepository,
	userRepo *repositories.UserRepository,
	postRepo *repositories.PostRepository,
	mediaRepo *repositories.MediaRepository) *PaymentService {
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
