import type { WSEvent } from '../types';

/**
 * Generates a unique client message ID using timestamp and random component
 * @returns A unique client message ID string
 */
export function generateClientMsgId(): string {
  const timestamp = Date.now();
  const random = Math.random().toString(36).substring(2, 9);
  return `${timestamp}-${random}`;
}

/**
 * Creates a WebSocket event with the specified event name and payload,
 * automatically adding an ISO8601 timestamp
 * @param event The event name
 * @param payload The event payload
 * @returns A WSEvent object with event, payload, and ISO8601 timestamp
 */
export function createWSEvent<T>(event: string, payload: T): WSEvent<T> {
  return {
    event,
    payload,
    ts: new Date().toISOString()
  };
}

/**
 * Parses and validates a JSON string as a WSEvent structure
 * @param json The JSON string to parse
 * @returns The parsed WSEvent object
 * @throws Error if the JSON is invalid or doesn't match WSEvent structure
 */
export function parseWSEvent(json: string): WSEvent {
  try {
    const parsed = JSON.parse(json);
    
    // Validate WSEvent structure
    if (typeof parsed !== 'object' || parsed === null) {
      throw new Error('Invalid WSEvent: not an object');
    }
    
    if (typeof parsed.event !== 'string') {
      throw new Error('Invalid WSEvent: missing or invalid event field');
    }
    
    if (!('payload' in parsed)) {
      throw new Error('Invalid WSEvent: missing payload field');
    }
    
    if (typeof parsed.ts !== 'string') {
      throw new Error('Invalid WSEvent: missing or invalid ts field');
    }
    
    return parsed as WSEvent;
  } catch (error) {
    if (error instanceof SyntaxError) {
      throw new Error(`Invalid JSON: ${error.message}`);
    }
    throw error;
  }
}

/**
 * Calculates the reconnection delay using exponential backoff
 * Starting at 1 second, doubling each attempt, capped at 30 seconds
 * @param attemptNumber The current reconnection attempt number (0-indexed)
 * @returns The delay in milliseconds
 */
export function calculateReconnectDelay(attemptNumber: number): number {
  const baseDelay = 1000; // 1 second
  const maxDelay = 30000; // 30 seconds
  
  const delay = baseDelay * Math.pow(2, attemptNumber);
  return Math.min(delay, maxDelay);
}
