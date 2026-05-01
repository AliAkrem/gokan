import { useState, useEffect, useCallback } from 'react';
import { createClient } from '@supabase/supabase-js';
import type { SupabaseClient, Session } from '@supabase/supabase-js';

interface UseSupabaseAuthReturn {
  jwt: string | null;
  userId: string | null;
  isLoading: boolean;
  error: Error | null;
  refreshToken: () => Promise<void>;
}

/**
 * Custom hook for managing Supabase anonymous authentication and JWT lifecycle
 * @param side - Panel identifier ('left' or 'right') for scoped storage
 * @returns Authentication state and JWT management functions
 */
export function useSupabaseAuth(side: 'left' | 'right'): UseSupabaseAuthReturn {
  const [jwt, setJwt] = useState<string | null>(null);
  const [userId, setUserId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<Error | null>(null);
  const [supabaseClient, setSupabaseClient] = useState<SupabaseClient | null>(null);

  // Initialize Supabase client with scoped storage key
  useEffect(() => {
    const supabaseUrl = import.meta.env.VITE_SUPABASE_URL;
    const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY;

    if (!supabaseUrl || !supabaseAnonKey) {
      setError(new Error('Supabase configuration missing'));
      setIsLoading(false);
      return;
    }

    const client = createClient(supabaseUrl, supabaseAnonKey, {
      auth: {
        storageKey: `supabase-${side}`,
        autoRefreshToken: true,
        persistSession: true,
      },
    });

    setSupabaseClient(client);
  }, [side]);

  // Extract user ID from JWT payload
  const extractUserIdFromJwt = useCallback((token: string): string | null => {
    try {
      const payload = token.split('.')[1];
      const decoded = JSON.parse(atob(payload));
      return decoded.sub || null;
    } catch (err) {
      console.error('Failed to decode JWT:', err);
      return null;
    }
  }, []);

  // Store JWT and extract user ID
  const storeJwt = useCallback((token: string) => {
    const storageKey = `chat_${side}_token`;
    localStorage.setItem(storageKey, token);
    setJwt(token);
    
    const extractedUserId = extractUserIdFromJwt(token);
    setUserId(extractedUserId);
  }, [side, extractUserIdFromJwt]);

  // Handle session and extract JWT
  const handleSession = useCallback((session: Session | null) => {
    if (session?.access_token) {
      storeJwt(session.access_token);
    } else {
      setJwt(null);
      setUserId(null);
    }
  }, [storeJwt]);

  // Refresh token function
  const refreshToken = useCallback(async () => {
    if (!supabaseClient) {
      setError(new Error('Supabase client not initialized'));
      return;
    }

    try {
      const { data, error: refreshError } = await supabaseClient.auth.refreshSession();
      
      if (refreshError) {
        throw refreshError;
      }

      if (data.session) {
        handleSession(data.session);
      }
    } catch (err) {
      const errorObj = err instanceof Error ? err : new Error('Failed to refresh token');
      setError(errorObj);
      console.error('Token refresh failed:', errorObj);
    }
  }, [supabaseClient, handleSession]);

  // Check for existing session or sign in anonymously
  useEffect(() => {
    if (!supabaseClient) return;

    const initAuth = async () => {
      try {
        setIsLoading(true);
        setError(null);

        // Check for stored JWT first
        const storageKey = `chat_${side}_token`;
        const storedJwt = localStorage.getItem(storageKey);

        // Get current session
        const { data: { session }, error: sessionError } = await supabaseClient.auth.getSession();

        if (sessionError) {
          throw sessionError;
        }

        // If we have a valid session, use it
        if (session) {
          handleSession(session);
        } 
        // If we have a stored JWT but no session, try to use it
        else if (storedJwt) {
          setJwt(storedJwt);
          const extractedUserId = extractUserIdFromJwt(storedJwt);
          setUserId(extractedUserId);
          
          // Try to refresh the session
          await refreshToken();
        } 
        // No session and no stored JWT - sign in anonymously
        else {
          const { data, error: signInError } = await supabaseClient.auth.signInAnonymously();
          
          if (signInError) {
            throw signInError;
          }

          if (data.session) {
            handleSession(data.session);
          }
        }
      } catch (err) {
        const errorObj = err instanceof Error ? err : new Error('Authentication failed');
        setError(errorObj);
        console.error('Authentication error:', errorObj);
      } finally {
        setIsLoading(false);
      }
    };

    initAuth();

    // Set up auth state change listener
    const { data: { subscription } } = supabaseClient.auth.onAuthStateChange((_event, session) => {
      handleSession(session);
    });

    return () => {
      subscription.unsubscribe();
    };
  }, [supabaseClient, side, handleSession, extractUserIdFromJwt, refreshToken]);

  return {
    jwt,
    userId,
    isLoading,
    error,
    refreshToken,
  };
}
