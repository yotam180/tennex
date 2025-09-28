import { useSetState } from 'minimal-shared/hooks';
import { useMemo, useEffect, useCallback } from 'react';

import axios, { endpoints } from 'src/lib/axios';

import { JWT_STORAGE_KEY } from './constant';
import { AuthContext } from '../auth-context';
import { setSession, isValidToken, jwtDecode } from './utils';

import type { AuthState } from '../../types';

// ----------------------------------------------------------------------

/**
 * NOTE:
 * We only build demo at basic level.
 * Customer will need to do some extra handling yourself if you want to extend the logic and other features...
 */

type Props = {
  children: React.ReactNode;
};

export function AuthProvider({ children }: Props) {
  const { state, setState } = useSetState<AuthState>({ user: null, loading: true });

  const checkUserSession = useCallback(async () => {
    console.log('🔍 JWT Auth Provider - Checking user session...');
    
    try {
      const accessToken = localStorage.getItem(JWT_STORAGE_KEY);
      console.log('🔍 Found token in localStorage:', !!accessToken);

      if (accessToken && isValidToken(accessToken)) {
        console.log('✅ Token is valid, setting session...');
        setSession(accessToken);

        try {
          console.log('🔄 Fetching user details from /auth/me...');
          const res = await axios.get(endpoints.auth.me);
          const user = res.data; // Backend returns user object directly, not wrapped
          console.log('✅ User details fetched successfully:', user);
          setState({ user: { ...user, accessToken, role: 'admin' }, loading: false });
        } catch (error) {
          // If /auth/me fails, still consider user logged in with token info
          console.warn('⚠️ Could not fetch user details, using token info:', error);
          const decodedToken = jwtDecode(accessToken);
          console.log('🔍 Decoded token:', decodedToken);
          setState({ 
            user: { 
              id: decodedToken.user_id || decodedToken.sub, // Our backend uses user_id
              username: decodedToken.username || 'user',
              email: decodedToken.email || 'user@example.com',
              displayName: decodedToken.name || decodedToken.username || 'User',
              role: 'admin',
              accessToken 
            }, 
            loading: false 
          });
        }
      } else {
        console.log('❌ No valid token found, setting user to null');
        setState({ user: null, loading: false });
      }
    } catch (error) {
      console.error('🚨 Error in checkUserSession:', error);
      setState({ user: null, loading: false });
    }
  }, [setState]);

  useEffect(() => {
    checkUserSession();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ----------------------------------------------------------------------

  const checkAuthenticated = state.user ? 'authenticated' : 'unauthenticated';

  const status = state.loading ? 'loading' : checkAuthenticated;

  const memoizedValue = useMemo(() => {
    const authValue = {
      user: state.user ? { ...state.user, role: state.user?.role ?? 'admin' } : null,
      checkUserSession,
      loading: status === 'loading',
      authenticated: status === 'authenticated',
      unauthenticated: status === 'unauthenticated',
    };
    
    console.log('🔍 Auth Context Value:', authValue);
    return authValue;
  }, [checkUserSession, state.user, status]);

  return <AuthContext.Provider value={memoizedValue}>{children}</AuthContext.Provider>;
}
