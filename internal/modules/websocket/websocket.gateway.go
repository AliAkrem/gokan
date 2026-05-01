package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aliakrem/gokan/internal/modules/message/entities"
	"github.com/aliakrem/gokan/internal/shared/config"
	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type TicketService interface {
	ValidateTicket(ctx context.Context, ticket string) (user_id string, jwt string, err error)
}

type WSEvent struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
	TS      string      `json:"ts"` // ISO8601 timestamp
}

type WSConnection struct {
	UserID   string
	Conn     *websocket.Conn
	Send     chan WSEvent
	LastPong time.Time
	Mutex    sync.RWMutex
	Done     chan struct{}
	once     sync.Once
}

type Gateway struct {
	connections map[string]*WSConnection
	mutex       sync.RWMutex
	config      *config.Config
	redisClient *redis.Client
	handlers    *MessageHandlers
	ticketSvc   TicketService
}

func NewGateway(cfg *config.Config, redisClient *redis.Client, msgRepo MessageRepository, roomRepo RoomRepository, ticketSvc TicketService) *Gateway {
	gateway := &Gateway{
		connections: make(map[string]*WSConnection),
		config:      cfg,
		redisClient: redisClient,
		ticketSvc:   ticketSvc,
	}

	handlers := NewMessageHandlers(redisClient, msgRepo, roomRepo, gateway, cfg)
	gateway.handlers = handlers

	return gateway
}

func (g *Gateway) HandleConnection(c *gin.Context) {
	ticket := c.Query("ticket")

	if c.Query("token") != "" {
		log.Warn().Str("clientIP", c.ClientIP()).Msg("connection attempt with deprecated token parameter")
		c.JSON(400, gin.H{"error": "token parameter not supported, use ticket"})
		return
	}

	if ticket == "" {
		log.Warn().Str("clientIP", c.ClientIP()).Msg("connection attempt without ticket")
		c.JSON(401, gin.H{"error": "ticket required"})
		return
	}

	userID, jwt, err := g.ticketSvc.ValidateTicket(c.Request.Context(), ticket)
	if err != nil {
		log.Warn().Err(err).Str("clientIP", c.ClientIP()).Msg("ticket validation failed")
		c.JSON(401, gin.H{"error": "invalid or expired ticket"})
		return
	}

	c.Set("user_id", userID)
	c.Set("jwt", jwt)

	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		CompressionMode:    websocket.CompressionDisabled,
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("failed to upgrade connection")
		return
	}

	// Set read limit
	conn.SetReadLimit(int64(g.config.MaxMessageBytes))

	// Handle duplicate connections - close old connection if exists
	g.mutex.Lock()
	if oldConn, exists := g.connections[userID]; exists {
		log.Info().Str("user_id", userID).Msg("closing old connection for duplicate user")
		oldConn.once.Do(func() { close(oldConn.Done) })
		oldConn.Conn.Close(websocket.StatusNormalClosure, "duplicate connection")
		delete(g.connections, userID)
	}
	// Don't register new connection yet — deliver offline messages first
	g.mutex.Unlock()

	// Create new connection
	wsConn := &WSConnection{
		UserID:   userID,
		Conn:     conn,
		Send:     make(chan WSEvent, 256), // Buffer size limit: 256 messages
		LastPong: time.Now(),
		Done:     make(chan struct{}),
	}

	log.Info().Str("user_id", userID).Msg("websocket connection established")

	go g.writePump(wsConn)
	go g.heartbeat(wsConn)

	// Deliver offline messages, then register and start read pump.
	// This ensures offline messages are enqueued on conn.Send before
	// the connection is visible to other users (preventing out-of-order delivery).
	go func() {
		if g.handlers != nil {
			if err := g.handlers.DeliverOfflineMessages(wsConn); err != nil {
				log.Error().Err(err).Str("user_id", userID).Msg("failed to deliver offline messages")
			}
		}

		// Register connection — now other users can route messages here
		g.mutex.Lock()
		g.connections[userID] = wsConn
		g.mutex.Unlock()

		// Subscribe to real-time events for this user across all instances
		if g.redisClient != nil {
			pubsub := g.redisClient.Subscribe(context.Background(), fmt.Sprintf("chat:pubsub:user:%s", userID))
			go g.pubsubPump(wsConn, pubsub)
		}

		// Start read pump (blocks until connection closes)
		g.readPump(wsConn)
	}()
}

func (g *Gateway) GetConnection(userID string) (*WSConnection, bool) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	conn, exists := g.connections[userID]
	return conn, exists
}

func (g *Gateway) GetConnectionCount() int {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return len(g.connections)
}

func (g *Gateway) CloseConnection(userID string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	conn, exists := g.connections[userID]
	if !exists {
		return fmt.Errorf("connection not found for user: %s", userID)
	}

	conn.once.Do(func() { close(conn.Done) })
	conn.Conn.Close(websocket.StatusNormalClosure, "connection closed")
	delete(g.connections, userID)

	log.Info().Str("user_id", userID).Msg("connection closed")
	return nil
}

func (g *Gateway) removeConnection(userID string, conn *WSConnection) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if existing, ok := g.connections[userID]; ok && existing == conn {
		delete(g.connections, userID)
		log.Info().Str("user_id", userID).Msg("connection removed from map")
	}
}

func (g *Gateway) readPump(conn *WSConnection) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-conn.Done
		cancel()
	}()

	defer func() {
		conn.once.Do(func() { close(conn.Done) })
		g.removeConnection(conn.UserID, conn)
		conn.Conn.Close(websocket.StatusNormalClosure, "read pump closed")
	}()

	for {
		msgType, data, err := conn.Conn.Read(ctx)
		if err != nil {
			log.Error().Err(err).Str("user_id", conn.UserID).Msg("error reading from websocket")
			return
		}

		if msgType != websocket.MessageText {
			log.Warn().Str("user_id", conn.UserID).Int("msgType", int(msgType)).Msg("unsupported message type")
			continue
		}

		var event WSEvent
		if err := json.Unmarshal(data, &event); err != nil {
			log.Error().Err(err).Str("user_id", conn.UserID).Msg("failed to parse event")
			g.sendError(conn, apperrors.ErrorCodeInvalidPayload, "failed to parse event")
			continue
		}

		if event.Event == "pong" {
			conn.Mutex.Lock()
			conn.LastPong = time.Now()
			conn.Mutex.Unlock()
			continue
		}

		if g.handlers != nil {
			var err error
			switch event.Event {
			case "send_message":
				if payloadMap, ok := event.Payload.(map[string]interface{}); ok {
					err = g.handlers.HandleSendMessage(conn, payloadMap)
				} else {
					err = fmt.Errorf("invalid payload format")
				}
			case "mark_read":
				if payloadMap, ok := event.Payload.(map[string]interface{}); ok {
					err = g.handlers.HandleMarkRead(conn, payloadMap)
				} else {
					err = fmt.Errorf("invalid payload format")
				}
			default:
				log.Warn().Str("user_id", conn.UserID).Str("event", event.Event).Msg("unknown event type")
				g.sendError(conn, apperrors.ErrorCodeInvalidPayload, fmt.Sprintf("unknown event type: %s", event.Event))
				continue
			}

			if err != nil {
				log.Error().Err(err).Str("user_id", conn.UserID).Str("event", event.Event).Msg("error handling event")
				g.sendError(conn, apperrors.ErrorCodeInternalError, "an internal error occurred")
			}
		} else {
			log.Debug().Str("user_id", conn.UserID).Str("event", event.Event).Msg("received event (no handlers configured)")
		}
	}
}

func (g *Gateway) writePump(conn *WSConnection) {
	defer func() {
		conn.Conn.Close(websocket.StatusNormalClosure, "write pump closed")
	}()

	for {
		select {
		case <-conn.Done:
			return
		case event, ok := <-conn.Send:
			if !ok {
				// Channel closed
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				log.Error().Err(err).Str("user_id", conn.UserID).Msg("failed to marshal event")
				continue
			}

			writeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := conn.Conn.Write(writeCtx, websocket.MessageText, data); err != nil {
				cancel()
				log.Error().Err(err).Str("user_id", conn.UserID).Msg("error writing to websocket")
				return
			}
			cancel()
		}
	}
}

func (g *Gateway) heartbeat(conn *WSConnection) {
	ticker := time.NewTicker(g.config.PingInterval())
	defer ticker.Stop()

	for {
		select {
		case <-conn.Done:
			return
		case <-ticker.C:
			pingSentAt := time.Now()

			pingEvent := WSEvent{
				Event:   "ping",
				Payload: map[string]interface{}{},
				TS:      pingSentAt.Format(time.RFC3339),
			}

			select {
			case conn.Send <- pingEvent:
				// Ping queued successfully
			case <-conn.Done:
				return
			}

			// Wait for pong with timeout
			pongTimer := time.NewTimer(g.config.PongTimeout())
			select {
			case <-pongTimer.C:
				// Check if a pong was received AFTER the ping was sent
				conn.Mutex.RLock()
				lastPong := conn.LastPong
				conn.Mutex.RUnlock()

				if lastPong.Before(pingSentAt) {
					// No pong received since we sent the ping — client is dead
					log.Warn().Str("user_id", conn.UserID).Msg("pong timeout - closing connection")
					conn.once.Do(func() { close(conn.Done) })
					conn.Conn.Close(websocket.StatusNormalClosure, "pong timeout")
					g.removeConnection(conn.UserID, conn)
					pongTimer.Stop()
					return
				}
			case <-conn.Done:
				pongTimer.Stop()
				return
			}
			pongTimer.Stop()
		}
	}
}

// sendError sends an error event to the client
func (g *Gateway) sendError(conn *WSConnection, code apperrors.ErrorCode, message string) {
	errorEvent := WSEvent{
		Event: "error",
		Payload: map[string]string{
			"code":    string(code),
			"message": message,
		},
		TS: time.Now().Format(time.RFC3339),
	}

	select {
	case conn.Send <- errorEvent:
		// Error sent
	case <-conn.Done:
		// Connection closed
	default:
		// Channel full, log and skip
		log.Warn().Str("user_id", conn.UserID).Msg("send channel full, dropping error event")
	}
}

// Shutdown gracefully shuts down the WebSocket gateway
func (g *Gateway) Shutdown(ctx context.Context) error {
	log.Info().Msg("shutting down WebSocket gateway")

	g.mutex.Lock()
	connections := make([]*WSConnection, 0, len(g.connections))
	for _, conn := range g.connections {
		connections = append(connections, conn)
	}
	g.mutex.Unlock()

	// Create wait group to track all connection shutdowns
	var wg sync.WaitGroup

	for _, conn := range connections {
		wg.Add(1)
		go func(c *WSConnection) {
			defer wg.Done()

			// Signal connection to stop
			c.once.Do(func() { close(c.Done) })

			// Drain in-flight messages from send channel
			drainCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			drained := 0
		drainLoop:
			for {
				select {
				case event, ok := <-c.Send:
					if !ok {
						break drainLoop
					}
					// Try to send the message
					data, err := json.Marshal(event)
					if err != nil {
						log.Error().Err(err).Str("user_id", c.UserID).Msg("failed to marshal event during drain")
						continue
					}
					if err := c.Conn.Write(drainCtx, websocket.MessageText, data); err != nil {
						log.Error().Err(err).Str("user_id", c.UserID).Msg("failed to send message during drain")
						break drainLoop
					}
					drained++
				case <-drainCtx.Done():
					// Timeout reached, stop draining
					remaining := len(c.Send)
					if remaining > 0 {
						log.Warn().Str("user_id", c.UserID).Int("remaining", remaining).Msg("timeout draining messages")
					}
					break drainLoop
				default:
					// Channel is empty
					break drainLoop
				}
			}

			if drained > 0 {
				log.Info().Str("user_id", c.UserID).Int("count", drained).Msg("drained in-flight messages")
			}

			// Close WebSocket connection
			c.Conn.Close(websocket.StatusNormalClosure, "server shutdown")
			log.Info().Str("user_id", c.UserID).Msg("connection closed during shutdown")
		}(conn)
	}

	// Wait for all connections to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("all WebSocket connections closed")
	case <-ctx.Done():
		log.Warn().Msg("shutdown timeout reached, forcing connection closure")
	}

	// Clear connections map
	g.mutex.Lock()
	g.connections = make(map[string]*WSConnection)
	g.mutex.Unlock()

	return nil
}

// pubsubPump listens for real-time messages published to the user's Pub/Sub channel
func (g *Gateway) pubsubPump(conn *WSConnection, pubsub *redis.PubSub) {
	ctx := context.Background()
	ch := pubsub.Channel()

	defer pubsub.Close()

	for {
		select {
		case <-conn.Done:
			return
		case msg := <-ch:
			// Parse the wrapper payload
			var payload struct {
				StreamID string  `json:"stream_id"`
				Event    WSEvent `json:"event"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
				log.Error().Err(err).Str("user_id", conn.UserID).Msg("failed to unmarshal pubsub message")
				continue
			}

			select {
			case conn.Send <- payload.Event:
				// Successfully enqueued to WS. If there's a StreamID, it's a chat message and needs to be XACKed
				if payload.StreamID != "" {
					streamKey := fmt.Sprintf("chat:stream:user:%s", conn.UserID)
					groupName := fmt.Sprintf("cg:user:%s", conn.UserID)

					if err := g.redisClient.XAck(ctx, streamKey, groupName, payload.StreamID).Err(); err != nil {
						log.Error().Err(err).Str("user_id", conn.UserID).Str("streamId", payload.StreamID).Msg("failed to XACK message via pubsub")
					} else {
						// Update Mongo status to delivered
						if payloadMap, ok := payload.Event.Payload.(map[string]interface{}); ok {
							if msgID, ok := payloadMap["msg_id"].(string); ok {
								if err := g.handlers.msgRepo.UpdateStatus(ctx, msgID, entities.MessageStatusDelivered); err != nil {
									log.Error().Err(err).Str("msg_id", msgID).Msg("failed to update message status to delivered via pubsub")
								}
							}
						}
					}
				}
			case <-conn.Done:
				return
			default:
				// Channel full, message will be caught on next reconnect via XREADGROUP
				log.Warn().Str("user_id", conn.UserID).Msg("send channel full during real-time pubsub delivery")
			}
		}
	}
}
