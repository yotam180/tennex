# Tennex Electron Client

Local-first WhatsApp messaging client built with Electron, TypeScript, and React.

## Architecture

- **Schema-first**: Types generated from backend OpenAPI spec
- **Local-first**: SQLite database with event sourcing
- **Cursor-based sync**: Efficient synchronization with backend
- **Offline-capable**: Queue messages when disconnected

## Tech Stack

- **Framework**: Electron + TypeScript + React 18
- **Build Tool**: Vite
- **Database**: better-sqlite3 + Drizzle ORM
- **API Client**: @tanstack/react-query
- **State Management**: Zustand
- **UI**: Radix UI + Tailwind CSS
- **Type Generation**: openapi-typescript

## Configuration

The app uses a centralized configuration system that supports environment variables:

### Environment Variables

Create a `.env.local` file (copy from `.env.example`):

```bash
# Backend Configuration
TENNEX_BACKEND_URL=http://localhost:8082
TENNEX_BACKEND_TIMEOUT=10000

# Sync Configuration  
TENNEX_SYNC_INTERVAL=5000
TENNEX_RETRY_DELAY=1000
TENNEX_MAX_RETRIES=3

# App Configuration
TENNEX_APP_NAME=Tennex
NODE_ENV=development
```

### Build-time Configuration

For different environments, you can set environment variables at build time:

```bash
# Development (default)
npm run dev

# Production build with custom backend
TENNEX_BACKEND_URL=https://api.tennex.com npm run build

# Or use predefined scripts
npm run build:prod     # Uses production defaults
npm run build:staging  # Uses staging defaults
```

## Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Run type generation
npm run codegen

# Package for distribution
npm run package
```

## Project Structure

```
src/
├── main/                    # Electron main process
│   ├── index.ts            # App initialization
│   ├── database/           # SQLite setup and migrations
│   ├── sync/               # Background sync service
│   └── ipc/                # IPC handlers
├── renderer/               # React app (renderer process)
│   ├── components/         # Reusable UI components
│   ├── pages/              # Main application pages
│   ├── hooks/              # React hooks (API, local state)
│   ├── stores/             # Zustand stores
│   ├── types/              # Generated + custom types
│   └── utils/              # Utilities
├── shared/                 # Code shared between processes
│   ├── config.ts           # Centralized configuration
│   ├── types/              # Common type definitions
│   ├── constants/          # App constants
│   └── schemas/            # Zod schemas
└── generated/              # Generated code (do not edit)
    ├── api-types.ts        # From OpenAPI spec
    └── database-schema.ts  # From SQL schema
```

## Deployment

The configuration system makes it easy to deploy to different environments:

```bash
# Local development
npm run dev

# Staging deployment
TENNEX_BACKEND_URL=https://staging-api.tennex.com npm run package

# Production deployment  
TENNEX_BACKEND_URL=https://api.tennex.com npm run package:prod
```

All backend URLs and configuration are centralized in `src/shared/config.ts`.