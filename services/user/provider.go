package services

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewUserService,
	NewAIService,
	NewNewsService,
	NewPostService,
	NewNotificationsService,
	NewPlaceService,
	NewMatchService,
	NewChatService,
	NewPaymentService,
	NewClassifiedService,
)
