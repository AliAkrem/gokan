package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aliakrem/gokan/internal/modules/message/entities"
	roomEntities "github.com/aliakrem/gokan/internal/modules/room/entities"
	"github.com/aliakrem/gokan/internal/shared/config"
	apperrors "github.com/aliakrem/gokan/internal/shared/errors"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *entities.Message) error
	FindByID(ctx context.Context, msgID string) (*entities.Message, error)
	UpdateStatus(ctx context.Context, msgID string, status entities.MessageStatus) error
}

type RoomRepository interface {
	FindByID(ctx context.Context, roomID string) (*roomEntities.Room, error)
	UpdateLastMessage(ctx context.Context, roomID string, msg *entities.Message) error
}

type MessageHandlers struct {
	redisClient *redis.Client
	msgRepo     MessageRepository
	roomRepo    RoomRepository
	gateway     *Gateway
	config      *config.Config
}

func NewMessageHandlers(
	redisClient *redis.Client,
	msgRepo MessageRepository,
	roomRepo RoomRepository,
	gateway *Gateway,
	config *config.Config,
) *MessageHandlers {
	return &MessageHandlers{
		redisClient: redisClient,
		msgRepo:     msgRepo,
		roomRepo:    roomRepo,
		gateway:     gateway,
		config:      config,
	}
}

type SendMessagePayload struct {
	RoomID         string                 `json:"room_id"`
	Content        string                 `json:"content"`
	Type           string                 `json:"type"`
	ClientMsgID    string                 `json:"client_msg_id"`
	RepliedMessage map[string]interface{} `json:"replied_message,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type MarkReadPayload struct {
	MsgID string `json:"msg_id"`
}

func (h *MessageHandlers) HandleSendMessage(conn *WSConnection, payloadMap map[string]interface{}) error {
	ctx := context.Background()

	payloadJSON, err := json.Marshal(payloadMap)
	if err != nil {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "failed to parse payload")
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	var payload SendMessagePayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "failed to parse payload")
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if payload.RoomID == "" {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "room_id is required")
		return fmt.Errorf("roomId is required")
	}
	if payload.Content == "" {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "content is required")
		return fmt.Errorf("content is required")
	}
	if payload.Type == "" {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "type is required")
		return fmt.Errorf("type is required")
	}
	if payload.ClientMsgID == "" {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "client_msg_id is required")
		return fmt.Errorf("client_msg_id is required")
	}

	if payload.Type != "text" && payload.Type != "binary" {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "type must be 'text' or 'binary'")
		return fmt.Errorf("invalid message type: %s", payload.Type)
	}

	room, err := h.roomRepo.FindByID(ctx, payload.RoomID)
	if err != nil {
		h.sendError(conn, apperrors.ErrorCodeRoomNotFound, fmt.Sprintf("room not found: %s", payload.RoomID))
		return fmt.Errorf("room not found: %w", err)
	}

	isParticipant := false
	for _, p := range room.Participants {
		if p == conn.UserID {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		h.sendError(conn, apperrors.ErrorCodeForbidden, "user not in room")
		log.Warn().Str("user_id", conn.UserID).Str("room_id", payload.RoomID).Msg("user attempted to send message to room they're not in")
		return fmt.Errorf("user not in room")
	}

	msg := &entities.Message{
		ClientMsgID:    payload.ClientMsgID,
		RoomID:         payload.RoomID,
		AuthorID:       conn.UserID,
		Text:           &payload.Content,
		Type:           entities.MessageType(payload.Type),
		Status:         entities.MessageStatusSent,
		RepliedMessage: payload.RepliedMessage,
		Metadata:       payload.Metadata,
	}

	if err := h.msgRepo.Create(ctx, msg); err != nil {
		h.sendError(conn, apperrors.ErrorCodeInternalError, "failed to persist message")
		return fmt.Errorf("failed to persist message: %w", err)
	}

	if err := h.roomRepo.UpdateLastMessage(ctx, payload.RoomID, msg); err != nil {
		log.Error().Err(err).Str("room_id", payload.RoomID).Msg("failed to update room lastMessages")
	}

	go h.deliverMessage(msg, room)

	return nil
}

func (h *MessageHandlers) deliverMessage(msg *entities.Message, room *roomEntities.Room) {
	ctx := context.Background()

	var recipientID string
	for _, p := range room.Participants {
		if p != msg.AuthorID {
			recipientID = p
			break
		}
	}

	if recipientID == "" {
		log.Error().Str("msg_id", msg.MsgID).Str("room_id", msg.RoomID).Msg("no recipient found")
		return
	}

	messageReceivedEvent := WSEvent{
		Event:   "message_received",
		Payload: h.messageToPayload(msg),
		TS:      time.Now().Format(time.RFC3339),
	}
	eventJSON, err := json.Marshal(messageReceivedEvent)
	if err != nil {
		log.Error().Err(err).Str("msg_id", msg.MsgID).Msg("failed to marshal message event")
		return
	}

	// XADD to recipient's stream
	streamKey := fmt.Sprintf("chat:stream:user:%s", recipientID)
	xaddArgs := &redis.XAddArgs{
		Stream: streamKey,
		MaxLen: 1000,
		Approx: true,
		Values: map[string]interface{}{"event": string(eventJSON)},
	}
	streamID, err := h.redisClient.XAdd(ctx, xaddArgs).Result()
	if err != nil {
		log.Error().Err(err).Str("msg_id", msg.MsgID).Msg("failed to XADD message to stream")
		return
	}

	// PUBLISH via real-time Pub/Sub
	pubsubPayload := map[string]interface{}{
		"stream_id": streamID,
		"event":     messageReceivedEvent,
	}
	pubsubJSON, _ := json.Marshal(pubsubPayload)
	receivers, err := h.redisClient.Publish(ctx, fmt.Sprintf("chat:pubsub:user:%s", recipientID), string(pubsubJSON)).Result()

	if err != nil {
		log.Error().Err(err).Str("msg_id", msg.MsgID).Msg("failed to publish message")
		return
	}

	// If receivers > 0, the recipient is connected to some instance
	if receivers > 0 {
		deliveredEvent := WSEvent{
			Event: "message_delivered",
			Payload: map[string]interface{}{
				"msg_id":  msg.MsgID,
				"room_id": msg.RoomID,
			},
			TS: time.Now().Format(time.RFC3339),
		}

		// Send the message_delivered event to the sender's Pub/Sub channel
		senderPubsubPayload := map[string]interface{}{
			"stream_id": "", // No stream for acks
			"event":     deliveredEvent,
		}
		senderPubsubJSON, _ := json.Marshal(senderPubsubPayload)
		if err := h.redisClient.Publish(ctx, fmt.Sprintf("chat:pubsub:user:%s", msg.AuthorID), string(senderPubsubJSON)).Err(); err != nil {
			log.Error().Err(err).Str("msg_id", msg.MsgID).Msg("failed to publish delivery ack")
		}
	} else {
		log.Debug().Str("msg_id", msg.MsgID).Str("recipientId", recipientID).Msg("recipient offline, message queued in stream")
	}
}

func (h *MessageHandlers) HandleMarkRead(conn *WSConnection, payloadMap map[string]interface{}) error {
	ctx := context.Background()

	payloadJSON, err := json.Marshal(payloadMap)
	if err != nil {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "failed to parse payload")
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	var payload MarkReadPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "failed to parse payload")
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	if payload.MsgID == "" {
		h.sendError(conn, apperrors.ErrorCodeInvalidPayload, "msg_id is required")
		return fmt.Errorf("msg_id is required")
	}

	msg, err := h.msgRepo.FindByID(ctx, payload.MsgID)
	if err != nil {
		h.sendError(conn, apperrors.ErrorCodeMessageNotFound, fmt.Sprintf("message not found: %s", payload.MsgID))
		return fmt.Errorf("message not found: %w", err)
	}

	// Verify user is the recipient (participant in room but NOT the author)
	room, err := h.roomRepo.FindByID(ctx, msg.RoomID)
	if err != nil {
		h.sendError(conn, apperrors.ErrorCodeRoomNotFound, "room not found")
		return fmt.Errorf("room not found: %w", err)
	}

	isRecipient := false
	for _, p := range room.Participants {
		if p == conn.UserID && p != msg.AuthorID {
			isRecipient = true
			break
		}
	}

	if !isRecipient {
		h.sendError(conn, apperrors.ErrorCodeForbidden, "not authorized to mark this message as read")
		log.Warn().Str("user_id", conn.UserID).Str("msg_id", msg.MsgID).Msg("unauthorized mark_read attempt")
		return fmt.Errorf("user %s is not recipient of message %s", conn.UserID, msg.MsgID)
	}

	// Status transition guard: only allow forward transitions (sent → delivered → read)
	if msg.Status == entities.MessageStatusRead {
		// Already read, nothing to do
		return nil
	}

	// Update message status to "read"
	if err := h.msgRepo.UpdateStatus(ctx, payload.MsgID, entities.MessageStatusRead); err != nil {
		h.sendError(conn, apperrors.ErrorCodeInternalError, "failed to update message status")
		return fmt.Errorf("failed to update message status: %w", err)
	}

	// Send message_read event to sender via Pub/Sub
	readEvent := WSEvent{
		Event: "message_read",
		Payload: map[string]interface{}{
			"msg_id":  msg.MsgID,
			"room_id": msg.RoomID,
		},
		TS: time.Now().Format(time.RFC3339),
	}

	senderPubsubPayload := map[string]interface{}{
		"stream_id": "",
		"event":     readEvent,
	}
	senderPubsubJSON, _ := json.Marshal(senderPubsubPayload)
	if err := h.redisClient.Publish(ctx, fmt.Sprintf("chat:pubsub:user:%s", msg.AuthorID), string(senderPubsubJSON)).Err(); err != nil {
		log.Error().Err(err).Str("msg_id", msg.MsgID).Msg("failed to publish read ack")
	}

	return nil
}

func (h *MessageHandlers) DeliverOfflineMessages(conn *WSConnection) error {
	ctx := context.Background()
	streamKey := fmt.Sprintf("chat:stream:user:%s", conn.UserID)
	groupName := fmt.Sprintf("cg:user:%s", conn.UserID)
	consumerName := "primary"

	// Ensure consumer group exists (MKSTREAM creates stream if it doesn't exist)
	err := h.redisClient.XGroupCreateMkStream(ctx, streamKey, groupName, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	// Helper to process a batch of messages
	processMessages := func(messages []redis.XMessage) {
		for _, xmsg := range messages {
			eventJSON, ok := xmsg.Values["event"].(string)
			if !ok {
				// Poison pill, discard it
				h.redisClient.XAck(ctx, streamKey, groupName, xmsg.ID)
				continue
			}

			var event WSEvent
			if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
				// Poison pill, discard it
				h.redisClient.XAck(ctx, streamKey, groupName, xmsg.ID)
				continue
			}

			// Try to send to WS
			select {
			case conn.Send <- event:
				// Delivered to WS channel, now XACK
				if err := h.redisClient.XAck(ctx, streamKey, groupName, xmsg.ID).Err(); err != nil {
					log.Error().Err(err).Str("user_id", conn.UserID).Str("streamId", xmsg.ID).Msg("failed to XACK offline message")
				}

				// Also update Mongo status to delivered
				if payloadMap, ok := event.Payload.(map[string]interface{}); ok {
					if msgID, ok := payloadMap["msg_id"].(string); ok {
						if err := h.msgRepo.UpdateStatus(ctx, msgID, entities.MessageStatusDelivered); err != nil {
							log.Error().Err(err).Str("msg_id", msgID).Msg("failed to update message status to delivered")
						}
					}
				}
			case <-conn.Done:
				return
			case <-time.After(5 * time.Second):
				// Channel full, skip and let it sit in pending
				log.Warn().Str("user_id", conn.UserID).Msg("send channel full during offline delivery, skipping message")
			}
		}
	}

	// 1. Read pending messages (from start of pending queue)
	startID := "0"
	for {
		streams, err := h.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: consumerName,
			Streams:  []string{streamKey, startID},
			Count:    100,
			Block:    -1,
		}).Result()

		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to read pending messages: %w", err)
		}

		if len(streams) == 0 || len(streams[0].Messages) == 0 {
			break
		}

		processMessages(streams[0].Messages)

		// Update startID for the next iteration to paginate through pending messages
		startID = streams[0].Messages[len(streams[0].Messages)-1].ID
	}

	// 2. Read new undelivered messages
	for {
		streams, err := h.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: consumerName,
			Streams:  []string{streamKey, ">"},
			Count:    100,
			Block:    -1,
		}).Result()

		if err != nil && err != redis.Nil {
			return fmt.Errorf("failed to read new messages: %w", err)
		}

		if len(streams) == 0 || len(streams[0].Messages) == 0 {
			break
		}
		processMessages(streams[0].Messages)
	}

	log.Info().Str("user_id", conn.UserID).Msg("processed offline messages")
	return nil
}

func (h *MessageHandlers) messageToPayload(msg *entities.Message) map[string]interface{} {
	payload := map[string]interface{}{
		"msg_id":        msg.MsgID,
		"client_msg_id": msg.ClientMsgID,
		"room_id":       msg.RoomID,
		"author_id":     msg.AuthorID,
		"type":          msg.Type,
		"status":        msg.Status,
		"created_at":    msg.CreatedAt,
		"updated_at":    msg.UpdatedAt,
	}

	if msg.Text != nil {
		payload["text"] = *msg.Text
	}

	if msg.RepliedMessage != nil {
		payload["replied_message"] = msg.RepliedMessage
	}

	if msg.Metadata != nil {
		payload["metadata"] = msg.Metadata
	}

	return payload
}

func (h *MessageHandlers) sendError(conn *WSConnection, code apperrors.ErrorCode, message string) {
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
