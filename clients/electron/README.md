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
```

## Project Structure

```
src/
├── main/                    # Electron main process
│   ├── index.ts
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
│   ├── types/              # Common type definitions
│   ├── constants/          # App constants
│   └── schemas/            # Zod schemas
└── generated/              # Generated code (do not edit)
    ├── api-types.ts        # From OpenAPI spec
    └── database-schema.ts  # From SQL schema
```
