import Database from 'better-sqlite3';
import { drizzle } from 'drizzle-orm/better-sqlite3';
import { migrate } from 'drizzle-orm/better-sqlite3/migrator';
import { app } from 'electron';
import path from 'path';
import * as schema from './schema.js';

let db: ReturnType<typeof drizzle<typeof schema>>;

export function initializeDatabase() {
  const dbPath = path.join(app.getPath('userData'), 'tennex.db');
  const sqlite = new Database(dbPath);
  
  // Enable WAL mode for better concurrency
  sqlite.pragma('journal_mode = WAL');
  sqlite.pragma('synchronous = NORMAL');
  sqlite.pragma('cache_size = 1000000');
  sqlite.pragma('foreign_keys = ON');
  sqlite.pragma('temp_store = MEMORY');
  
  db = drizzle(sqlite, { schema });
  
  // Run migrations
  migrate(db, { migrationsFolder: './drizzle' });
  
  return db;
}

export function getDatabase() {
  if (!db) {
    throw new Error('Database not initialized. Call initializeDatabase() first.');
  }
  return db;
}

export { schema };
