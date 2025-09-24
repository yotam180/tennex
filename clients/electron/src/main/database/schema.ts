import { sql } from 'drizzle-orm';
import { sqliteTable, text, integer, blob } from 'drizzle-orm/sqlite-core';

// Mirror of backend schema for local-first operation

export const events = sqliteTable('events', {
  seq: integer('seq').primaryKey({ autoIncrement: true }),
  id: text('id').notNull().unique(),
  timestamp: text('timestamp').notNull().default(sql`datetime('now')`),
  type: text('type', { 
    enum: ['msg_in', 'msg_out_pending', 'msg_out_sent', 'msg_delivery', 'presence', 'contact_update', 'history_sync'] 
  }).notNull(),
  accountId: text('account_id').notNull(),
  deviceId: text('device_id'),
  convoId: text('convo_id').notNull(),
  waMessageId: text('wa_message_id'),
  senderJid: text('sender_jid'),
  payload: text('payload', { mode: 'json' }).notNull(),
  attachmentRef: text('attachment_ref', { mode: 'json' }),
  // Local-specific fields
  applied: integer('applied', { mode: 'boolean' }).notNull().default(false),
  syncedSeq: integer('synced_seq'), // Server sequence number when synced
});

export const outbox = sqliteTable('outbox', {
  clientMsgUuid: text('client_msg_uuid').primaryKey(),
  accountId: text('account_id').notNull(),
  convoId: text('convo_id').notNull(),
  serverMsgId: integer('server_msg_id'),
  status: text('status', { 
    enum: ['queued', 'sending', 'sent', 'failed', 'retry'] 
  }).notNull().default('queued'),
  lastError: text('last_error'),
  createdAt: text('created_at').notNull().default(sql`datetime('now')`),
  updatedAt: text('updated_at').notNull().default(sql`datetime('now')`),
  // Local fields
  retryCount: integer('retry_count').notNull().default(0),
  nextRetryAt: text('next_retry_at'),
});

export const accounts = sqliteTable('accounts', {
  id: text('id').primaryKey(),
  waJid: text('wa_jid').unique(),
  displayName: text('display_name'),
  avatarUrl: text('avatar_url'),
  status: text('status', { 
    enum: ['connected', 'disconnected', 'connecting', 'error'] 
  }).notNull().default('disconnected'),
  lastSeen: text('last_seen'),
  createdAt: text('created_at').notNull().default(sql`datetime('now')`),
  updatedAt: text('updated_at').notNull().default(sql`datetime('now')`),
});

export const mediaBlobs = sqliteTable('media_blobs', {
  contentHash: text('content_hash').primaryKey(),
  mimeType: text('mime_type').notNull(),
  sizeBytes: integer('size_bytes').notNull(),
  storageUrl: text('storage_url'),
  localPath: text('local_path'), // Local file system path
  downloadStatus: text('download_status', { 
    enum: ['pending', 'downloading', 'completed', 'failed'] 
  }).notNull().default('pending'),
  createdAt: text('created_at').notNull().default(sql`datetime('now')`),
});

// Local-only tables for client state
export const syncState = sqliteTable('sync_state', {
  accountId: text('account_id').primaryKey(),
  lastSyncSeq: integer('last_sync_seq').notNull().default(0),
  lastSyncAt: text('last_sync_at'),
  isOnline: integer('is_online', { mode: 'boolean' }).notNull().default(false),
});

export const conversations = sqliteTable('conversations', {
  id: text('id').primaryKey(),
  accountId: text('account_id').notNull(),
  displayName: text('display_name'),
  lastMessage: text('last_message'),
  lastMessageAt: text('last_message_at'),
  unreadCount: integer('unread_count').notNull().default(0),
  isPinned: integer('is_pinned', { mode: 'boolean' }).notNull().default(false),
  isArchived: integer('is_archived', { mode: 'boolean' }).notNull().default(false),
  avatarUrl: text('avatar_url'),
  updatedAt: text('updated_at').notNull().default(sql`datetime('now')`),
});

// Type exports for use in application
export type Event = typeof events.$inferSelect;
export type NewEvent = typeof events.$inferInsert;
export type OutboxEntry = typeof outbox.$inferSelect;
export type NewOutboxEntry = typeof outbox.$inferInsert;
export type Account = typeof accounts.$inferSelect;
export type NewAccount = typeof accounts.$inferInsert;
export type MediaBlob = typeof mediaBlobs.$inferSelect;
export type NewMediaBlob = typeof mediaBlobs.$inferInsert;
export type SyncState = typeof syncState.$inferSelect;
export type NewSyncState = typeof syncState.$inferInsert;
export type Conversation = typeof conversations.$inferSelect;
export type NewConversation = typeof conversations.$inferInsert;
