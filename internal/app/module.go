package app

import (
	"fmt"
	"time"

	"github.com/aliakrem/gokan/internal/modules/message"
	"github.com/aliakrem/gokan/internal/modules/room"
	"github.com/aliakrem/gokan/internal/modules/ticket"
	"github.com/aliakrem/gokan/internal/modules/user"
	"github.com/aliakrem/gokan/internal/modules/websocket"
	"github.com/aliakrem/gokan/internal/shared/config"
	"github.com/aliakrem/gokan/internal/shared/database"
	"github.com/aliakrem/gokan/internal/shared/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type AppModule struct {
	config      *config.Config
	database    *database.MongoDB
	redisClient *redis.Client
	router      *gin.Engine
	logger      *zerolog.Logger

	userRepo    *user.Repository
	roomRepo    *room.Repository
	messageRepo *message.Repository

	userService    user.Service
	roomService    room.Service
	messageService message.Service
	ticketService  ticket.Service

	userController    *user.Controller
	roomController    *room.Controller
	messageController *message.Controller
	ticketController  *ticket.Controller

	jwtMiddleware      *middleware.JWTMiddleware
	userSyncMiddleware *middleware.UserSyncMiddleware

	wsGateway *websocket.Gateway

	ticketSvc *ticket.TicketService

	startTime time.Time
}

type Dependencies struct {
	Config      *config.Config
	Database    *database.MongoDB
	RedisClient *redis.Client
	Router      *gin.Engine
}

func NewAppModule(deps Dependencies) (*AppModule, error) {
	if deps.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if deps.Database == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	if deps.RedisClient == nil {
		return nil, fmt.Errorf("Redis client is required")
	}
	if deps.Router == nil {
		return nil, fmt.Errorf("router is required")
	}

	app := &AppModule{
		config:      deps.Config,
		database:    deps.Database,
		redisClient: deps.RedisClient,
		router:      deps.Router,
		logger:      &log.Logger,
		startTime:   time.Now(),
	}

	if err := app.initializeRepositories(); err != nil {
		return nil, err
	}

	if err := app.initializeCoreServices(); err != nil {
		return nil, err
	}

	if err := app.initializeMiddleware(); err != nil {
		return nil, err
	}

	if err := app.initializeServices(); err != nil {
		return nil, err
	}

	if err := app.initializeControllers(); err != nil {
		return nil, err
	}

	if err := app.initializeWebSocketGateway(); err != nil {
		return nil, err
	}

	return app, nil
}

func (app *AppModule) initializeRepositories() error {
	var err error

	if app.database == nil {
		return fmt.Errorf("database connection is required")
	}

	app.userRepo, err = user.NewRepository(app.database.Database)
	if err != nil {
		app.logger.Error().Err(err).Msg("failed to create user repository")
		return err
	}

	app.roomRepo, err = room.NewRepository(app.database.Database)
	if err != nil {
		app.logger.Error().Err(err).Msg("failed to create room repository")
		return err
	}

	app.messageRepo, err = message.NewRepository(app.database.Database)
	if err != nil {
		app.logger.Error().Err(err).Msg("failed to create message repository")
		return err
	}

	app.logger.Info().Msg("repositories initialized successfully")
	return nil
}

func (app *AppModule) initializeCoreServices() error {
	app.ticketSvc = ticket.NewTicketService(app.redisClient, app.config.WSTicketTTL())

	app.logger.Info().Msg("core services initialized successfully")
	return nil
}

func (app *AppModule) initializeMiddleware() error {
	app.jwtMiddleware = middleware.NewJWTMiddleware(middleware.JWTConfig{
		Secret:   app.config.JWTSecret,
		ClaimKey: app.config.JWTClaimKey,
		JWKSURL:  app.config.JWKSURL,
	})

	app.userSyncMiddleware = middleware.NewUserSyncMiddleware(
		middleware.UserSyncConfig{
			UserInfoURL: app.config.UserInfoURL,
			SyncTTL:     app.config.UserSyncTTL(),
		},
		app.userRepo,
	)

	app.logger.Info().Msg("middleware initialized successfully")
	return nil
}

func (app *AppModule) initializeServices() error {
	app.userService = user.NewUserService(app.userRepo, app.logger)

	app.roomService = room.NewRoomService(app.roomRepo, app.userRepo, app.logger)

	app.messageService = message.NewMessageService(app.messageRepo, app.roomRepo, app.logger)

	app.ticketService = ticket.NewService(app.ticketSvc, app.logger)

	app.logger.Info().Msg("services initialized successfully")
	return nil
}

func (app *AppModule) initializeControllers() error {
	app.userController = user.NewController(app.userService)
	app.roomController = room.NewController(app.roomService)
	app.messageController = message.NewController(app.messageService)
	app.ticketController = ticket.NewController(app.ticketService)

	app.logger.Info().Msg("controllers initialized successfully")
	return nil
}

func (app *AppModule) initializeWebSocketGateway() error {
	app.wsGateway = websocket.NewGateway(
		app.config,
		app.redisClient,
		app.messageRepo,
		app.roomRepo,
		app.ticketSvc,
	)

	app.logger.Info().Msg("WebSocket gateway initialized successfully")
	return nil
}

func (app *AppModule) GetRouter() *gin.Engine {
	return app.router
}

func (app *AppModule) GetWSGateway() *websocket.Gateway {
	return app.wsGateway
}

func (app *AppModule) GetTicketService() *ticket.TicketService {
	return app.ticketSvc
}

func (app *AppModule) GetDatabase() *database.MongoDB {
	return app.database
}

func (app *AppModule) GetStartTime() time.Time {
	return app.startTime
}

type Module interface {
	RegisterRoutes(router *gin.RouterGroup)
}

func (app *AppModule) RegisterUserModule(router *gin.RouterGroup) {
	if app.userController != nil {
		app.userController.RegisterRoutes(router)
	}
}

func (app *AppModule) RegisterRoomModule(router *gin.RouterGroup) {
	if app.roomController != nil {
		app.roomController.RegisterRoutes(router)
	}
}

func (app *AppModule) RegisterMessageModule(router *gin.RouterGroup) {
	if app.messageController != nil {
		app.messageController.RegisterRoutes(router)
	}
}

func (app *AppModule) RegisterTicketModule(router *gin.RouterGroup) {
	if app.ticketController != nil {
		app.ticketController.RegisterRoutes(router)
	}
}
