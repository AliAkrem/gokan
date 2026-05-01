/**
 * Display name generation and management utilities
 * Implements Requirements 3.1, 3.2, 3.3, 3.4
 */

/**
 * Generates a random display name with format "User_XXXX"
 * where XXXX is 4 random alphanumeric characters
 * 
 * @returns Display name string (e.g., "User_A3k9")
 */
export function generateDisplayName(): string {
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let randomChars = '';
  
  for (let i = 0; i < 4; i++) {
    const randomIndex = Math.floor(Math.random() * characters.length);
    randomChars += characters[randomIndex];
  }
  
  return `User_${randomChars}`;
}

/**
 * Gets the display name for a specific panel side
 * Reads from localStorage if exists, otherwise generates and stores a new one
 * 
 * @param side - Panel side ('left' or 'right')
 * @returns Display name string
 */
export function getDisplayName(side: 'left' | 'right'): string {
  const storageKey = `chat_${side}_username`;
  
  // Try to read existing display name from localStorage
  const existingName = localStorage.getItem(storageKey);
  
  if (existingName) {
    return existingName;
  }
  
  // Generate new display name if none exists
  const newName = generateDisplayName();
  localStorage.setItem(storageKey, newName);
  
  return newName;
}
