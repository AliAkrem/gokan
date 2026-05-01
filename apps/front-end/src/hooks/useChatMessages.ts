import { useState, useEffect, useCallback } from 'react';
import { getMessages } from '../api/client';
import type { Message } from '../types';

interface UseChatMessagesReturn {
  messages: Message[];
  setMessages: React.Dispatch<React.SetStateAction<Message[]>>;
  isLoading: boolean;
  error: Error | null;
  refreshMessages: () => Promise<void>;
}

export function useChatMessages(
  jwt: string | null,
  roomId: string | null
): UseChatMessagesReturn {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const fetchMessages = useCallback(async () => {
    if (!jwt || !roomId) {
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await getMessages(jwt, roomId);
      // API returns messages newest-first; reverse to chronological order
      setMessages(response.messages.reverse());
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to fetch messages');
      setError(error);
      console.error('Error fetching messages:', error);
    } finally {
      setIsLoading(false);
    }
  }, [jwt, roomId]);

  const refreshMessages = useCallback(async () => {
    await fetchMessages();
  }, [fetchMessages]);

  useEffect(() => {
    fetchMessages();
  }, [fetchMessages]);

  return {
    messages,
    setMessages,
    isLoading,
    error,
    refreshMessages,
  };
}
