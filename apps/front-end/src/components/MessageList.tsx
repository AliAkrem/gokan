import { useEffect, useRef } from 'react';
import { Paper } from '@mantine/core';
import type { Message } from '../types';
import { MessageBubble } from './MessageBubble';
import './MessageList.css';

interface MessageListProps {
  messages: Message[];
  currentUserId: string;
  onDeleteMessage?: (msgId: string) => void;
}

export function MessageList({ messages, currentUserId }: MessageListProps) {
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new message arrives
  useEffect(() => {
    // Scroll immediately for fast feedback
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });

    // Scroll again after the 300ms Mantine transition completes
    // to ensure we reach the very bottom after the element expands
    const timeoutId = setTimeout(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, 310);

    return () => clearTimeout(timeoutId);
  }, [messages]);

  return (
    <Paper className="message-list-container" style={{ backgroundColor: 'var(--mantine-color-dark-7)' }}>
      {messages.length === 0 ? (
        <div className="empty-state">No messages yet. Say something!</div>
      ) : (
        <div className="messages-wrapper">
          {messages.map((message) => (
            <MessageBubble
              key={message.msg_id}
              message={message}
              isOwnMessage={message.author_id === currentUserId}
            />
          ))}
          <div ref={messagesEndRef} />
        </div>
      )}
    </Paper>
  );
}
