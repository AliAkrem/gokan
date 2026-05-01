import React, { useState, useEffect } from 'react';
import { TextInput, Text } from '@mantine/core';

interface EditableUsernameProps {
  username: string;
  onChange: (newUsername: string) => void;
  storageKey: string;
}

export const EditableUsername: React.FC<EditableUsernameProps> = ({
  username,
  onChange,
  storageKey,
}) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(username);

  useEffect(() => {
    setEditValue(username);
  }, [username]);

  const handleSave = () => {
    const trimmedValue = editValue.trim();
    if (trimmedValue && trimmedValue !== username) {
      localStorage.setItem(storageKey, trimmedValue);
      onChange(trimmedValue);
    } else {
      setEditValue(username);
    }
    setIsEditing(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleSave();
    } else if (e.key === 'Escape') {
      setEditValue(username);
      setIsEditing(false);
    }
  };

  const handleBlur = () => {
    handleSave();
  };

  if (isEditing) {
    return (
      <TextInput
        value={editValue}
        onChange={(e) => setEditValue(e.currentTarget.value)}
        onKeyDown={handleKeyDown}
        onBlur={handleBlur}
        size="sm"
        style={{ maxWidth: '200px' }}
        autoFocus
      />
    );
  }

  return (
    <Text
      component="span"
      onClick={() => setIsEditing(true)}
      style={{
        cursor: 'pointer',
        padding: '0.5rem',
        borderRadius: '4px',
        transition: 'background-color 0.2s',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = 'rgba(255, 255, 255, 0.1)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = 'transparent';
      }}
    >
      {username}
    </Text>
  );
};
