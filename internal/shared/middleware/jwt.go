package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
)

type JWTConfig struct {
	Secret   string // HS256 signing secret (used when JWKSURL is empty)
	ClaimKey string // JWT claim containing user_id (default: "sub")
	JWKSURL  string // JWKS endpoint URL for asymmetric key verification (e.g. ES256)
}

type JWTMiddleware struct {
	config JWTConfig
	jwksKf keyfunc.Keyfunc
}

func NewJWTMiddleware(config JWTConfig) *JWTMiddleware {
	if config.ClaimKey == "" {
		config.ClaimKey = "sub"
	}

	m := &JWTMiddleware{
		config: config,
	}

	if config.JWKSURL != "" {
		kf, err := keyfunc.NewDefault([]string{config.JWKSURL})
		if err != nil {
			log.Fatal().
				Err(err).
				Str("jwksURL", config.JWKSURL).
				Msg("failed to initialize JWKS keyfunc – check that the URL is reachable")
		}
		m.jwksKf = kf
		log.Info().Str("jwksURL", config.JWKSURL).Msg("JWKS keyfunc initialized for JWT verification")
	} else {
		log.Info().Msg("using HS256 shared secret for JWT verification")
	}

	return m
}

func (m *JWTMiddleware) VerifyToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := m.extractToken(c)
		log.Debug().Bool("jwks_initialized", m.jwksKf != nil).Msg("verifying token")
		if tokenString == "" {
			log.Warn().
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Str("reason", "JWT token is missing").
				Msg("authentication failure")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"code":    "INVALID_JWT",
				"details": map[string]interface{}{"reason": "JWT token is missing"},
			})
			c.Abort()
			return
		}

		parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256", "ES256"}))
		token, err := parser.Parse(tokenString, m.keyFunc())

		if err != nil {
			log.Warn().
				Err(err).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Str("reason", "JWT verification failed").
				Msg("authentication failure")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"code":    "INVALID_JWT",
				"details": map[string]interface{}{"reason": "JWT verification failed"},
			})
			c.Abort()
			return
		}

		if !token.Valid {
			log.Warn().
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Str("reason", "JWT token is invalid").
				Msg("authentication failure")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"code":    "INVALID_JWT",
				"details": map[string]interface{}{"reason": "JWT token is invalid"},
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Warn().
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Str("reason", "Failed to parse JWT claims").
				Msg("authentication failure")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"code":    "INVALID_JWT",
				"details": map[string]interface{}{"reason": "Failed to parse JWT claims"},
			})
			c.Abort()
			return
		}

		userID, ok := claims[m.config.ClaimKey].(string)
		if !ok || userID == "" {
			log.Warn().
				Str("claimKey", m.config.ClaimKey).
				Str("path", c.Request.URL.Path).
				Str("method", c.Request.Method).
				Str("reason", fmt.Sprintf("user_id not found in claim '%s'", m.config.ClaimKey)).
				Msg("authentication failure")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"code":    "INVALID_JWT",
				"details": map[string]interface{}{"reason": fmt.Sprintf("user_id not found in claim '%s'", m.config.ClaimKey)},
			})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("jwt", tokenString)

		log.Debug().Str("user_id", userID).Msg("JWT verified successfully")
		c.Next()
	}
}

func (m *JWTMiddleware) keyFunc() jwt.Keyfunc {
	if m.jwksKf != nil {
		return func(token *jwt.Token) (interface{}, error) {
			key, err := m.jwksKf.Keyfunc(token)
			if err == nil {
				return key, nil
			}
			log.Debug().Err(err).Msg("JWKS keyfunc failed, falling back to HS256 if possible")

			if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
				return []byte(m.config.Secret), nil
			}
			return nil, err
		}
	}

	return func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	}
}

func (m *JWTMiddleware) extractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	return ""
}
