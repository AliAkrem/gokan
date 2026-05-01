package main

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aliakrem/gokan/internal/app"
	"github.com/aliakrem/gokan/internal/shared/config"
	"github.com/aliakrem/gokan/internal/shared/database"
	"github.com/aliakrem/gokan/internal/shared/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := loadConfiguration()
	setupLogger(cfg)
	db := connectToMongoDB(cfg)
	redisClient := connectToRedis(cfg)
	router := initializeGinRouter(cfg)
	appModule := createAppModule(cfg, db, redisClient, router)
	server := createHTTPServer(cfg, appModule)
	startServerInBackground(cfg, server)
	waitForShutdownSignal()
	shutdownGracefully(appModule, server)
}

func loadConfiguration() *config.Config {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load cfg")
	}
	return cfg
}

func setupLogger(cfg *config.Config) {
	logger.Setup(cfg.LogLevel)
}

func connectToMongoDB(cfg *config.Config) *database.MongoDB {
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer mongoCancel()

	db, err := database.ConnectMongoDB(mongoCtx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to MongoDB")
	}
	return db
}

func connectToRedis(cfg *config.Config) *redis.Client {
	redisOpts := &redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	if cfg.RedisUseTLS {
		redisOpts.TLSConfig = &tls.Config{}
	}

	redisClient := redis.NewClient(redisOpts)

	redisCtx, redisCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer redisCancel()

	if err := redisClient.Ping(redisCtx).Err(); err != nil {
		log.Fatal().Err(err).Str("addr", cfg.RedisAddr()).Bool("tls", cfg.RedisUseTLS).Msg("failed to connect to Redis")
	}
	log.Info().Str("addr", cfg.RedisAddr()).Bool("tls", cfg.RedisUseTLS).Msg("connected to Redis")

	return redisClient
}

func initializeGinRouter(cfg *config.Config) *gin.Engine {
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	return gin.New()
}

func createAppModule(cfg *config.Config, db *database.MongoDB, redisClient *redis.Client, router *gin.Engine) *app.AppModule {
	appModule, err := app.NewAppModule(app.Dependencies{
		Config:      cfg,
		Database:    db,
		RedisClient: redisClient,
		Router:      router,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize app module")
	}

	appModule.ConfigureRoutes()
	return appModule
}

func createHTTPServer(cfg *config.Config, appModule *app.AppModule) *http.Server {
	return &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: appModule.GetRouter(),
	}
}

func startServerInBackground(cfg *config.Config, srv *http.Server) {
	go func() {
		log.Info().Str("port", cfg.Port).Msg("starting Gokan server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()
}

func waitForShutdownSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server...")
}

func shutdownGracefully(appModule *app.AppModule, srv *http.Server) {
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	stopAcceptingNewConnections(srv, shutdownCtx)
	closeAllWebSocketConnections(appModule, shutdownCtx)
	closeRedisConnection(appModule)
	disconnectFromMongoDB(appModule, shutdownCtx)

	log.Info().Msg("server exited gracefully")
}

func stopAcceptingNewConnections(srv *http.Server, ctx context.Context) {
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}
}

func closeAllWebSocketConnections(appModule *app.AppModule, ctx context.Context) {
	if err := appModule.GetWSGateway().Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("failed to shutdown WebSocket gateway")
	}
}

func closeRedisConnection(appModule *app.AppModule) {
	if err := appModule.GetTicketService().Close(); err != nil {
		log.Error().Err(err).Msg("failed to close Redis connection")
	}
}

func disconnectFromMongoDB(appModule *app.AppModule, ctx context.Context) {
	if err := appModule.GetDatabase().Disconnect(ctx); err != nil {
		log.Error().Err(err).Msg("failed to disconnect from MongoDB")
	}
}
