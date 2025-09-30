/**
 * Database Operations (Main Process Only)
 *
 * These functions wrap Drizzle queries and are exposed via IPC to the renderer process.
 */

import { eq, and, gte, desc, asc } from 'drizzle-orm';
import { getDatabaseService } from './database';
import * as schema from './schema';
import type {
  SyncState,
  InsertSyncState,
  Conversation,
  InsertConversation,
  Message,
  InsertMessage,
  Contact,
  InsertContact,
  MessageMedia,
  InsertMessageMedia,
  ConversationParticipant,
  InsertConversationParticipant,
} from './schema';

// =============================================================================
// SYNC STATE OPERATIONS
// =============================================================================

export async function getSyncState(integrationId: number): Promise<SyncState | null> {
  const db = getDatabaseService().getDb();
  const result = await db
    .select()
    .from(schema.syncState)
    .where(eq(schema.syncState.integrationId, integrationId))
    .limit(1);

  return result[0] || null;
}

export async function upsertSyncState(
  data: Partial<InsertSyncState> & { integrationId: number }
): Promise<SyncState> {
  const db = getDatabaseService().getDb();
  const sqlite = getDatabaseService().getSqlite();

  // Use INSERT OR REPLACE for upsert
  const stmt = sqlite.prepare(`
    INSERT INTO sync_state (integration_id, last_conv_seq, last_message_seq, last_contact_seq, last_sync_at, updated_at)
    VALUES (?, ?, ?, ?, unixepoch(), unixepoch())
    ON CONFLICT(integration_id) DO UPDATE SET
      last_conv_seq = COALESCE(excluded.last_conv_seq, last_conv_seq),
      last_message_seq = COALESCE(excluded.last_message_seq, last_message_seq),
      last_contact_seq = COALESCE(excluded.last_contact_seq, last_contact_seq),
      last_sync_at = unixepoch(),
      updated_at = unixepoch()
  `);

  stmt.run(
    data.integrationId,
    data.lastConvSeq ?? 0,
    data.lastMessageSeq ?? 0,
    data.lastContactSeq ?? 0
  );

  return (await getSyncState(data.integrationId))!;
}

// =============================================================================
// CONVERSATION OPERATIONS
// =============================================================================

export async function upsertConversations(conversations: InsertConversation[]): Promise<number> {
  const sqlite = getDatabaseService().getSqlite();

  const stmt = sqlite.prepare(`
    INSERT INTO conversations (
      id, integration_id, external_conversation_id, integration_type, conversation_type,
      name, description, avatar_url, is_archived, is_pinned, is_muted, mute_until,
      is_read_only, is_locked, unread_count, unread_mention_count, total_message_count,
      last_message_at, last_activity_at, platform_metadata, seq, created_at, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(id) DO UPDATE SET
      name = excluded.name,
      description = excluded.description,
      avatar_url = excluded.avatar_url,
      is_archived = excluded.is_archived,
      is_pinned = excluded.is_pinned,
      is_muted = excluded.is_muted,
      mute_until = excluded.mute_until,
      is_read_only = excluded.is_read_only,
      is_locked = excluded.is_locked,
      unread_count = excluded.unread_count,
      unread_mention_count = excluded.unread_mention_count,
      total_message_count = excluded.total_message_count,
      last_message_at = excluded.last_message_at,
      last_activity_at = excluded.last_activity_at,
      platform_metadata = excluded.platform_metadata,
      seq = excluded.seq,
      updated_at = unixepoch()
  `);

  const insertMany = sqlite.transaction((convs: InsertConversation[]) => {
    for (const conv of convs) {
      stmt.run(
        conv.id,
        conv.integrationId,
        conv.externalConversationId,
        conv.integrationType,
        conv.conversationType,
        conv.name || null,
        conv.description || null,
        conv.avatarUrl || null,
        conv.isArchived ? 1 : 0,
        conv.isPinned ? 1 : 0,
        conv.isMuted ? 1 : 0,
        conv.muteUntil || null,
        conv.isReadOnly ? 1 : 0,
        conv.isLocked ? 1 : 0,
        conv.unreadCount || 0,
        conv.unreadMentionCount || 0,
        conv.totalMessageCount || 0,
        conv.lastMessageAt || null,
        conv.lastActivityAt || null,
        conv.platformMetadata ? JSON.stringify(conv.platformMetadata) : null,
        conv.seq,
        conv.createdAt || Math.floor(Date.now() / 1000),
        Math.floor(Date.now() / 1000)
      );
    }
  });

  insertMany(conversations);
  return conversations.length;
}

export async function getConversations(
  integrationId: number,
  limit = 100
): Promise<Conversation[]> {
  const db = getDatabaseService().getDb();
  return db
    .select()
    .from(schema.conversations)
    .where(eq(schema.conversations.integrationId, integrationId))
    .orderBy(desc(schema.conversations.lastMessageAt))
    .limit(limit);
}

// =============================================================================
// MESSAGE OPERATIONS
// =============================================================================

export async function upsertMessages(messages: InsertMessage[]): Promise<number> {
  const sqlite = getDatabaseService().getSqlite();

  const stmt = sqlite.prepare(`
    INSERT INTO messages (
      id, conversation_id, integration_id, external_message_id, external_server_id,
      integration_type, sender_external_id, sender_display_name, message_type, content,
      timestamp, edit_timestamp, is_from_me, is_forwarded, is_deleted, deleted_at,
      reply_to_message_id, reply_to_external_id, delivery_status, platform_metadata,
      seq, created_at, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(id) DO UPDATE SET
      content = excluded.content,
      edit_timestamp = excluded.edit_timestamp,
      is_deleted = excluded.is_deleted,
      deleted_at = excluded.deleted_at,
      delivery_status = excluded.delivery_status,
      platform_metadata = excluded.platform_metadata,
      seq = excluded.seq,
      updated_at = unixepoch()
  `);

  const insertMany = sqlite.transaction((msgs: InsertMessage[]) => {
    for (const msg of msgs) {
      stmt.run(
        msg.id,
        msg.conversationId,
        msg.integrationId,
        msg.externalMessageId,
        msg.externalServerId || null,
        msg.integrationType,
        msg.senderExternalId,
        msg.senderDisplayName || null,
        msg.messageType,
        msg.content || null,
        msg.timestamp,
        msg.editTimestamp || null,
        msg.isFromMe ? 1 : 0,
        msg.isForwarded ? 1 : 0,
        msg.isDeleted ? 1 : 0,
        msg.deletedAt || null,
        msg.replyToMessageId || null,
        msg.replyToExternalId || null,
        msg.deliveryStatus || 'pending',
        msg.platformMetadata ? JSON.stringify(msg.platformMetadata) : null,
        msg.seq,
        msg.createdAt || Math.floor(Date.now() / 1000),
        Math.floor(Date.now() / 1000)
      );
    }
  });

  insertMany(messages);
  return messages.length;
}

export async function getMessages(conversationId: string, limit = 100): Promise<Message[]> {
  const db = getDatabaseService().getDb();
  return db
    .select()
    .from(schema.messages)
    .where(eq(schema.messages.conversationId, conversationId))
    .orderBy(desc(schema.messages.timestamp))
    .limit(limit);
}

export async function getMessagesByIntegration(
  integrationId: number,
  limit = 1000
): Promise<Message[]> {
  const db = getDatabaseService().getDb();
  return db
    .select()
    .from(schema.messages)
    .where(eq(schema.messages.integrationId, integrationId))
    .orderBy(asc(schema.messages.timestamp))
    .limit(limit);
}

// =============================================================================
// CONTACT OPERATIONS
// =============================================================================

export async function upsertContacts(contacts: InsertContact[]): Promise<number> {
  const sqlite = getDatabaseService().getSqlite();

  const stmt = sqlite.prepare(`
    INSERT INTO contacts (
      id, integration_id, external_contact_id, integration_type, display_name,
      first_name, last_name, phone_number, email, username, avatar_url, about,
      is_favorite, is_blocked, is_verified, is_business, status, last_seen,
      platform_metadata, seq, created_at, updated_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(id) DO UPDATE SET
      display_name = excluded.display_name,
      first_name = excluded.first_name,
      last_name = excluded.last_name,
      phone_number = excluded.phone_number,
      email = excluded.email,
      username = excluded.username,
      avatar_url = excluded.avatar_url,
      about = excluded.about,
      is_favorite = excluded.is_favorite,
      is_blocked = excluded.is_blocked,
      is_verified = excluded.is_verified,
      is_business = excluded.is_business,
      status = excluded.status,
      last_seen = excluded.last_seen,
      platform_metadata = excluded.platform_metadata,
      seq = excluded.seq,
      updated_at = unixepoch()
  `);

  const insertMany = sqlite.transaction((ctcts: InsertContact[]) => {
    for (const contact of ctcts) {
      stmt.run(
        contact.id,
        contact.integrationId,
        contact.externalContactId,
        contact.integrationType,
        contact.displayName || null,
        contact.firstName || null,
        contact.lastName || null,
        contact.phoneNumber || null,
        contact.email || null,
        contact.username || null,
        contact.avatarUrl || null,
        contact.about || null,
        contact.isFavorite ? 1 : 0,
        contact.isBlocked ? 1 : 0,
        contact.isVerified ? 1 : 0,
        contact.isBusiness ? 1 : 0,
        contact.status || null,
        contact.lastSeen || null,
        contact.platformMetadata ? JSON.stringify(contact.platformMetadata) : null,
        contact.seq,
        contact.createdAt || Math.floor(Date.now() / 1000),
        Math.floor(Date.now() / 1000)
      );
    }
  });

  insertMany(contacts);
  return contacts.length;
}

export async function getContacts(integrationId: number, limit = 1000): Promise<Contact[]> {
  const db = getDatabaseService().getDb();
  return db
    .select()
    .from(schema.contacts)
    .where(eq(schema.contacts.integrationId, integrationId))
    .orderBy(asc(schema.contacts.displayName))
    .limit(limit);
}

// =============================================================================
// DATABASE STATS
// =============================================================================

export async function getDatabaseStats(): Promise<{
  path: string;
  size: number;
  conversations: number;
  messages: number;
  contacts: number;
}> {
  return getDatabaseService().getStats();
}
