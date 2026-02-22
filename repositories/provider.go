package repositories

import (
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewUserRepository,
	NewEngagementRepository,
	NewMediaRepository,
	NewNotificationRepository,
	NewPostRepository,
	NewPlaceRepository,
	NewMatchesRepository,
	NewNewsRepository,
	NewChatRepository,
	NewPaymentRepository,
	NewSitemapRepository,
)
