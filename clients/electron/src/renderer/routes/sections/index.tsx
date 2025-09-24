import type { RouteObject } from 'react-router';

import { lazy, Suspense } from 'react';

import { SplashScreen } from 'src/components/loading-screen';

import { authRoutes } from './auth';
import { dashboardRoutes } from './dashboard';
// import { mainRoutes } from './main';
// import { authDemoRoutes } from './auth-demo';
// import { componentsRoutes } from './components';

// ----------------------------------------------------------------------

const Page404 = lazy(() => import('src/pages/error/404'));
const RootRedirect = lazy(() => import('src/pages/auth/root-redirect'));

// Temporary debug component - EXTREMELY OBVIOUS
function DebugRoot() {
  console.log('🔥🔥🔥 DEBUG: DebugRoot component is loading! This means the / route is working! 🔥🔥🔥');
  console.log('🔥 CONFIG:', {
    auth: { method: 'jwt', skip: false, redirectPath: '/dashboard' },
    localStorage: typeof window !== 'undefined' ? Object.keys(localStorage) : []
  });
  
  // Add window alert to be extra obvious
  setTimeout(() => {
    alert('🔥 DebugRoot component loaded! The / route is working!');
  }, 1000);
  
  return (
    <div style={{ 
      padding: '20px', 
      background: 'red', // EXTREMELY OBVIOUS RED BACKGROUND
      minHeight: '100vh',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      color: 'white',
      fontSize: '24px'
    }}>
      <h1 style={{ fontSize: '48px' }}>🔥🔥🔥 DEBUG MODE 🔥🔥🔥</h1>
      <p>The / route is working!</p>
      <p>Check the console for debug logs.</p>
      <p>If you see this, the routing is working but RootRedirect has an issue.</p>
      <button 
        onClick={() => window.location.href = '/auth/jwt/sign-in'}
        style={{ padding: '20px', fontSize: '20px', backgroundColor: 'yellow', color: 'black' }}
      >
        Go to Login
      </button>
    </div>
  );
}

export const routesSection: RouteObject[] = [
  {
    path: '/',
    // Temporary debug to see if route is working
    element: <DebugRoot />,
    // element: (
    //   <Suspense fallback={<SplashScreen />}>
    //     <RootRedirect />
    //   </Suspense>
    // ),
  },

  // Auth (JWT only)
  ...authRoutes,

  // Dashboard (protected by AuthGuard)
  ...dashboardRoutes,

  // Disabled routes - uncomment if needed for development
  // ...authDemoRoutes,
  // ...mainRoutes,
  // ...componentsRoutes,

  // No match
  { path: '*', element: <Page404 /> },
];
