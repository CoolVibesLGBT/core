package bootstrap

import (
	"core/application/mcpserver"
	usecases "core/application/usecases"
	"core/helpers"
	"core/infrastructure/ai"
	"core/infrastructure/auth"
	"core/infrastructure/db"
	"core/infrastructure/identity"
	infraMedia "core/infrastructure/media"
	"core/infrastructure/repositories"
	"core/infrastructure/socket"
	"core/mcp"
	"core/routes"
	"os"
)

func InitializeApp() (*App, error) {
	gormDB, err := db.NewDatabase()
	if err != nil {
		return nil, err
	}
	node, err := helpers.NewDefaultNode()
	if err != nil {
		return nil, err
	}
	config := ai.NewConfig()
	client := ai.NewHTTPClient()
	registry, err := ai.NewRegistry(config, client)
	if err != nil {
		return nil, err
	}

	aiService := usecases.NewAIService(registry)
	reader, err := routes.GeoIPDBProvider()
	if err != nil {
		return nil, err
	}

	engagementRepository := repositories.NewEngagementRepository(gormDB)
	notificationRepository := repositories.NewNotificationRepository(gormDB, node)
	userRepository := repositories.NewUserRepository(gormDB, reader, node, engagementRepository, notificationRepository)
	mediaRepository := repositories.NewMediaRepository(gormDB, node)
	postRepository := repositories.NewPostRepository(gormDB, node, mediaRepository, userRepository, notificationRepository)
	placeRepository := repositories.NewPlaceRepository(gormDB, node, mediaRepository, userRepository, notificationRepository, postRepository)
	newsRepository := repositories.NewNewsRepository(gormDB, node, mediaRepository, userRepository, notificationRepository, postRepository)
	newsService := usecases.NewNewsService(userRepository, postRepository, mediaRepository, placeRepository, newsRepository)
	placeService := usecases.NewPlaceService(userRepository, postRepository, mediaRepository, placeRepository)
	mcpServer, err := mcpserver.NewServer(aiService, newsService, placeService)
	if err != nil {
		return nil, err
	}

	publicIDGenerator := identity.NewSnowflakePublicIDGenerator(node)
	userService := usecases.NewUserService(
		userRepository,
		postRepository,
		mediaRepository,
		engagementRepository,
		notificationRepository,
		usecases.WithCaptchaVerifier(auth.NewGoogleCaptchaVerifier(os.Getenv("CAPTCHA_SECRET_KEY"), nil)),
		usecases.WithPasswordHasher(auth.PasswordHasher{}),
		usecases.WithTokenIssuer(auth.TokenIssuer{}),
		usecases.WithPublicIDGenerator(publicIDGenerator),
		usecases.WithRemoteImageFetcher(infraMedia.NewRemoteImageFetcher(nil)),
	)
	postService := usecases.NewPostService(userRepository, postRepository, mediaRepository)
	listingRepository := repositories.NewListingRepository(gormDB, node, mediaRepository, userRepository, notificationRepository, postRepository)
	classifiedService := usecases.NewClassifiedService(userRepository, postRepository, mediaRepository, placeRepository, listingRepository)
	matchesRepository := repositories.NewMatchesRepository(gormDB, engagementRepository, notificationRepository)
	matchesService := usecases.NewMatchService(userRepository, postRepository, mediaRepository, matchesRepository)
	socketService := socket.NewSocketService(gormDB)
	chatRepository := repositories.NewChatRepository(gormDB, node, postRepository, userRepository, notificationRepository)
	chatService := usecases.NewChatService(socketService, userRepository, postRepository, mediaRepository, matchesRepository, chatRepository, notificationRepository)
	notificationsService := usecases.NewNotificationsService(notificationRepository)
	paymentRepository := repositories.NewPaymentRepository(gormDB, node, mediaRepository, userRepository, notificationRepository)
	paymentService := usecases.NewPaymentService(paymentRepository, userRepository, postRepository, mediaRepository)
	sitemapRepository := repositories.NewSitemapRepository(gormDB)
	systemRepository := repositories.NewSystemRepository(gormDB)
	systemService := usecases.NewSystemService(systemRepository)
	moderationRepository := repositories.NewModerationRepository(gormDB)
	moderationService := usecases.NewModerationService(moderationRepository)
	router := routes.NewRouter(gormDB, node, mcpServer, userService, postService, placeService, newsService, classifiedService, matchesService, chatService, notificationsService, paymentService, systemService, moderationService, userRepository, sitemapRepository, reader)

	return &App{
		DB:            gormDB,
		Router:        router,
		MCPServer:     mcpServer,
		SnowFlakeNode: node,
		AIRegistry:    registry,
	}, nil
}

func InitializeMCPOnly() (*mcp.MCPServer, error) {
	config := ai.NewConfig()
	client := ai.NewHTTPClient()
	registry, err := ai.NewRegistry(config, client)
	if err != nil {
		return nil, err
	}
	aiService := usecases.NewAIService(registry)
	gormDB, err := db.NewDatabase()
	if err != nil {
		return nil, err
	}
	reader, err := routes.GeoIPDBProvider()
	if err != nil {
		return nil, err
	}
	node, err := helpers.NewDefaultNode()
	if err != nil {
		return nil, err
	}
	engagementRepository := repositories.NewEngagementRepository(gormDB)
	notificationRepository := repositories.NewNotificationRepository(gormDB, node)
	userRepository := repositories.NewUserRepository(gormDB, reader, node, engagementRepository, notificationRepository)
	mediaRepository := repositories.NewMediaRepository(gormDB, node)
	postRepository := repositories.NewPostRepository(gormDB, node, mediaRepository, userRepository, notificationRepository)
	placeRepository := repositories.NewPlaceRepository(gormDB, node, mediaRepository, userRepository, notificationRepository, postRepository)
	newsRepository := repositories.NewNewsRepository(gormDB, node, mediaRepository, userRepository, notificationRepository, postRepository)
	newsService := usecases.NewNewsService(userRepository, postRepository, mediaRepository, placeRepository, newsRepository)
	placeService := usecases.NewPlaceService(userRepository, postRepository, mediaRepository, placeRepository)
	return mcpserver.NewServer(aiService, newsService, placeService)
}
