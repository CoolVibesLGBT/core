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
	NewListingRepository,
	NewMatchesRepository,
	NewNewsRepository,
	NewChatRepository,
	NewPaymentRepository,
	NewSitemapRepository,
)
