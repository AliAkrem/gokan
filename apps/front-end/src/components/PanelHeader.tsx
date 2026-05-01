import React from 'react';
import { EditableUsername } from './EditableUsername';
import { ConnectionStatus as ConnectionStatusComponent } from './ConnectionStatus';
import { ConnectionStatus } from '../types';

interface PanelHeaderProps {
  username: string;
  onUsernameChange: (newUsername: string) => void;
  connectionStatus: ConnectionStatus;
}

export const PanelHeader: React.FC<PanelHeaderProps> = ({
  username,
  onUsernameChange,
  connectionStatus,
}) => {
  const storageKey = username.includes('left') ? 'chat_left_username' : 'chat_right_username';

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '1rem',
        borderBottom: '1px solid #3a3b3c',
        backgroundColor: '#1b1c1d',
        color: '#e4e6eb',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
        <EditableUsername
          username={username}
          onChange={onUsernameChange}
          storageKey={storageKey}
        />
      </div>
      <ConnectionStatusComponent status={connectionStatus} />
    </div>
  );
};
