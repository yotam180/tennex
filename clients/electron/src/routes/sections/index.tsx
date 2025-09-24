import type { RouteObject } from 'react-router';

import { lazy, Suspense } from 'react';

import { MainLayout } from 'src/layouts/main';

import { SplashScreen } from 'src/components/loading-screen';

import { authRoutes } from './auth';
import { mainRoutes } from './main';
import { authDemoRoutes } from './auth-demo';
import { dashboardRoutes } from './dashboard';
import { componentsRoutes } from './components';

// ----------------------------------------------------------------------

const HomePage = lazy(() => import('src/pages/home'));
const Page404 = lazy(() => import('src/pages/error/404'));
const RootRedirect = lazy(() => import('src/pages/auth/root-redirect'));

export const routesSection: RouteObject[] = [
  {
    path: '/',
    // 🔥 TENNEX: Authentication-based redirect
    // If logged in → redirect to dashboard
    // If not logged in → redirect to login page
    element: (
      <Suspense fallback={<SplashScreen />}>
        <RootRedirect />
      </Suspense>
    ),
  },

  // Auth
  ...authRoutes,
  ...authDemoRoutes,

  // Dashboard
  ...dashboardRoutes,

  // Main
  ...mainRoutes,

  // Components
  ...componentsRoutes,

  // No match
  { path: '*', element: <Page404 /> },
];
