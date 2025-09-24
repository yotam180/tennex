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
  
  // Run migrations (only if drizzle folder exists)
  try {
    migrate(db, { migrationsFolder: './drizzle' });
  } catch (error) {
    console.log('No migrations found, creating tables manually...');
    // For development, we'll create tables on the fly
    sqlite.exec(`
      CREATE TABLE IF NOT EXISTS events (
        seq INTEGER PRIMARY KEY AUTOINCREMENT,
        id TEXT NOT NULL UNIQUE,
        timestamp TEXT NOT NULL DEFAULT (datetime('now')),
        type TEXT NOT NULL,
        account_id TEXT NOT NULL,
        device_id TEXT,
        convo_id TEXT NOT NULL,
        wa_message_id TEXT,
        sender_jid TEXT,
        payload TEXT NOT NULL,
        attachment_ref TEXT,
        applied INTEGER NOT NULL DEFAULT 0,
        synced_seq INTEGER
      );
      
      CREATE TABLE IF NOT EXISTS outbox (
        client_msg_uuid TEXT PRIMARY KEY,
        account_id TEXT NOT NULL,
        convo_id TEXT NOT NULL,
        server_msg_id INTEGER,
        status TEXT NOT NULL DEFAULT 'queued',
        last_error TEXT,
        created_at TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at TEXT NOT NULL DEFAULT (datetime('now')),
        retry_count INTEGER NOT NULL DEFAULT 0,
        next_retry_at TEXT
      );
      
      CREATE TABLE IF NOT EXISTS accounts (
        id TEXT PRIMARY KEY,
        wa_jid TEXT UNIQUE,
        display_name TEXT,
        avatar_url TEXT,
        status TEXT NOT NULL DEFAULT 'disconnected',
        last_seen TEXT,
        created_at TEXT NOT NULL DEFAULT (datetime('now')),
        updated_at TEXT NOT NULL DEFAULT (datetime('now'))
      );
      
      CREATE TABLE IF NOT EXISTS sync_state (
        account_id TEXT PRIMARY KEY,
        last_sync_seq INTEGER NOT NULL DEFAULT 0,
        last_sync_at TEXT,
        is_online INTEGER NOT NULL DEFAULT 0
      );
      
      CREATE TABLE IF NOT EXISTS conversations (
        id TEXT PRIMARY KEY,
        account_id TEXT NOT NULL,
        display_name TEXT,
        last_message TEXT,
        last_message_at TEXT,
        unread_count INTEGER NOT NULL DEFAULT 0,
        is_pinned INTEGER NOT NULL DEFAULT 0,
        is_archived INTEGER NOT NULL DEFAULT 0,
        avatar_url TEXT,
        updated_at TEXT NOT NULL DEFAULT (datetime('now'))
      );
    `);
  }
  
  return db;
}

export function getDatabase() {
  if (!db) {
    throw new Error('Database not initialized. Call initializeDatabase() first.');
  }
  return db;
}

export { schema };
