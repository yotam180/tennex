/**
 * Preload Script - Exposes safe APIs to the renderer process
 *
 * This script runs in a sandboxed context and exposes IPC methods via contextBridge.
 * See: https://www.electronjs.org/docs/latest/tutorial/process-model#preload-scripts
 */

import { contextBridge, ipcRenderer } from 'electron';

// Expose database API to renderer process
contextBridge.exposeInMainWorld('electronDB', {
  // Sync State
  getSyncState: (integrationId: number) => ipcRenderer.invoke('db:getSyncState', integrationId),
  upsertSyncState: (data: any) => ipcRenderer.invoke('db:upsertSyncState', data),

  // Conversations
  upsertConversations: (conversations: any[]) =>
    ipcRenderer.invoke('db:upsertConversations', conversations),
  getConversations: (integrationId: number, limit?: number) =>
    ipcRenderer.invoke('db:getConversations', integrationId, limit),

  // Messages
  upsertMessages: (messages: any[]) => ipcRenderer.invoke('db:upsertMessages', messages),
  getMessages: (conversationId: string, limit?: number) =>
    ipcRenderer.invoke('db:getMessages', conversationId, limit),
  getMessagesByIntegration: (integrationId: number, limit?: number) =>
    ipcRenderer.invoke('db:getMessagesByIntegration', integrationId, limit),

  // Contacts
  upsertContacts: (contacts: any[]) => ipcRenderer.invoke('db:upsertContacts', contacts),
  getContacts: (integrationId: number, limit?: number) =>
    ipcRenderer.invoke('db:getContacts', integrationId, limit),

  // Database Stats
  getStats: () => ipcRenderer.invoke('db:getStats'),
});

// TypeScript declaration for global window object
declare global {
  interface Window {
    electronDB: {
      getSyncState: (integrationId: number) => Promise<any>;
      upsertSyncState: (data: any) => Promise<any>;
      upsertConversations: (conversations: any[]) => Promise<number>;
      getConversations: (integrationId: number, limit?: number) => Promise<any[]>;
      upsertMessages: (messages: any[]) => Promise<number>;
      getMessages: (conversationId: string, limit?: number) => Promise<any[]>;
      getMessagesByIntegration: (integrationId: number, limit?: number) => Promise<any[]>;
      upsertContacts: (contacts: any[]) => Promise<number>;
      getContacts: (integrationId: number, limit?: number) => Promise<any[]>;
      getStats: () => Promise<{
        path: string;
        size: number;
        conversations: number;
        messages: number;
        contacts: number;
      }>;
    };
  }
}
