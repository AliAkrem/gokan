import { useState, useEffect } from 'react';
import { Transition } from '@mantine/core';
import { IconCheck } from '@tabler/icons-react';
import type { Message } from '../types';
import './MessageBubble.css';

interface MessageBubbleProps {
  message: Message;
  isOwnMessage: boolean;
}

export function MessageBubble({ message, isOwnMessage }: MessageBubbleProps) {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const formatTimestamp = (isoString: string): string => {
    const date = new Date(isoString);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const getStatusIcon = () => {
    if (!isOwnMessage) return null;

    switch (message.status) {
      case 'sent':
        return <IconCheck size={14} className="status-icon" />;
      case 'delivered':
        return (
          <span className="status-icon-group">
            <IconCheck size={14} className="status-icon" />
            <IconCheck size={14} className="status-icon status-icon-offset" />
          </span>
        );
      case 'read':
        return (
          <span className="status-icon-group status-read">
            <IconCheck size={14} className="status-icon" />
            <IconCheck size={14} className="status-icon status-icon-offset" />
          </span>
        );
      default:
        return null;
    }
  };

  return (
    <Transition mounted={mounted} transition="slide-up" duration={300} timingFunction="ease">
      {(styles) => (
        <div style={styles} className={`message-bubble ${isOwnMessage ? 'own' : 'other'}`}>
          <div className="message-content">
            <span className="message-text">{message.text}</span>
            <span className="message-footer">
              <span className="message-timestamp">{formatTimestamp(message.created_at)}</span>
              {getStatusIcon()}
            </span>
          </div>
        </div>
      )}
    </Transition>
  );
}
