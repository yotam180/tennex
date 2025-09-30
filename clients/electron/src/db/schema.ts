/**
 * Drizzle ORM Schema for Local SQLite Database
 *
 * This mirrors the backend PostgreSQL schema but optimized for local-first architecture.
 * Uses auto-incrementing integers for primary keys to avoid UUID collisions.
 */

import { sql } from 'drizzle-orm';
import { sqliteTable, text, integer, real } from 'drizzle-orm/sqlite-core';

// =============================================================================
// SYNC STATE TABLE
// =============================================================================

export const syncState = sqliteTable('sync_state', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  integrationId: integer('integration_id').notNull().unique(),
  lastConvSeq: integer('last_conv_seq').notNull().default(0),
  lastMessageSeq: integer('last_message_seq').notNull().default(0),
  lastContactSeq: integer('last_contact_seq').notNull().default(0),
  lastSyncAt: integer('last_sync_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
  createdAt: integer('created_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
  updatedAt: integer('updated_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
});

// =============================================================================
// CONVERSATIONS TABLE
// =============================================================================

export const conversations = sqliteTable('conversations', {
  id: text('id').primaryKey(), // UUID from backend
  integrationId: integer('integration_id').notNull(),
  externalConversationId: text('external_conversation_id').notNull(),
  integrationType: text('integration_type').notNull(), // 'whatsapp', 'telegram', etc.
  conversationType: text('conversation_type').notNull(), // 'direct', 'group', 'broadcast'
  name: text('name'),
  description: text('description'),
  avatarUrl: text('avatar_url'),

  // Flags
  isArchived: integer('is_archived', { mode: 'boolean' }).notNull().default(false),
  isPinned: integer('is_pinned', { mode: 'boolean' }).notNull().default(false),
  isMuted: integer('is_muted', { mode: 'boolean' }).notNull().default(false),
  muteUntil: integer('mute_until', { mode: 'timestamp' }),
  isReadOnly: integer('is_read_only', { mode: 'boolean' }).notNull().default(false),
  isLocked: integer('is_locked', { mode: 'boolean' }).notNull().default(false),

  // Counters
  unreadCount: integer('unread_count').notNull().default(0),
  unreadMentionCount: integer('unread_mention_count').notNull().default(0),
  totalMessageCount: integer('total_message_count').notNull().default(0),

  // Timestamps
  lastMessageAt: integer('last_message_at', { mode: 'timestamp' }),
  lastActivityAt: integer('last_activity_at', { mode: 'timestamp' }),

  // Metadata
  platformMetadata: text('platform_metadata', { mode: 'json' }), // Store JSON blob

  // Sync
  seq: integer('seq').notNull(), // Sequence number from backend

  // Timestamps
  createdAt: integer('created_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
  updatedAt: integer('updated_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
});

// =============================================================================
// MESSAGES TABLE
// =============================================================================

export const messages = sqliteTable('messages', {
  id: text('id').primaryKey(), // UUID from backend
  conversationId: text('conversation_id')
    .notNull()
    .references(() => conversations.id),
  integrationId: integer('integration_id').notNull(),

  // External IDs
  externalMessageId: text('external_message_id').notNull(),
  externalServerId: text('external_server_id'),
  integrationType: text('integration_type').notNull(),

  // Sender info
  senderExternalId: text('sender_external_id').notNull(),
  senderDisplayName: text('sender_display_name'),

  // Message content
  messageType: text('message_type').notNull(), // 'text', 'image', 'video', 'audio', 'document', etc.
  content: text('content'),
  timestamp: integer('timestamp', { mode: 'timestamp' }).notNull(),
  editTimestamp: integer('edit_timestamp', { mode: 'timestamp' }),

  // Flags
  isFromMe: integer('is_from_me', { mode: 'boolean' }).notNull().default(false),
  isForwarded: integer('is_forwarded', { mode: 'boolean' }).notNull().default(false),
  isDeleted: integer('is_deleted', { mode: 'boolean' }).notNull().default(false),
  deletedAt: integer('deleted_at', { mode: 'timestamp' }),

  // Reply chain
  replyToMessageId: text('reply_to_message_id'),
  replyToExternalId: text('reply_to_external_id'),

  // Delivery
  deliveryStatus: text('delivery_status').notNull().default('pending'), // 'pending', 'sent', 'delivered', 'read', 'failed'

  // Metadata
  platformMetadata: text('platform_metadata', { mode: 'json' }),

  // Sync
  seq: integer('seq').notNull(),

  // Timestamps
  createdAt: integer('created_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
  updatedAt: integer('updated_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
});

// =============================================================================
// MESSAGE MEDIA TABLE
// =============================================================================

export const messageMedia = sqliteTable('message_media', {
  id: text('id').primaryKey(),
  messageId: text('message_id')
    .notNull()
    .references(() => messages.id),
  integrationId: integer('integration_id').notNull(),

  mediaType: text('media_type').notNull(), // 'image', 'video', 'audio', 'document', 'sticker', 'voice'
  mimeType: text('mime_type'),
  fileName: text('file_name'),
  fileSize: integer('file_size'),

  // URLs
  url: text('url'),
  thumbnailUrl: text('thumbnail_url'),
  localPath: text('local_path'), // Path to locally downloaded file

  // Media dimensions/duration
  width: integer('width'),
  height: integer('height'),
  duration: real('duration'), // For video/audio (seconds)

  // Download state
  downloadStatus: text('download_status').notNull().default('pending'), // 'pending', 'downloading', 'completed', 'failed'
  downloadProgress: integer('download_progress').notNull().default(0), // 0-100

  // Metadata
  platformMetadata: text('platform_metadata', { mode: 'json' }),

  // Sync
  seq: integer('seq').notNull(),

  // Timestamps
  createdAt: integer('created_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
  updatedAt: integer('updated_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
});

// =============================================================================
// CONTACTS TABLE
// =============================================================================

export const contacts = sqliteTable('contacts', {
  id: text('id').primaryKey(),
  integrationId: integer('integration_id').notNull(),
  externalContactId: text('external_contact_id').notNull(),
  integrationType: text('integration_type').notNull(),

  // Contact info
  displayName: text('display_name'),
  firstName: text('first_name'),
  lastName: text('last_name'),
  phoneNumber: text('phone_number'),
  email: text('email'),
  username: text('username'),

  // Profile
  avatarUrl: text('avatar_url'),
  about: text('about'),

  // Flags
  isFavorite: integer('is_favorite', { mode: 'boolean' }).notNull().default(false),
  isBlocked: integer('is_blocked', { mode: 'boolean' }).notNull().default(false),
  isVerified: integer('is_verified', { mode: 'boolean' }).notNull().default(false),
  isBusiness: integer('is_business', { mode: 'boolean' }).notNull().default(false),

  // Status
  status: text('status'),
  lastSeen: integer('last_seen', { mode: 'timestamp' }),

  // Metadata
  platformMetadata: text('platform_metadata', { mode: 'json' }),

  // Sync
  seq: integer('seq').notNull(),

  // Timestamps
  createdAt: integer('created_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
  updatedAt: integer('updated_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
});

// =============================================================================
// CONVERSATION PARTICIPANTS TABLE
// =============================================================================

export const conversationParticipants = sqliteTable('conversation_participants', {
  id: text('id').primaryKey(),
  conversationId: text('conversation_id')
    .notNull()
    .references(() => conversations.id),
  contactId: text('contact_id')
    .notNull()
    .references(() => contacts.id),
  integrationId: integer('integration_id').notNull(),

  // Participant info
  externalParticipantId: text('external_participant_id').notNull(),
  role: text('role').notNull().default('member'), // 'owner', 'admin', 'member'

  // Permissions
  canSend: integer('can_send', { mode: 'boolean' }).notNull().default(true),
  canEdit: integer('can_edit', { mode: 'boolean' }).notNull().default(false),
  canDelete: integer('can_delete', { mode: 'boolean' }).notNull().default(false),
  canAddMembers: integer('can_add_members', { mode: 'boolean' }).notNull().default(false),

  // Timestamps
  joinedAt: integer('joined_at', { mode: 'timestamp' }),
  leftAt: integer('left_at', { mode: 'timestamp' }),

  // Metadata
  platformMetadata: text('platform_metadata', { mode: 'json' }),

  // Sync
  seq: integer('seq').notNull(),

  // Timestamps
  createdAt: integer('created_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
  updatedAt: integer('updated_at', { mode: 'timestamp' })
    .notNull()
    .default(sql`(unixepoch())`),
});

// =============================================================================
// TYPE EXPORTS
// =============================================================================

export type SyncState = typeof syncState.$inferSelect;
export type InsertSyncState = typeof syncState.$inferInsert;

export type Conversation = typeof conversations.$inferSelect;
export type InsertConversation = typeof conversations.$inferInsert;

export type Message = typeof messages.$inferSelect;
export type InsertMessage = typeof messages.$inferInsert;

export type MessageMedia = typeof messageMedia.$inferSelect;
export type InsertMessageMedia = typeof messageMedia.$inferInsert;

export type Contact = typeof contacts.$inferSelect;
export type InsertContact = typeof contacts.$inferInsert;

export type ConversationParticipant = typeof conversationParticipants.$inferSelect;
export type InsertConversationParticipant = typeof conversationParticipants.$inferInsert;
