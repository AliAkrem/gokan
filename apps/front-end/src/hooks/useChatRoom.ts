import { useState, useEffect } from "react";
import { createRoom, getRoom } from "../api/client";

const ROOM_ID_KEY = "chat_demo_room_id";
const POLL_INTERVAL = 500; // 500ms

interface UseChatRoomReturn {
  roomId: string | null;
  isLoading: boolean;
  error: Error | null;
}

type Props = {
  jwt: string | null;
  side: "left" | "right";
  leftUserId: string;
  rightUserId: string;
};

export function useChatRoom({
  jwt,
  side,
  leftUserId,
  rightUserId,
}: Props): UseChatRoomReturn {
  const [roomId, setRoomId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!jwt) {
      setIsLoading(true);
      return;
    }

    const handleLeftPanel = async () => {
      try {
        setIsLoading(true);
        setError(null);

        // Check localStorage for existing room_id
        const storedRoomId = localStorage.getItem(ROOM_ID_KEY);

        if (storedRoomId) {
          // Verify the room exists
          try {
            await getRoom(jwt, storedRoomId);
            setRoomId(storedRoomId);
          } catch (verifyError) {
            // Room doesn't exist, clear localStorage and create new room
            localStorage.removeItem(ROOM_ID_KEY);
            const response = await createRoom(jwt, [leftUserId, rightUserId]);
            localStorage.setItem(ROOM_ID_KEY, response.room_id);
            setRoomId(response.room_id);
          }
        } else {
          // No room_id exists, create new room
          const response = await createRoom(jwt, [leftUserId, rightUserId]);
          localStorage.setItem(ROOM_ID_KEY, response.room_id);
          setRoomId(response.room_id);
        }
      } catch (err) {
        setError(
          err instanceof Error ? err : new Error("Failed to bootstrap room"),
        );
      } finally {
        setIsLoading(false);
      }
    };

    const handleRightPanel = () => {
      setIsLoading(true);
      setError(null);

      // Poll localStorage until room_id is available
      const pollInterval = setInterval(() => {
        const storedRoomId = localStorage.getItem(ROOM_ID_KEY);
        if (storedRoomId) {
          setRoomId(storedRoomId);
          setIsLoading(false);
          clearInterval(pollInterval);
        }
      }, POLL_INTERVAL);

      // Cleanup interval on unmount
      return () => clearInterval(pollInterval);
    };

    if (side === "left") {
      handleLeftPanel();
    } else {
      return handleRightPanel();
    }
  }, [jwt, side]);

  return { roomId, isLoading, error };
}
