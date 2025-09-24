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

export const routesSection: RouteObject[] = [
  {
    path: '/',
    // Smart redirect based on authentication status
    element: (
      <Suspense fallback={<SplashScreen />}>
        <RootRedirect />
      </Suspense>
    ),
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
