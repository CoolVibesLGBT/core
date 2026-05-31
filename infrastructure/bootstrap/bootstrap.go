package bootstrap

import (
	"core/ai"
	"core/application/mcpserver"
	"core/helpers"
	"core/infrastructure/auth"
	"core/infrastructure/identity"
	"core/mcp"
	"core/repositories"
	"core/routes"
	"core/services/db"
	"core/services/socket"
	services "core/services/user"
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

	aiService := services.NewAIService(registry)
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
	newsService := services.NewNewsService(userRepository, postRepository, mediaRepository, placeRepository, newsRepository)
	placeService := services.NewPlaceService(userRepository, postRepository, mediaRepository, placeRepository)
	mcpServer := mcpserver.NewServer(aiService, newsService, placeService)

	publicIDGenerator := identity.NewSnowflakePublicIDGenerator(node)
	userService := services.NewUserService(
		userRepository,
		postRepository,
		mediaRepository,
		engagementRepository,
		notificationRepository,
		services.WithCaptchaVerifier(auth.NewGoogleCaptchaVerifier(os.Getenv("CAPTCHA_SECRET_KEY"), nil)),
		services.WithPasswordHasher(auth.PasswordHasher{}),
		services.WithTokenIssuer(auth.TokenIssuer{}),
		services.WithPublicIDGenerator(publicIDGenerator),
	)
	postService := services.NewPostService(userRepository, postRepository, mediaRepository)
	listingRepository := repositories.NewListingRepository(gormDB, node, mediaRepository, userRepository, notificationRepository, postRepository)
	classifiedService := services.NewClassifiedService(userRepository, postRepository, mediaRepository, placeRepository, listingRepository)
	matchesRepository := repositories.NewMatchesRepository(gormDB, engagementRepository, notificationRepository)
	matchesService := services.NewMatchService(userRepository, postRepository, mediaRepository, matchesRepository)
	socketService := socket.NewSocketService(gormDB)
	chatRepository := repositories.NewChatRepository(gormDB, node, postRepository, userRepository, notificationRepository)
	chatService := services.NewChatService(socketService, userRepository, postRepository, mediaRepository, matchesRepository, chatRepository, notificationRepository)
	notificationsService := services.NewNotificationsService(notificationRepository)
	paymentRepository := repositories.NewPaymentRepository(gormDB, node, mediaRepository, userRepository, notificationRepository)
	paymentService := services.NewPaymentService(paymentRepository, userRepository, postRepository, mediaRepository)
	sitemapRepository := repositories.NewSitemapRepository(gormDB)
	router := routes.NewRouter(gormDB, node, mcpServer, userService, postService, placeService, newsService, classifiedService, matchesService, chatService, notificationsService, paymentService, userRepository, notificationRepository, sitemapRepository, reader)

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
	aiService := services.NewAIService(registry)
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
	newsService := services.NewNewsService(userRepository, postRepository, mediaRepository, placeRepository, newsRepository)
	placeService := services.NewPlaceService(userRepository, postRepository, mediaRepository, placeRepository)
	return mcpserver.NewServer(aiService, newsService, placeService), nil
}
