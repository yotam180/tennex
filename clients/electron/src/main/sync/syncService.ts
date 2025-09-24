import { getDatabase, schema } from '../database/index.js';
import { eq, gt, and } from 'drizzle-orm';
import type { Event as BackendEvent, SyncResponse } from '../../generated/api-types.js';

interface SyncServiceConfig {
  backendUrl: string;
  authToken: string;
  syncIntervalMs: number;
}

export class SyncService {
  private config: SyncServiceConfig;
  private syncInterval: NodeJS.Timeout | null = null;
  private isOnline = false;
  private isSyncing = false;

  constructor(config: SyncServiceConfig) {
    this.config = config;
  }

  async start() {
    this.isOnline = true;
    await this.performInitialSync();
    this.startPeriodicSync();
  }

  stop() {
    this.isOnline = false;
    if (this.syncInterval) {
      clearInterval(this.syncInterval);
      this.syncInterval = null;
    }
  }

  private async performInitialSync() {
    const db = getDatabase();
    
    // Get all accounts and their last sync positions
    const accounts = await db.select().from(schema.accounts);
    
    for (const account of accounts) {
      await this.syncAccount(account.id);
    }
  }

  private async syncAccount(accountId: string) {
    if (this.isSyncing) return;
    
    this.isSyncing = true;
    const db = getDatabase();

    try {
      // Get last synced sequence number
      let syncState = await db.select()
        .from(schema.syncState)
        .where(eq(schema.syncState.accountId, accountId))
        .get();

      if (!syncState) {
        // Initialize sync state
        await db.insert(schema.syncState).values({
          accountId,
          lastSyncSeq: 0,
          isOnline: this.isOnline,
        });
        syncState = { accountId, lastSyncSeq: 0, lastSyncAt: null, isOnline: this.isOnline };
      }

      // Fetch events from backend
      const response = await this.fetchEventsFromBackend(accountId, syncState.lastSyncSeq);
      
      if (response.events.length > 0) {
        // Apply events to local database
        await this.applyEvents(response.events);
        
        // Update sync state
        await db.update(schema.syncState)
          .set({
            lastSyncSeq: response.next_seq,
            lastSyncAt: new Date().toISOString(),
            isOnline: true,
          })
          .where(eq(schema.syncState.accountId, accountId));
      }

      // Process outbox - send pending messages
      await this.processOutbox(accountId);

    } catch (error) {
      console.error(`Sync failed for account ${accountId}:`, error);
      
      // Update sync state to reflect offline status
      await db.update(schema.syncState)
        .set({ isOnline: false })
        .where(eq(schema.syncState.accountId, accountId));
    } finally {
      this.isSyncing = false;
    }
  }

  private async fetchEventsFromBackend(accountId: string, since: number): Promise<SyncResponse> {
    const url = new URL('/sync', this.config.backendUrl);
    url.searchParams.set('account_id', accountId);
    url.searchParams.set('since', since.toString());
    url.searchParams.set('limit', '1000');

    const response = await fetch(url.toString(), {
      headers: {
        'Authorization': `Bearer ${this.config.authToken}`,
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`Sync API error: ${response.status} ${response.statusText}`);
    }

    return await response.json();
  }

  private async applyEvents(events: BackendEvent[]) {
    const db = getDatabase();

    for (const event of events) {
      // Transform backend event to local event format
      const localEvent = {
        id: event.id,
        timestamp: event.timestamp,
        type: event.type,
        accountId: event.account_id,
        deviceId: event.device_id || null,
        convoId: event.convo_id,
        waMessageId: event.wa_message_id || null,
        senderJid: event.sender_jid || null,
        payload: event.payload,
        attachmentRef: event.attachment_ref || null,
        applied: true,
        syncedSeq: event.seq,
      };

      // Insert event (ignore duplicates)
      await db.insert(schema.events)
        .values(localEvent)
        .onConflictDoNothing();

      // Update conversation projections based on event type
      await this.updateConversationProjection(event);
    }
  }

  private async updateConversationProjection(event: BackendEvent) {
    const db = getDatabase();

    if (event.type === 'msg_in' || event.type === 'msg_out_sent') {
      const payload = event.payload as any;
      
      // Update or create conversation
      await db.insert(schema.conversations)
        .values({
          id: event.convo_id,
          accountId: event.account_id,
          displayName: payload.chat_name || event.convo_id,
          lastMessage: payload.body || '[Media]',
          lastMessageAt: event.timestamp,
          unreadCount: event.type === 'msg_in' ? 1 : 0,
          updatedAt: event.timestamp,
        })
        .onConflictDoUpdate({
          target: schema.conversations.id,
          set: {
            lastMessage: payload.body || '[Media]',
            lastMessageAt: event.timestamp,
            unreadCount: event.type === 'msg_in' 
              ? sql`${schema.conversations.unreadCount} + 1` 
              : schema.conversations.unreadCount,
            updatedAt: event.timestamp,
          }
        });
    }
  }

  private async processOutbox(accountId: string) {
    const db = getDatabase();
    
    // Get pending outbox entries
    const pendingEntries = await db.select()
      .from(schema.outbox)
      .where(
        and(
          eq(schema.outbox.accountId, accountId),
          eq(schema.outbox.status, 'queued')
        )
      )
      .limit(10);

    for (const entry of pendingEntries) {
      try {
        await this.sendMessage(entry);
      } catch (error) {
        console.error(`Failed to send message ${entry.clientMsgUuid}:`, error);
        
        // Update outbox entry with error
        await db.update(schema.outbox)
          .set({
            status: 'failed',
            lastError: error instanceof Error ? error.message : 'Unknown error',
            retryCount: entry.retryCount + 1,
            updatedAt: new Date().toISOString(),
          })
          .where(eq(schema.outbox.clientMsgUuid, entry.clientMsgUuid));
      }
    }
  }

  private async sendMessage(outboxEntry: any) {
    // Get the original event from local database
    const db = getDatabase();
    const event = await db.select()
      .from(schema.events)
      .where(eq(schema.events.id, outboxEntry.clientMsgUuid))
      .get();

    if (!event) {
      throw new Error(`Event not found for outbox entry ${outboxEntry.clientMsgUuid}`);
    }

    const payload = event.payload as any;
    
    // Send to backend outbox API
    const response = await fetch(`${this.config.backendUrl}/outbox`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.config.authToken}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        client_msg_uuid: outboxEntry.clientMsgUuid,
        account_id: outboxEntry.accountId,
        convo_id: outboxEntry.convoId,
        message_type: payload.message_type,
        content: payload.content,
        reply_to: payload.reply_to,
      }),
    });

    if (!response.ok) {
      throw new Error(`Outbox API error: ${response.status} ${response.statusText}`);
    }

    const result = await response.json();
    
    // Update outbox entry
    await db.update(schema.outbox)
      .set({
        status: 'sent',
        serverMsgId: result.server_msg_id,
        updatedAt: new Date().toISOString(),
      })
      .where(eq(schema.outbox.clientMsgUuid, outboxEntry.clientMsgUuid));
  }

  private startPeriodicSync() {
    this.syncInterval = setInterval(async () => {
      if (!this.isOnline || this.isSyncing) return;
      
      const db = getDatabase();
      const accounts = await db.select().from(schema.accounts);
      
      for (const account of accounts) {
        await this.syncAccount(account.id);
      }
    }, this.config.syncIntervalMs);
  }
}

// SQL import needed for the updateConversationProjection method
import { sql } from 'drizzle-orm';
