import React, { useState, useEffect, useCallback } from "react";
import { useChatRoom } from "../hooks/useChatRoom";
import { useChatMessages } from "../hooks/useChatMessages";
import { useChatSocket } from "../hooks/useChatSocket";
import { PanelHeader } from "./PanelHeader";
import { MessageList } from "./MessageList";
import { MessageInput } from "./MessageInput";
import { getDisplayName } from "../utils/displayName";
import { generateClientMsgId } from "../utils/websocket";
import type { Message, ConnectionStatus } from "../types";
import { useSyncUser } from "../hooks/useSyncUser";

interface ChatPanelProps {
  side: "left" | "right";
  jwt: string;
  userId: string;
  leftUserId: string;
  rightUserId: string;
}

export const ChatPanel: React.FC<ChatPanelProps> = ({
  side,
  jwt,
  userId,
  leftUserId,
  rightUserId,
}) => {
  const { isLoading: SyncingUser } = useSyncUser({
    jwt: jwt,
    userId,
  });

  const {
    roomId,
    isLoading: roomLoading,
    error: roomError,
  } = useChatRoom({
    jwt,
    side,
    leftUserId,
    rightUserId,
  });

  // Message history
  const { messages, setMessages } = useChatMessages(jwt, roomId);

  // Local state
  const [username, setUsername] = useState<string>(() => getDisplayName(side));
  const [connectionStatus, setConnectionStatus] =
    useState<ConnectionStatus>("Disconnected");

  // Handle incoming messages
  const onMessageReceived = useCallback(
    (message: Message) => {
      setMessages((prevMessages) => {
        // Check if message already exists (by msg_id or client_msg_id)
        const existingIndex = prevMessages.findIndex(
          (m) =>
            m.msg_id === message.msg_id ||
            (message.client_msg_id &&
              m.client_msg_id === message.client_msg_id),
        );

        if (existingIndex !== -1) {
          // Update existing message
          const updated = [...prevMessages];
          updated[existingIndex] = message;
          return updated;
        }

        // Add new message
        return [...prevMessages, message];
      });
    },
    [setMessages],
  );

  // Handle message status updates
  const onMessageDelivered = useCallback(
    (msgId: string) => {
      setMessages((prevMessages) =>
        prevMessages.map((m) =>
          m.msg_id === msgId ? { ...m, status: "delivered" as const } : m,
        ),
      );
    },
    [setMessages],
  );

  const onMessageRead = useCallback(
    (msgId: string) => {
      setMessages((prevMessages) =>
        prevMessages.map((m) =>
          m.msg_id === msgId ? { ...m, status: "read" as const } : m,
        ),
      );
    },
    [setMessages],
  );

  // WebSocket connection
  const {
    connectionStatus: wsConnectionStatus,
    sendMessage,
    markAsRead,
  } = useChatSocket({
    jwt,
    roomId,
    onMessage: onMessageReceived,
    onMessageDelivered,
    onMessageRead,
  });

  // Automatically send mark_read event for messages from other users
  useEffect(() => {
    const lastMessage = messages[messages.length - 1];
    if (lastMessage && lastMessage.author_id !== userId) {
      markAsRead(lastMessage.msg_id);
    }
  }, [messages, userId, markAsRead]);

  // Update connection status from WebSocket hook
  useEffect(() => {
    setConnectionStatus(wsConnectionStatus);
  }, [wsConnectionStatus]);

  // Handle username change
  const handleUsernameChange = useCallback(
    (newUsername: string) => {
      setUsername(newUsername);
      const storageKey = `chat_${side}_username`;
      localStorage.setItem(storageKey, newUsername);
    },
    [side],
  );

  // Handle send message
  const handleSendMessage = useCallback(
    (content: string) => {
      if (!roomId || !userId) {
        console.error("Cannot send message: roomId or userId not available");
        return;
      }

      const clientMsgId = generateClientMsgId();

      // Optimistically add message to UI
      const optimisticMessage: Message = {
        msg_id: clientMsgId, // Temporary ID until server responds
        client_msg_id: clientMsgId,
        room_id: roomId,
        author_id: userId,
        text: content,
        type: "text",
        status: "sent",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      setMessages((prev) => [...prev, optimisticMessage]);
      sendMessage(content, clientMsgId);
    },
    [roomId, userId, sendMessage, setMessages],
  );

  // Render loading state
  if (roomLoading || SyncingUser) {
    return (
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          height: "100vh",
          backgroundColor: "#1b1c1d",
          color: "#e4e6eb",
          justifyContent: "center",
          alignItems: "center",
        }}
      >
        <div>Loading...</div>
      </div>
    );
  }

  // Render error state
  if (roomError) {
    return (
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          height: "100vh",
          backgroundColor: "#1b1c1d",
          color: "#e4e6eb",
          justifyContent: "center",
          alignItems: "center",
        }}
      >
        <div style={{ color: "#ff4444" }}>Error: {roomError?.message}</div>
      </div>
    );
  }

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        height: "100vh",
        backgroundColor: "#1b1c1d",
        overflow: "hidden",
      }}
    >
      <PanelHeader
        username={username}
        onUsernameChange={handleUsernameChange}
        connectionStatus={connectionStatus}
      />

      <div style={{ flex: 1, overflow: "scroll" }}>
        <MessageList messages={messages} currentUserId={userId || ""} />
      </div>

      <MessageInput
        onSendMessage={handleSendMessage}
        disabled={connectionStatus !== "Connected"}
      />
    </div>
  );
};
