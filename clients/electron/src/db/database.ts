/**
 * Database Service for Electron Main Process
 *
 * This service manages the SQLite database connection using better-sqlite3 and Drizzle ORM.
 * It should only be used in the main process, not the renderer.
 */

import { app } from 'electron';
import path from 'path';
import Database from 'better-sqlite3';
import { drizzle, BetterSQLite3Database } from 'drizzle-orm/better-sqlite3';
import { migrate } from 'drizzle-orm/better-sqlite3/migrator';
import * as schema from './schema';

export class DatabaseService {
  private db: BetterSQLite3Database<typeof schema> | null = null;
  private sqlite: Database.Database | null = null;
  private dbPath: string;

  constructor() {
    // Store database in Electron's userData directory
    const userDataPath = app.getPath('userData');
    this.dbPath = path.join(userDataPath, 'tennex.db');
    console.log(`📁 Database path: ${this.dbPath}`);
  }

  /**
   * Initialize the database connection and run migrations
   */
  async initialize(): Promise<void> {
    try {
      console.log('🔧 Initializing database...');

      // Create SQLite connection
      this.sqlite = new Database(this.dbPath);
      this.sqlite.pragma('journal_mode = WAL'); // Write-Ahead Logging for better concurrency
      this.sqlite.pragma('foreign_keys = ON'); // Enable foreign keys

      // Wrap with Drizzle ORM
      this.db = drizzle(this.sqlite, { schema });

      // Run migrations
      await this.runMigrations();

      console.log('✅ Database initialized successfully');
    } catch (error) {
      console.error('❌ Failed to initialize database:', error);
      throw error;
    }
  }

  /**
   * Run database migrations
   */
  private async runMigrations(): Promise<void> {
    if (!this.db) {
      throw new Error('Database not initialized');
    }

    try {
      console.log('🚀 Running migrations...');
      const migrationsFolder = path.join(__dirname, 'migrations');

      // Create tables manually (drizzle-kit generate might not work in production)
      this.createTables();

      console.log('✅ Migrations completed');
    } catch (error) {
      console.error('❌ Migration failed:', error);
      throw error;
    }
  }

  /**
   * Create tables manually (fallback if migrations folder doesn't exist)
   */
  private createTables(): void {
    if (!this.sqlite) {
      throw new Error('SQLite connection not established');
    }

    // Enable foreign keys
    this.sqlite.exec('PRAGMA foreign_keys = ON;');

    // Create sync_state table
    this.sqlite.exec(`
      CREATE TABLE IF NOT EXISTS sync_state (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        integration_id INTEGER NOT NULL UNIQUE,
        last_conv_seq INTEGER NOT NULL DEFAULT 0,
        last_message_seq INTEGER NOT NULL DEFAULT 0,
        last_contact_seq INTEGER NOT NULL DEFAULT 0,
        last_sync_at INTEGER NOT NULL DEFAULT (unixepoch()),
        created_at INTEGER NOT NULL DEFAULT (unixepoch()),
        updated_at INTEGER NOT NULL DEFAULT (unixepoch())
      );
    `);

    // Create conversations table
    this.sqlite.exec(`
      CREATE TABLE IF NOT EXISTS conversations (
        id TEXT PRIMARY KEY,
        integration_id INTEGER NOT NULL,
        external_conversation_id TEXT NOT NULL,
        integration_type TEXT NOT NULL,
        conversation_type TEXT NOT NULL,
        name TEXT,
        description TEXT,
        avatar_url TEXT,
        is_archived INTEGER NOT NULL DEFAULT 0,
        is_pinned INTEGER NOT NULL DEFAULT 0,
        is_muted INTEGER NOT NULL DEFAULT 0,
        mute_until INTEGER,
        is_read_only INTEGER NOT NULL DEFAULT 0,
        is_locked INTEGER NOT NULL DEFAULT 0,
        unread_count INTEGER NOT NULL DEFAULT 0,
        unread_mention_count INTEGER NOT NULL DEFAULT 0,
        total_message_count INTEGER NOT NULL DEFAULT 0,
        last_message_at INTEGER,
        last_activity_at INTEGER,
        platform_metadata TEXT,
        seq INTEGER NOT NULL,
        created_at INTEGER NOT NULL DEFAULT (unixepoch()),
        updated_at INTEGER NOT NULL DEFAULT (unixepoch())
      );
    `);

    // Create messages table
    this.sqlite.exec(`
      CREATE TABLE IF NOT EXISTS messages (
        id TEXT PRIMARY KEY,
        conversation_id TEXT NOT NULL,
        integration_id INTEGER NOT NULL,
        external_message_id TEXT NOT NULL,
        external_server_id TEXT,
        integration_type TEXT NOT NULL,
        sender_external_id TEXT NOT NULL,
        sender_display_name TEXT,
        message_type TEXT NOT NULL,
        content TEXT,
        timestamp INTEGER NOT NULL,
        edit_timestamp INTEGER,
        is_from_me INTEGER NOT NULL DEFAULT 0,
        is_forwarded INTEGER NOT NULL DEFAULT 0,
        is_deleted INTEGER NOT NULL DEFAULT 0,
        deleted_at INTEGER,
        reply_to_message_id TEXT,
        reply_to_external_id TEXT,
        delivery_status TEXT NOT NULL DEFAULT 'pending',
        platform_metadata TEXT,
        seq INTEGER NOT NULL,
        created_at INTEGER NOT NULL DEFAULT (unixepoch()),
        updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
        FOREIGN KEY (conversation_id) REFERENCES conversations(id)
      );
    `);

    // Create contacts table
    this.sqlite.exec(`
      CREATE TABLE IF NOT EXISTS contacts (
        id TEXT PRIMARY KEY,
        integration_id INTEGER NOT NULL,
        external_contact_id TEXT NOT NULL,
        integration_type TEXT NOT NULL,
        display_name TEXT,
        first_name TEXT,
        last_name TEXT,
        phone_number TEXT,
        email TEXT,
        username TEXT,
        avatar_url TEXT,
        about TEXT,
        is_favorite INTEGER NOT NULL DEFAULT 0,
        is_blocked INTEGER NOT NULL DEFAULT 0,
        is_verified INTEGER NOT NULL DEFAULT 0,
        is_business INTEGER NOT NULL DEFAULT 0,
        status TEXT,
        last_seen INTEGER,
        platform_metadata TEXT,
        seq INTEGER NOT NULL,
        created_at INTEGER NOT NULL DEFAULT (unixepoch()),
        updated_at INTEGER NOT NULL DEFAULT (unixepoch())
      );
    `);

    // Create message_media table
    this.sqlite.exec(`
      CREATE TABLE IF NOT EXISTS message_media (
        id TEXT PRIMARY KEY,
        message_id TEXT NOT NULL,
        integration_id INTEGER NOT NULL,
        media_type TEXT NOT NULL,
        mime_type TEXT,
        file_name TEXT,
        file_size INTEGER,
        url TEXT,
        thumbnail_url TEXT,
        local_path TEXT,
        width INTEGER,
        height INTEGER,
        duration REAL,
        download_status TEXT NOT NULL DEFAULT 'pending',
        download_progress INTEGER NOT NULL DEFAULT 0,
        platform_metadata TEXT,
        seq INTEGER NOT NULL,
        created_at INTEGER NOT NULL DEFAULT (unixepoch()),
        updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
        FOREIGN KEY (message_id) REFERENCES messages(id)
      );
    `);

    // Create conversation_participants table
    this.sqlite.exec(`
      CREATE TABLE IF NOT EXISTS conversation_participants (
        id TEXT PRIMARY KEY,
        conversation_id TEXT NOT NULL,
        contact_id TEXT NOT NULL,
        integration_id INTEGER NOT NULL,
        external_participant_id TEXT NOT NULL,
        role TEXT NOT NULL DEFAULT 'member',
        can_send INTEGER NOT NULL DEFAULT 1,
        can_edit INTEGER NOT NULL DEFAULT 0,
        can_delete INTEGER NOT NULL DEFAULT 0,
        can_add_members INTEGER NOT NULL DEFAULT 0,
        joined_at INTEGER,
        left_at INTEGER,
        platform_metadata TEXT,
        seq INTEGER NOT NULL,
        created_at INTEGER NOT NULL DEFAULT (unixepoch()),
        updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
        FOREIGN KEY (conversation_id) REFERENCES conversations(id),
        FOREIGN KEY (contact_id) REFERENCES contacts(id)
      );
    `);

    // Create indexes
    this.sqlite.exec(`
      CREATE INDEX IF NOT EXISTS idx_conversations_integration_id ON conversations(integration_id);
      CREATE INDEX IF NOT EXISTS idx_conversations_seq ON conversations(seq);
      CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
      CREATE INDEX IF NOT EXISTS idx_messages_integration_id ON messages(integration_id);
      CREATE INDEX IF NOT EXISTS idx_messages_seq ON messages(seq);
      CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
      CREATE INDEX IF NOT EXISTS idx_contacts_integration_id ON contacts(integration_id);
      CREATE INDEX IF NOT EXISTS idx_contacts_seq ON contacts(seq);
      CREATE INDEX IF NOT EXISTS idx_message_media_message_id ON message_media(message_id);
      CREATE INDEX IF NOT EXISTS idx_conversation_participants_conversation_id ON conversation_participants(conversation_id);
      CREATE INDEX IF NOT EXISTS idx_conversation_participants_contact_id ON conversation_participants(contact_id);
    `);

    console.log('✅ Tables created successfully');
  }

  /**
   * Get the Drizzle database instance
   */
  getDb(): BetterSQLite3Database<typeof schema> {
    if (!this.db) {
      throw new Error('Database not initialized. Call initialize() first.');
    }
    return this.db;
  }

  /**
   * Get the raw SQLite instance (for advanced queries)
   */
  getSqlite(): Database.Database {
    if (!this.sqlite) {
      throw new Error('Database not initialized. Call initialize() first.');
    }
    return this.sqlite;
  }

  /**
   * Close the database connection
   */
  close(): void {
    if (this.sqlite) {
      this.sqlite.close();
      this.db = null;
      this.sqlite = null;
      console.log('📪 Database connection closed');
    }
  }

  /**
   * Get database statistics
   */
  getStats(): {
    path: string;
    size: number;
    conversations: number;
    messages: number;
    contacts: number;
  } {
    if (!this.sqlite) {
      throw new Error('Database not initialized');
    }

    const fs = require('fs');
    const stats = fs.statSync(this.dbPath);

    const convCount = this.sqlite.prepare('SELECT COUNT(*) as count FROM conversations').get() as {
      count: number;
    };
    const msgCount = this.sqlite.prepare('SELECT COUNT(*) as count FROM messages').get() as {
      count: number;
    };
    const contactCount = this.sqlite.prepare('SELECT COUNT(*) as count FROM contacts').get() as {
      count: number;
    };

    return {
      path: this.dbPath,
      size: stats.size,
      conversations: convCount.count,
      messages: msgCount.count,
      contacts: contactCount.count,
    };
  }
}

// Export singleton instance
let dbServiceInstance: DatabaseService | null = null;

export function getDatabaseService(): DatabaseService {
  if (!dbServiceInstance) {
    dbServiceInstance = new DatabaseService();
  }
  return dbServiceInstance;
}
