import { ipcMain } from 'electron';
import { drizzle } from 'drizzle-orm/better-sqlite3';
import { eq, desc, and, gte } from 'drizzle-orm';
import { v4 as uuidv4 } from 'uuid';
import { schema } from '../database/index.js';
import { SyncService } from '../sync/syncService.js';
import { setSyncService } from '../index.js';
import { getBackendUrl, getSyncInterval } from '../../shared/config.js';

export function registerIpcHandlers(db: ReturnType<typeof drizzle<typeof schema>>) {
  
  // Authentication
  ipcMain.handle('auth:login', async (_, credentials: { username: string; password: string }) => {
    try {
      const response = await fetch(`${getBackendUrl()}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(credentials),
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(`Login failed: ${response.status} ${errorData}`);
      }

      const authData = await response.json() as any;
      
      // Initialize sync service with auth token
      const syncService = new SyncService({
        backendUrl: getBackendUrl(),
        authToken: authData.token,
        syncIntervalMs: getSyncInterval(),
      });
      
      setSyncService(syncService);
      await syncService.start();

      return authData;
    } catch (error) {
      throw new Error(`Login failed: ${error instanceof Error ? error.message : error}`);
    }
  });

  ipcMain.handle('auth:register', async (_, userData: {
    username: string;
    password: string;
    email: string;
    full_name?: string;
  }) => {
    try {
      const response = await fetch(`${getBackendUrl()}/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(userData),
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(`Registration failed: ${response.status} ${errorData}`);
      }

      const authData = await response.json() as any;
      
      // Initialize sync service with auth token
      const syncService = new SyncService({
        backendUrl: getBackendUrl(),
        authToken: authData.token,
        syncIntervalMs: getSyncInterval(),
      });
      
      setSyncService(syncService);
      await syncService.start();

      return authData;
    } catch (error) {
      throw new Error(`Registration failed: ${error instanceof Error ? error.message : error}`);
    }
  });

  ipcMain.handle('auth:me', async () => {
    try {
      // TODO: Implement token storage and retrieval
      // For now, this endpoint won't work without token management
      throw new Error('Token management not yet implemented');
      
      // const response = await fetch(`${getBackendUrl()}/auth/me`, {
      //   headers: {
      //     'Authorization': `Bearer ${storedToken}`,
      //     'Content-Type': 'application/json',
      //   },
      // });

      // if (!response.ok) {
      //   throw new Error('Failed to get current user');
      // }

      // return await response.json();
    } catch (error) {
      throw new Error(`Failed to get current user: ${error instanceof Error ? error.message : error}`);
    }
  });

  // Account management
  ipcMain.handle('accounts:list', async () => {
    return await db.select().from(schema.accounts).orderBy(desc(schema.accounts.createdAt));
  });

  ipcMain.handle('accounts:get', async (_, accountId: string) => {
    return await db.select()
      .from(schema.accounts)
      .where(eq(schema.accounts.id, accountId))
      .get();
  });

  // Conversations
  ipcMain.handle('conversations:list', async (_, accountId: string) => {
    return await db.select()
      .from(schema.conversations)
      .where(eq(schema.conversations.accountId, accountId))
      .orderBy(desc(schema.conversations.lastMessageAt));
  });

  ipcMain.handle('conversations:get', async (_, convoId: string) => {
    return await db.select()
      .from(schema.conversations)
      .where(eq(schema.conversations.id, convoId))
      .get();
  });

  // Messages/Events
  ipcMain.handle('messages:list', async (_, convoId: string, limit = 50, beforeSeq?: number) => {
    const whereConditions = [
      eq(schema.events.convoId, convoId),
      eq(schema.events.applied, true)
    ];

    if (beforeSeq) {
      whereConditions.push(gte(schema.events.seq, beforeSeq));
    }

    const events = await db.select()
      .from(schema.events)
      .where(and(...whereConditions))
      .orderBy(desc(schema.events.seq))
      .limit(limit);

    return events.reverse(); // Return in chronological order
  });

  // Send message
  ipcMain.handle('messages:send', async (_, messageData: {
    accountId: string;
    convoId: string;
    messageType: 'text' | 'image' | 'audio' | 'video' | 'document';
    content: any;
    replyTo?: string;
  }) => {
    const clientMsgUuid = uuidv4();
    const now = new Date().toISOString();

    // Create local event
    const localEvent = {
      id: clientMsgUuid,
      timestamp: now,
      type: 'msg_out_pending' as const,
      accountId: messageData.accountId,
      convoId: messageData.convoId,
      payload: {
        message_type: messageData.messageType,
        content: messageData.content,
        reply_to: messageData.replyTo,
      },
      applied: true,
    };

    // Insert into local events
    await db.insert(schema.events).values(localEvent);

    // Insert into outbox for sync
    await db.insert(schema.outbox).values({
      clientMsgUuid,
      accountId: messageData.accountId,
      convoId: messageData.convoId,
      status: 'queued',
      createdAt: now,
      updatedAt: now,
    });

    // Update conversation immediately for better UX
    await db.insert(schema.conversations)
      .values({
        id: messageData.convoId,
        accountId: messageData.accountId,
        lastMessage: typeof messageData.content === 'string' 
          ? messageData.content 
          : '[Media]',
        lastMessageAt: now,
        updatedAt: now,
      })
      .onConflictDoUpdate({
        target: schema.conversations.id,
        set: {
          lastMessage: typeof messageData.content === 'string' 
            ? messageData.content 
            : '[Media]',
          lastMessageAt: now,
          updatedAt: now,
        }
      });

    return { clientMsgUuid, timestamp: now };
  });

  // Sync status
  ipcMain.handle('sync:status', async (_, accountId: string) => {
    const syncState = await db.select()
      .from(schema.syncState)
      .where(eq(schema.syncState.accountId, accountId))
      .get();

    const pendingCount = await db.select({ count: sql`count(*)` })
      .from(schema.outbox)
      .where(
        and(
          eq(schema.outbox.accountId, accountId),
          eq(schema.outbox.status, 'queued')
        )
      )
      .get();

    return {
      isOnline: syncState?.isOnline || false,
      lastSyncAt: syncState?.lastSyncAt,
      pendingMessages: pendingCount?.count || 0,
    };
  });

  // Media management
  ipcMain.handle('media:download', async (_, contentHash: string) => {
    const mediaBlob = await db.select()
      .from(schema.mediaBlobs)
      .where(eq(schema.mediaBlobs.contentHash, contentHash))
      .get();

    if (!mediaBlob) {
      throw new Error('Media not found');
    }

    // If already downloaded, return local path
    if (mediaBlob.downloadStatus === 'completed' && mediaBlob.localPath) {
      return mediaBlob.localPath;
    }

    // TODO: Implement actual download logic
    // This would download from mediaBlob.storageUrl to local filesystem
    // and update the media_blobs table with local path and status

    throw new Error('Media download not implemented');
  });

  // QR Code for pairing
  ipcMain.handle('auth:getQR', async (_, accountId: string) => {
    try {
      const response = await fetch(`${getBackendUrl()}/qr?account_id=${accountId}`);
      
      if (!response.ok) {
        throw new Error('Failed to get QR code');
      }

      return await response.json();
    } catch (error) {
      throw new Error(`QR generation failed: ${error}`);
    }
  });
}

// Import sql for the sync:status handler
import { sql } from 'drizzle-orm';
