package app

import (
	"context"
	"net/http"
	"time"

	"github.com/aliakrem/gokan/internal/shared/middleware"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (app *AppModule) ConfigureRoutes() {
	app.configureGlobalMiddleware()

	app.configureHealthEndpoints()

	app.configureWebSocketEndpoint()

	app.configureAPIRoutes()
}

func (app *AppModule) configureGlobalMiddleware() {
	app.router.Use(gin.Recovery())

	app.router.Use(middleware.ErrorHandlerMiddleware())

	corsConfig := cors.DefaultConfig()
	if len(app.config.AllowedOrigins) == 1 && app.config.AllowedOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = app.config.AllowedOrigins
	}
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization", "apikey")
	app.router.Use(cors.New(corsConfig))

	if app.logger != nil {
		app.logger.Info().Msg("global middleware configured")
	}
}

func (app *AppModule) configureHealthEndpoints() {
	app.router.GET("/health", app.healthHandler)

	app.router.GET("/metrics", app.metricsHandler)

	if app.logger != nil {
		app.logger.Info().Msg("health and metrics endpoints configured")
	}
}

func (app *AppModule) configureWebSocketEndpoint() {
	if app.wsGateway != nil {
		app.router.GET("/ws", app.wsGateway.HandleConnection)
		app.logger.Info().Msg("WebSocket endpoint configured")
	} else {
		app.router.GET("/ws", func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "WebSocket service not available",
			})
		})
		if app.logger != nil {
			app.logger.Warn().Msg("WebSocket endpoint configured with placeholder (wsGateway not available)")
		}
	}
}

func (app *AppModule) configureAPIRoutes() {
	v1 := app.router.Group("/api/v1")

	if app.jwtMiddleware != nil {
		v1.Use(app.jwtMiddleware.VerifyToken())
	}
	if app.userSyncMiddleware != nil {
		v1.Use(app.userSyncMiddleware.SyncUser())
	}

	app.RegisterTicketModule(v1)
	app.RegisterUserModule(v1)
	app.RegisterRoomModule(v1)
	app.RegisterMessageModule(v1)

	if app.logger != nil {
		app.logger.Info().Msg("API v1 routes configured")
	}
}

func (app *AppModule) healthHandler(c *gin.Context) {
	healthCtx, healthCancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer healthCancel()

	health := gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"services":  gin.H{},
	}

	if app.database != nil && app.database.Client != nil {
		if err := app.database.Client.Ping(healthCtx, nil); err != nil {
			health["status"] = "degraded"
			health["services"].(gin.H)["mongodb"] = gin.H{"status": "down", "error": err.Error()}
		} else {
			health["services"].(gin.H)["mongodb"] = gin.H{"status": "up"}
		}
	}

	if app.redisClient != nil {
		if err := app.redisClient.Ping(healthCtx).Err(); err != nil {
			health["status"] = "degraded"
			health["services"].(gin.H)["redis"] = gin.H{"status": "down", "error": err.Error()}
		} else {
			health["services"].(gin.H)["redis"] = gin.H{"status": "up"}
		}
	}

	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

func (app *AppModule) metricsHandler(c *gin.Context) {
	metrics := gin.H{
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
		"uptime_seconds": time.Since(app.startTime).Seconds(),
	}

	if app.wsGateway != nil {
		metrics["websocket_connections"] = app.wsGateway.GetConnectionCount()
	} else {
		metrics["websocket_connections"] = 0
	}

	c.JSON(http.StatusOK, metrics)
}
