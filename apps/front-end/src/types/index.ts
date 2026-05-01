// Message Interface
export interface Message {
  msg_id: string;
  client_msg_id: string;
  room_id: string;
  author_id: string;
  text: string;
  type: 'text' | 'binary';
  status: 'sent' | 'delivered' | 'read';
  created_at: string; // ISO8601
  updated_at: string; // ISO8601
  replied_message?: {
    msg_id: string;
    text: string;
    author_id: string;
  };
  metadata?: Record<string, any>;
}

// WebSocket Event Envelope
export interface WSEvent<T = any> {
  event: string;
  payload: T;
  ts: string; // ISO8601 timestamp
}

// Connection Status
export const ConnectionStatus = {
  Connected: 'Connected',
  Reconnecting: 'Reconnecting',
  Disconnected: 'Disconnected'
} as const;

export type ConnectionStatus = typeof ConnectionStatus[keyof typeof ConnectionStatus];

// User Interfaces
export interface User {
  id: string;
  username: string;
  full_name: string;
  email: string;
  avatar_url: string;
  website: string;
  updated_at: string; // ISO8601
}

export interface UserMetadata {
  avatar_url: string;
  email: string;
  full_name: string;
  id: string;
  updated_at: string; // ISO8601
  username: string;
  website: string;
}

export interface UserListItem {
  user_id: string;
  metadata: UserMetadata;
  synced_at: number; // Unix timestamp in milliseconds
  created_at: number; // Unix timestamp in milliseconds
  updated_at: number; // Unix timestamp in milliseconds
  last_seen_at: number; // Unix timestamp in milliseconds
}

// Room Interfaces
export interface LastMessage {
  author_id: string;
  created_at: number; // Unix timestamp in milliseconds
  msg_id: string;
  text: string;
  type: 'text' | 'binary';
}

export interface Room {
  room_id: string;
  participants: string[]; // Array of user IDs
  type: '1-to-1' | 'group';
  last_messages: LastMessage;
  created_at: number; // Unix timestamp in milliseconds
  updated_at: number; // Unix timestamp in milliseconds
}

// Event Payload Interfaces
export interface SendMessagePayload {
  room_id: string;
  content: string;
  type: 'text' | 'binary';
  client_msg_id: string;
  replied_message?: {
    msg_id: string;
  };
  metadata?: Record<string, any>;
}

export interface MessageReceivedPayload {
  message: Message;
}

export interface MessageDeliveredPayload {
  msg_id: string;
  room_id: string;
}

export interface MessageReadPayload {
  msg_id: string;
  room_id: string;
}

export interface MarkReadPayload {
  msg_id: string;
}

export interface ErrorPayload {
  code: string;
  message: string;
}

// API Response Interfaces
export interface GetUserResponse {
  id: string;
  username: string;
  full_name: string;
  email: string;
  avatar_url: string;
  website: string;
  updated_at: string; // ISO8601
}

export interface GetUsersResponse {
  users: UserListItem[];
}

export interface CreateRoomResponse {
  room_id: string;
  created_at: number; // Unix timestamp in milliseconds
}

export interface GetRoomResponse {
  room_id: string;
  participants: string[];
  type: '1-to-1' | 'group';
  last_messages: LastMessage;
  created_at: number; // Unix timestamp in milliseconds
  updated_at: number; // Unix timestamp in milliseconds
}

export interface GetRoomsResponse {
  rooms: Room[];
}

export interface GetMessagesResponse {
  messages: Message[];
}

export interface WSTicketResponse {
  ticket: string;
  expires_at: string; // ISO8601
}
