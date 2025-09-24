import { useEffect } from 'react';

import { useRouter } from 'src/routes/hooks';

import { CONFIG } from 'src/global-config';

import { SplashScreen } from 'src/components/loading-screen';

import { useAuthContext } from 'src/auth/hooks';

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

  useEffect(() => {
    if (loading) {
      return; // Still checking authentication status
    }

    if (authenticated) {
      // User is logged in, redirect to dashboard
      router.replace(CONFIG.auth.redirectPath);
    } else {
      // User is not logged in, redirect to sign-in
      router.replace(`/auth/${CONFIG.auth.method}/sign-in`);
    }
  }, [authenticated, loading, router]);

  // Show loading screen while determining where to redirect
  return <SplashScreen />;
}
