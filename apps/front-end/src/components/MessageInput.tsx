import React, { useState } from 'react';
import { TextInput, Button, Group } from '@mantine/core';

interface MessageInputProps {
  onSendMessage: (content: string) => void;
  disabled: boolean;
}

export const MessageInput: React.FC<MessageInputProps> = ({ onSendMessage, disabled }) => {
  const [inputValue, setInputValue] = useState('');

  const handleSend = () => {
    if (inputValue.trim() && !disabled) {
      onSendMessage(inputValue.trim());
      setInputValue('');
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <Group gap="xs" style={{ padding: '1rem' }}>
      <TextInput
        style={{ flex: 1 }}
        placeholder={disabled ? 'Connecting...' : 'Type a message...'}
        value={inputValue}
        onChange={(e) => setInputValue(e.currentTarget.value)}
        onKeyDown={handleKeyPress}
        disabled={disabled}
      />
      <Button
        onClick={handleSend}
        disabled={disabled || !inputValue.trim()}
      >
        Send
      </Button>
    </Group>
  );
};
