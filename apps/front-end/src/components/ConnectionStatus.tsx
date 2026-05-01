import { Badge } from '@mantine/core';
import { ConnectionStatus as ConnectionStatusType } from '../types';

interface ConnectionStatusProps {
  status: ConnectionStatusType;
}

export function ConnectionStatus({ status }: ConnectionStatusProps) {
  const getStatusConfig = () => {
    switch (status) {
      case 'Connected':
        return {
          color: 'green',
          text: 'Connected'
        };
      case 'Reconnecting':
        return {
          color: 'yellow',
          text: 'Reconnecting...'
        };
      case 'Disconnected':
        return {
          color: 'red',
          text: 'Disconnected'
        };
      default:
        return {
          color: 'gray',
          text: 'Unknown'
        };
    }
  };

  const config = getStatusConfig();

  return (
    <Badge color={config.color} variant="dot" size="lg">
      {config.text}
    </Badge>
  );
}
