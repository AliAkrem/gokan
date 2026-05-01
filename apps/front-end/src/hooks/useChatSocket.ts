import { useEffect, useRef, useState, useCallback } from "react";
import type {
  ConnectionStatus,
  Message,
  WSEvent,
  MessageDeliveredPayload,
  MessageReadPayload,
  ErrorPayload,
  SendMessagePayload,
  MarkReadPayload,
} from "../types";
import { ConnectionStatus as ConnectionStatusEnum } from "../types";
import { getWSTicket } from "../api/client";
import {
  createWSEvent,
  parseWSEvent,
  calculateReconnectDelay,
} from "../utils/websocket";

interface UseChatSocketReturn {
  connectionStatus: ConnectionStatus;
  sendMessage: (content: string, clientMsgId: string) => void;
  markAsRead: (msgId: string) => void;
  error: Error | null;
}

interface UseChatSocketProps {
  jwt: string | null;
  roomId: string | null;
  onMessage: (message: Message) => void;
  onMessageDelivered?: (msgId: string) => void;
  onMessageRead?: (msgId: string) => void;
}

export function useChatSocket({
  jwt,
  roomId,
  onMessage,
  onMessageDelivered,
  onMessageRead,
}: UseChatSocketProps): UseChatSocketReturn {
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>(
    ConnectionStatusEnum.Disconnected,
  );
  const [error, setError] = useState<Error | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef<number>(0);
  const shouldReconnectRef = useRef<boolean>(true);

  // Store ALL callback props in refs so the ws.onmessage handler (set once
  // at connection time) always invokes the latest versions.
  const onMessageRef = useRef(onMessage);
  const onMessageDeliveredRef = useRef(onMessageDelivered);
  const onMessageReadRef = useRef(onMessageRead);

  // Sync refs on every render — not in useEffect, so they're updated
  // synchronously before any WebSocket event can fire.
  onMessageRef.current = onMessage;
  onMessageDeliveredRef.current = onMessageDelivered;
  onMessageReadRef.current = onMessageRead;

  // Get WebSocket URL from environment
  const WS_BASE_URL = import.meta.env.VITE_CHAT_WS_URL || "localhost:8080";

  /**
   * Sends a WebSocket event
   */
  const sendWSEvent = useCallback((event: WSEvent) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(event));
    } else {
      console.warn("Cannot send message: WebSocket is not open");
    }
  }, []);

  /**
   * Ref-based event handler — always reads the latest callbacks from refs.
   * The ws.onmessage handler delegates to this ref so it never goes stale.
   */
  const handleEventRef = useRef((_wsEvent: WSEvent) => {});
  handleEventRef.current = (wsEvent: WSEvent) => {
    switch (wsEvent.event) {
      case "ping":
        sendWSEvent(createWSEvent("pong", {}));
        break;

      case "message_received": {
        const message = wsEvent.payload as any;
        if (message && message.msg_id) {
          const normalizedMessage: Message = {
            ...message,
            created_at:
              typeof message.created_at === "number"
                ? new Date(message.created_at).toISOString()
                : message.created_at,
            updated_at:
              typeof message.updated_at === "number"
                ? new Date(message.updated_at).toISOString()
                : message.updated_at,
          };
          onMessageRef.current(normalizedMessage);
        }
        break;
      }

      case "message_delivered": {
        const payload = wsEvent.payload as MessageDeliveredPayload;
        if (onMessageDeliveredRef.current && payload.msg_id) {
          onMessageDeliveredRef.current(payload.msg_id);
        }
        break;
      }

      case "message_read": {
        const payload = wsEvent.payload as MessageReadPayload;
        if (onMessageReadRef.current && payload.msg_id) {
          onMessageReadRef.current(payload.msg_id);
        }
        break;
      }

      case "error": {
        const payload = wsEvent.payload as ErrorPayload;
        const errorMessage = `WebSocket error: ${payload.code} - ${payload.message}`;
        console.error(errorMessage);
        setError(new Error(errorMessage));
        break;
      }

      default:
        console.log("Unrecognized WebSocket event:", wsEvent.event);
        break;
    }
  };

  /**
   * Establishes WebSocket connection with a ticket.
   * The onmessage handler is a thin wrapper that delegates to handleEventRef,
   * so it never goes stale even though it's set only once per connection.
   */
  const connectWebSocket = useCallback(async () => {
    if (!jwt || !roomId) {
      return;
    }

    try {
      // Request WebSocket ticket
      const ticketResponse = await getWSTicket(jwt);
      const ticket = ticketResponse.ticket;

      // Open WebSocket connection
      const wsUrl = `ws://${WS_BASE_URL}/ws?ticket=${ticket}`;
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log("WebSocket connected");
        setConnectionStatus(ConnectionStatusEnum.Connected);
        setError(null);
        reconnectAttemptsRef.current = 0;
      };

      ws.onmessage = (event) => {
        try {
          const wsEvent = parseWSEvent(event.data);
          // Always dispatch through the ref — never stale
          handleEventRef.current(wsEvent);
        } catch (err) {
          console.error("Failed to parse WebSocket message:", err);
          setError(
            err instanceof Error ? err : new Error("Failed to parse message"),
          );
        }
      };

      ws.onerror = (event) => {
        console.error("WebSocket error:", event);
        setError(new Error("WebSocket connection error"));
      };

      ws.onclose = () => {
        console.log("WebSocket closed");
        wsRef.current = null;

        if (shouldReconnectRef.current) {
          handleReconnect();
        } else {
          setConnectionStatus(ConnectionStatusEnum.Disconnected);
        }
      };
    } catch (err) {
      console.error("Failed to connect WebSocket:", err);
      setError(err instanceof Error ? err : new Error("Failed to connect"));

      if (shouldReconnectRef.current) {
        handleReconnect();
      } else {
        setConnectionStatus(ConnectionStatusEnum.Disconnected);
      }
    }
  }, [jwt, roomId, WS_BASE_URL]);

  /**
   * Handles reconnection with exponential backoff
   */
  const handleReconnect = useCallback(() => {
    setConnectionStatus(ConnectionStatusEnum.Reconnecting);

    const delay = calculateReconnectDelay(reconnectAttemptsRef.current);
    reconnectAttemptsRef.current += 1;

    console.log(
      `Reconnecting in ${delay}ms (attempt ${reconnectAttemptsRef.current})...`,
    );

    reconnectTimeoutRef.current = setTimeout(() => {
      connectWebSocket();
    }, delay);
  }, [connectWebSocket]);

  /**
   * Sends a message to the chat room
   */
  const sendMessage = useCallback(
    (content: string, clientMsgId: string) => {
      if (!roomId) {
        console.warn("Cannot send message: roomId is not available");
        return;
      }

      const payload: SendMessagePayload = {
        room_id: roomId,
        content,
        type: "text",
        client_msg_id: clientMsgId,
      };

      const event = createWSEvent("send_message", payload);
      sendWSEvent(event);
    },
    [roomId, sendWSEvent],
  );

  /**
   * Marks a message as read
   */
  const markAsRead = useCallback(
    (msgId: string) => {
      const payload: MarkReadPayload = {
        msg_id: msgId,
      };

      const event = createWSEvent("mark_read", payload);
      sendWSEvent(event);
    },
    [sendWSEvent],
  );

  /**
   * Initialize WebSocket connection when jwt and roomId are available
   */
  useEffect(() => {
    if (jwt && roomId) {
      shouldReconnectRef.current = true;
      connectWebSocket();
    }

    // Cleanup on unmount or when dependencies change
    return () => {
      shouldReconnectRef.current = false;

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }

      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [jwt, roomId, connectWebSocket]);

  return {
    connectionStatus,
    sendMessage,
    markAsRead,
    error,
  };
}
