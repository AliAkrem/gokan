import type {
  GetUserResponse,
  GetUsersResponse,
  CreateRoomResponse,
  GetRoomResponse,
  GetRoomsResponse,
  GetMessagesResponse,
  WSTicketResponse,
} from '../types';

const API_BASE_URL = import.meta.env.VITE_CHAT_API_URL || 'http://localhost:8080';


export function getAuthHeaders(jwt: string): HeadersInit {
  return {
    'Authorization': `Bearer ${jwt}`,
    'Content-Type': 'application/json',
  };
}


export async function getUser(jwt: string, userId : string ): Promise<GetUserResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/users/${userId}`, {
    method: 'GET',
    headers: getAuthHeaders(jwt),
  });
  if (!response.ok) throw new Error('Failed to get user');
  return response.json();
}


export async function getUsers(jwt: string): Promise<GetUsersResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/users`, {
    method: 'GET',
    headers: getAuthHeaders(jwt),
  });
  if (!response.ok) throw new Error('Failed to get users');
  return response.json();
}


export async function createRoom(jwt: string, participants: string[]): Promise<CreateRoomResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/rooms`, {
    method: 'POST',
    headers: getAuthHeaders(jwt),
    body: JSON.stringify({ participants }),
  });
  if (!response.ok) throw new Error('Failed to create room');
  return response.json();
}

export async function getRoom(jwt: string, roomId: string): Promise<GetRoomResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/rooms/${roomId}`, {
    method: 'GET',
    headers: getAuthHeaders(jwt),
  });
  if (!response.ok) throw new Error('Failed to get room');
  return response.json();
}

export async function getRooms(jwt: string): Promise<GetRoomsResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/rooms`, {
    method: 'GET',
    headers: getAuthHeaders(jwt),
  });
  if (!response.ok) throw new Error('Failed to get rooms');
  return response.json();
}

export async function getMessages(jwt: string, roomId: string): Promise<GetMessagesResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/rooms/${roomId}/messages`, {
    method: 'GET',
    headers: getAuthHeaders(jwt),
  });
  if (!response.ok) throw new Error('Failed to get messages');
  return response.json();
}

export async function deleteMessage(jwt: string, msgId: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/api/v1/messages/${msgId}`, {
    method: 'DELETE',
    headers: getAuthHeaders(jwt),
  });
  if (!response.ok) throw new Error('Failed to delete message');
}

export async function getWSTicket(jwt: string): Promise<WSTicketResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/auth/ws-ticket`, {
    method: 'POST',
    headers: getAuthHeaders(jwt),
  });
  if (!response.ok) throw new Error('Failed to get WS ticket');
  return response.json();
}
