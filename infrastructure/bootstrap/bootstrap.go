package bootstrap

import (
	"core/adapters/inbound/http/routes"
	"core/adapters/inbound/mcpserver"
	usecases "core/application/usecases"
	"core/helpers"
	"core/infrastructure/ai"
	"core/infrastructure/auth"
	telegramService "core/infrastructure/bot/telegram"
	infraBroadcast "core/infrastructure/broadcast"
	"core/infrastructure/db"
	"core/infrastructure/geoip"
	"core/infrastructure/identity"
	infraMedia "core/infrastructure/media"
	"core/infrastructure/repositories"
	"core/infrastructure/socket"
	"core/mcp"
	broadcastworker "core/workers/broadcast"
	newsworker "core/workers/news"
	"errors"
	"os"
)

func InitializeApp() (*App, error) {
	if err := helpers.ValidateUserJWTConfiguration(); err != nil {
		return nil, err
	}
	gormDB, err := db.NewDatabase()
	if err != nil {
		return nil, err
	}
	initialized := false
	defer func() {
		if !initialized {
			_ = db.Close(gormDB)
		}
	}()
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
	reader, err := geoip.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		if !initialized {
			_ = reader.Close()
		}
	}()

	engagementRepository := repositories.NewEngagementRepository(gormDB)
	notificationRepository := repositories.NewNotificationRepository(gormDB, node)
	userRepository := repositories.NewUserRepository(gormDB, reader, node, engagementRepository, notificationRepository)
	mediaRepository := repositories.NewMediaRepository(gormDB, node)
	privatePhotoRepository := repositories.NewPrivatePhotoRepository(gormDB, node, mediaRepository)
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
	tokenDecoder := auth.TokenDecoder{}
	socketService := socket.NewSocketService(gormDB)
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
		usecases.WithPrivatePhotoBlockRevoker(privatePhotoRepository),
		usecases.WithPrivatePhotoRealtimePublisher(socketService),
	)
	postService := usecases.NewPostService(userRepository, postRepository, mediaRepository)
	listingRepository := repositories.NewListingRepository(gormDB, node, mediaRepository, userRepository, notificationRepository, postRepository)
	classifiedService := usecases.NewClassifiedService(userRepository, postRepository, mediaRepository, placeRepository, listingRepository)
	matchesRepository := repositories.NewMatchesRepository(gormDB, notificationRepository)
	matchesService := usecases.NewMatchService(userRepository, matchesRepository)
	chatRepository := repositories.NewChatRepository(gormDB, node, postRepository, userRepository, notificationRepository)
	chatService := usecases.NewChatService(socketService, userRepository, postRepository, mediaRepository, matchesRepository, chatRepository, notificationRepository)
	notificationsService := usecases.NewNotificationsService(notificationRepository)
	privatePhotoService := usecases.NewPrivatePhotoService(privatePhotoRepository, notificationRepository, socketService)
	paymentRepository := repositories.NewPaymentRepository(gormDB, node, mediaRepository, userRepository, notificationRepository)
	paymentService := usecases.NewPaymentService(paymentRepository, userRepository, postRepository, mediaRepository)
	sitemapRepository := repositories.NewSitemapRepository(gormDB)
	sitemapService := usecases.NewSitemapService(sitemapRepository)
	sessionService := usecases.NewSessionService(userRepository, tokenDecoder)
	mediaAccessService := usecases.NewMediaAccessService(mediaRepository, privatePhotoRepository)
	systemRepository := repositories.NewSystemRepository(gormDB)
	systemService := usecases.NewSystemService(systemRepository)
	moderationRepository := repositories.NewModerationRepository(gormDB)
	moderationService := usecases.NewModerationService(moderationRepository)
	broadcastGateway := infraBroadcast.NewGateway(broadcastGatewayConfigFromEnvironment(), nil)
	broadcastService := usecases.NewBroadcastService(broadcastGateway)
	telegramRuntime, err := telegramService.New()
	if err != nil {
		helpers.Error("Telegram disabled: %v", err)
		telegramRuntime = nil
	}
	newsWorkerDependencies := newsworker.Dependencies{
		Users: userService,
		News:  newsService,
	}
	if telegramRuntime != nil {
		newsWorkerDependencies.Notifier = telegramRuntime
	}
	router := routes.NewRouter(routes.Dependencies{
		MCPServer:           mcpServer,
		UserService:         userService,
		PostService:         postService,
		PlaceService:        placeService,
		NewsService:         newsService,
		ClassifiedService:   classifiedService,
		MatchesService:      matchesService,
		ChatService:         chatService,
		NotificationService: notificationsService,
		PaymentService:      paymentService,
		SystemService:       systemService,
		ModerationService:   moderationService,
		BroadcastService:    broadcastService,
		PrivatePhotoService: privatePhotoService,
		SessionService:      sessionService,
		SitemapService:      sitemapService,
		MediaAccessService:  mediaAccessService,
		TokenDecoder:        tokenDecoder,
		TelegramProcessor:   telegramRuntime,
	})

	application := &App{
		DB:                       gormDB,
		Router:                   router,
		MCPServer:                mcpServer,
		SnowFlakeNode:            node,
		AIRegistry:               registry,
		GEOIPDB:                  reader,
		UserService:              userService,
		NewsService:              newsService,
		ChatService:              chatService,
		BroadcastService:         broadcastService,
		TelegramService:          telegramRuntime,
		NotificationRepository:   notificationRepository,
		MediaProcessorRepository: mediaRepository,
		MediaProcessingObserver:  privatePhotoService,
		NewsWorkerDependencies:   newsWorkerDependencies,
		BroadcastWorkerDependencies: broadcastworker.Dependencies{
			Repository: userRepository,
			Users:      userService,
			Gateway:    broadcastGateway,
		},
	}
	initialized = true
	return application, nil
}

func InitializeMCPOnly() (*mcp.MCPServer, error) {
	server, _, err := InitializeMCPOnlyWithCleanup()
	return server, err
}

func InitializeMCPOnlyWithCleanup() (*mcp.MCPServer, func() error, error) {
	config := ai.NewConfig()
	client := ai.NewHTTPClient()
	registry, err := ai.NewRegistry(config, client)
	if err != nil {
		return nil, nil, err
	}
	aiService := usecases.NewAIService(registry)
	gormDB, err := db.NewDatabase()
	if err != nil {
		return nil, nil, err
	}
	initialized := false
	defer func() {
		if !initialized {
			_ = db.Close(gormDB)
		}
	}()
	reader, err := geoip.Open()
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if !initialized {
			_ = reader.Close()
		}
	}()
	node, err := helpers.NewDefaultNode()
	if err != nil {
		return nil, nil, err
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
	server, err := mcpserver.NewServer(aiService, newsService, placeService)
	if err != nil {
		return nil, nil, err
	}
	initialized = true
	cleanup := func() error {
		return errors.Join(reader.Close(), db.Close(gormDB))
	}
	return server, cleanup, nil
}
