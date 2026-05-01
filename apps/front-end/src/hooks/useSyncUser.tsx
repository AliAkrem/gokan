import { useState, useEffect } from "react";
import { getUser } from "../api/client";

export function useSyncUser({ jwt, userId }: { jwt: string; userId: string }) {
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!jwt) {
      setIsLoading(true);
      return;
    }

    const getCurrentUser = async () => {
      try {
        setIsLoading(true);
        setError(null);
        try {
          await getUser(jwt, userId);
        } catch (verifyError) {}
      } catch (err) {
        setError(err instanceof Error ? err : new Error("Failed to sync User"));
      } finally {
        setIsLoading(false);
      }
    };

    getCurrentUser();
  }, [jwt]);

  return { isLoading, error };
}
