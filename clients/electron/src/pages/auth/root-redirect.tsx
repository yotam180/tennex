import { useEffect } from 'react';

import { useRouter } from 'src/routes/hooks';

import { CONFIG } from 'src/global-config';

import { SplashScreen } from 'src/components/loading-screen';

import { useAuthContext } from 'src/auth/hooks';
import { JWT_STORAGE_KEY } from 'src/auth/context/jwt/constant';

// ----------------------------------------------------------------------

/**
 * Root redirect component that handles authentication-based routing
 * - If authenticated: redirect to dashboard
 * - If not authenticated: redirect to sign-in page
 * - While loading: show splash screen
 */
export default function RootRedirect() {
  const router = useRouter();
  const { authenticated, loading } = useAuthContext();

  console.log('🔍 RootRedirect Debug:', {
    authenticated,
    loading,
    authMethod: CONFIG.auth.method,
    redirectPath: CONFIG.auth.redirectPath,
    jwtToken: typeof window !== 'undefined' ? localStorage.getItem(JWT_STORAGE_KEY) : null,
    localStorageKeys: typeof window !== 'undefined' ? Object.keys(localStorage) : []
  });

  useEffect(() => {
    console.log('🔍 RootRedirect useEffect:', { authenticated, loading });
    
    if (loading) {
      console.log('🔄 Still loading auth state...');
      return; // Still checking authentication status
    }

    if (authenticated) {
      // User is logged in, redirect to dashboard
      console.log('✅ User authenticated, redirecting to dashboard:', CONFIG.auth.redirectPath);
      router.replace(CONFIG.auth.redirectPath);
    } else {
      // User is not logged in, redirect to sign-in
      const signInPath = `/auth/${CONFIG.auth.method}/sign-in`;
      console.log('❌ User not authenticated, redirecting to sign-in:', signInPath);
      router.replace(signInPath);
    }
  }, [authenticated, loading, router]);

  // Show loading screen while determining where to redirect
  console.log('🔄 Showing splash screen');
  return <SplashScreen />;
}
