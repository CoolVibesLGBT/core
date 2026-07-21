package main

import (
	"context"
	"core/constants"
	"core/helpers"
	app "core/infrastructure/bootstrap"
	"core/infrastructure/db"
	"core/infrastructure/repositories"
	"core/infrastructure/socket"
	"core/infrastructure/socket/managers"
	"core/models"
	chatworker "core/workers/chat"
	mediaworker "core/workers/media"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

const shutdownTimeout = 3 * time.Second

var (
	appInstance  *app.App
	botFlag      = flag.Bool("bot", false, "Run the HTTP webhook server with Telegram enabled")
	migrateFlag  = flag.Bool("migrate", false, "Run DB migrations")
	seedFlag     = flag.Bool("seed", false, "Run DB seed")
	installFlag  = flag.Bool("install", false, "Run DB migrate & seed")
	mcpStdioFlag = flag.Bool("mcp-stdio", false, "Run the MCP server over stdio")
	grantAdmin   = flag.Int64("grant-admin", 0, "Grant admin role to an existing user public ID")
	grantMod     = flag.Int64("grant-moderator", 0, "Grant moderator role to an existing user public ID")
)

func ensureFlagsParsed() {
	if !flag.Parsed() {
		flag.Parse()
	}
}

func NewApp() (*app.App, error) {
	if appInstance != nil {
		return appInstance, nil
	}

	application, err := app.InitializeApp()
	if err != nil {
		return nil, err
	}
	appInstance = application
	return application, nil
}

func GetApp() (*app.App, error) {
	return NewApp()
}

func main() {
	if err := run(); err != nil {
		log.Printf("application stopped with error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	ensureFlagsParsed()
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using system env")
	}

	if handled, err := runMaintenance(); handled {
		return err
	}

	if *mcpStdioFlag {
		mcpServer, cleanup, err := app.InitializeMCPOnlyWithCleanup()
		if err != nil {
			return fmt.Errorf("initialize MCP server: %w", err)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		serveErr := mcpServer.ServeStdio(ctx, os.Stdin, os.Stdout)
		return errors.Join(serveErr, cleanup())
	}

	application, err := NewApp()
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	if *botFlag {
		log.Println("bot mode uses the HTTP Telegram webhook endpoint")
	}

	return serve(application)
}

func runMaintenance() (handled bool, retErr error) {
	runMigrate, runSeed := maintenanceModes(*migrateFlag, *seedFlag, *installFlag)
	if !runMigrate && !runSeed && *grantAdmin == 0 && *grantMod == 0 {
		return false, nil
	}
	if *grantAdmin < 0 || *grantMod < 0 {
		return true, errors.New("role grant public IDs must be positive")
	}
	if *grantAdmin != 0 && *grantAdmin == *grantMod {
		return true, errors.New("the same user cannot be granted two roles in one command")
	}

	database, err := db.NewDatabase()
	if err != nil {
		return true, err
	}
	defer func() {
		retErr = errors.Join(retErr, db.Close(database))
	}()

	err = executeMaintenance(runMigrate, runSeed, maintenanceSteps{
		migrate: func() error {
			log.Println("Migration:BEGIN")
			if err := db.Migrate(database); err != nil {
				return err
			}
			if err := db.MigrateIndexes(database); err != nil {
				return err
			}
			log.Println("Migration:END")
			return nil
		},
		seed: func() error {
			node, err := helpers.NewDefaultNode()
			if err != nil {
				return fmt.Errorf("initialize seed ID generator: %w", err)
			}
			return db.Seed(database, node)
		},
	})
	if err != nil {
		return true, err
	}
	if *grantAdmin != 0 {
		if err := grantUserRole(database, *grantAdmin, constants.UserRoleAdmin); err != nil {
			return true, err
		}
	}
	if *grantMod != 0 {
		if err := grantUserRole(database, *grantMod, constants.UserRoleModerator); err != nil {
			return true, err
		}
	}
	return true, err
}

func grantUserRole(database *gorm.DB, publicID int64, role constants.UserRole) error {
	result := database.Model(&models.User{}).
		Where("public_id = ? AND deleted_at IS NULL", publicID).
		Where("user_role NOT IN ?", []constants.UserRole{constants.UserRoleBanned, constants.UserRoleDeleted}).
		Update("user_role", role)
	if result.Error != nil {
		return fmt.Errorf("grant %s role to user %d: %w", role, publicID, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("eligible user %d was not found", publicID)
	}
	log.Printf("granted %s role to user %d", role, publicID)
	return nil
}

type maintenanceSteps struct {
	migrate func() error
	seed    func() error
}

func maintenanceModes(migrate, seed, install bool) (runMigrate, runSeed bool) {
	return migrate || install, seed || install
}

func executeMaintenance(runMigrate, runSeed bool, steps maintenanceSteps) error {
	if runMigrate {
		if steps.migrate == nil {
			return errors.New("migration step is not configured")
		}
		if err := steps.migrate(); err != nil {
			return fmt.Errorf("migrate database: %w", err)
		}
	}
	if runSeed {
		if steps.seed == nil {
			return errors.New("seed step is not configured")
		}
		if err := steps.seed(); err != nil {
			return fmt.Errorf("seed database: %w", err)
		}
	}
	return nil
}

func serve(application *app.App) (retErr error) {
	if application == nil || application.Router == nil || application.DB == nil {
		return errors.New("application is not fully initialized")
	}

	defer func() {
		if application.Router.GEOIPDB != nil {
			retErr = errors.Join(retErr, application.Router.GEOIPDB.Close())
		}
		retErr = errors.Join(retErr, db.Close(application.DB))
	}()

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		return errors.New("PORT is required")
	}
	listener, err := net.Listen("tcp", port)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", port, err)
	}

	serviceCtx, cancelServices := context.WithCancel(context.Background())
	defer cancelServices()

	notificationRepo := repositories.NewNotificationRepository(application.DB, application.SnowFlakeNode)
	notificationMgr := managers.NewNotificationManager(application.DB, notificationRepo)

	var socketRuntime *socket.Runtime
	socketPort := strings.TrimSpace(os.Getenv("SOCKET_PORT"))
	if socketPort != "" {
		socketRuntime, err = socket.StartServer(socketPort, application.DB, notificationMgr)
		if err != nil {
			_ = listener.Close()
			return fmt.Errorf("start socket server: %w", err)
		}
	}

	mediaProcessor := mediaworker.StartProcessorContext(serviceCtx, application.DB, application.SnowFlakeNode)
	chatProcessor := chatworker.StartProcessorContext(serviceCtx, application.Router.ChatService)

	fiberApp := application.Router.GetFiber()
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- fiberApp.Listener(listener, fiber.ListenConfig{DisableStartupMessage: true})
	}()

	registerTelegramWebhook(application)
	log.Printf("%s ready on %s", constants.APPLICATION_NAME, listener.Addr())
	if socketRuntime != nil {
		log.Printf("socket ready on %s", socketRuntime.Addr())
	}

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	var serveErr error
	var socketErrors <-chan error
	if socketRuntime != nil {
		socketErrors = socketRuntime.Errors()
	}

	waiting := true
	for waiting {
		select {
		case <-signalCtx.Done():
			waiting = false
		case err := <-serverErrors:
			if err != nil {
				serveErr = fmt.Errorf("HTTP server: %w", err)
			}
			waiting = false
		case err, ok := <-socketErrors:
			if !ok {
				socketErrors = nil
				continue
			}
			if err != nil {
				serveErr = fmt.Errorf("socket server: %w", err)
				waiting = false
			}
		}
	}

	cancelServices()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	shutdownErr := shutdown(
		shutdownCtx,
		fiberApp,
		socketRuntime,
		mediaProcessor,
		chatProcessor,
	)
	return errors.Join(serveErr, shutdownErr)
}

func registerTelegramWebhook(application *app.App) {
	service := application.Router.TelegramService
	if service == nil {
		return
	}

	go func() {
		if err := service.RegisterWebhook(); err != nil {
			log.Printf("Telegram webhook disabled: %v", err)
			return
		}
		log.Println("Telegram webhook registered")
	}()
}

type processorShutdown interface {
	Shutdown(context.Context) error
}

func shutdown(
	ctx context.Context,
	fiberApp *fiber.App,
	socketRuntime *socket.Runtime,
	processors ...processorShutdown,
) error {
	errorsCh := make(chan error, 2+len(processors))
	count := 0

	if fiberApp != nil {
		count++
		go func() { errorsCh <- fiberApp.ShutdownWithContext(ctx) }()
	}
	if socketRuntime != nil {
		count++
		go func() { errorsCh <- socketRuntime.Shutdown(ctx) }()
	}
	for _, processor := range processors {
		if processor == nil {
			continue
		}
		count++
		go func(p processorShutdown) { errorsCh <- p.Shutdown(ctx) }(processor)
	}

	var result error
	for range count {
		result = errors.Join(result, <-errorsCh)
	}
	return result
}
