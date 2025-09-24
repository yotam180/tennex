import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import type { Account, Conversation, Event } from '../../main/database/schema.js';

// IPC wrapper types
interface IPC {
  invoke(channel: string, ...args: any[]): Promise<any>;
}

declare global {
  interface Window {
    electronAPI: IPC;
  }
}

// Query keys
export const queryKeys = {
  accounts: ['accounts'] as const,
  account: (id: string) => ['accounts', id] as const,
  conversations: (accountId: string) => ['conversations', accountId] as const,
  conversation: (id: string) => ['conversations', id] as const,
  messages: (convoId: string) => ['messages', convoId] as const,
  syncStatus: (accountId: string) => ['sync', accountId] as const,
} as const;

// Authentication
export function useLogin() {
  return useMutation({
    mutationFn: async (credentials: { username: string; password: string }) => {
      return await window.electronAPI.invoke('auth:login', credentials);
    },
  });
}

export function useRegister() {
  return useMutation({
    mutationFn: async (userData: {
      username: string;
      password: string;
      email: string;
      full_name?: string;
    }) => {
      return await window.electronAPI.invoke('auth:register', userData);
    },
  });
}

export function useCurrentUser() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: async () => {
      return await window.electronAPI.invoke('auth:me');
    },
    retry: false, // Don't retry if token is invalid
  });
}

export function useGetQRCode(accountId: string) {
  return useQuery({
    queryKey: ['qr', accountId],
    queryFn: async () => {
      return await window.electronAPI.invoke('auth:getQR', accountId);
    },
    enabled: !!accountId,
    staleTime: 30000, // QR codes expire quickly
  });
}

// Accounts
export function useAccounts() {
  return useQuery({
    queryKey: queryKeys.accounts,
    queryFn: async (): Promise<Account[]> => {
      return await window.electronAPI.invoke('accounts:list');
    },
  });
}

export function useAccount(accountId: string) {
  return useQuery({
    queryKey: queryKeys.account(accountId),
    queryFn: async (): Promise<Account | undefined> => {
      return await window.electronAPI.invoke('accounts:get', accountId);
    },
    enabled: !!accountId,
  });
}

// Conversations
export function useConversations(accountId: string) {
  return useQuery({
    queryKey: queryKeys.conversations(accountId),
    queryFn: async (): Promise<Conversation[]> => {
      return await window.electronAPI.invoke('conversations:list', accountId);
    },
    enabled: !!accountId,
  });
}

export function useConversation(convoId: string) {
  return useQuery({
    queryKey: queryKeys.conversation(convoId),
    queryFn: async (): Promise<Conversation | undefined> => {
      return await window.electronAPI.invoke('conversations:get', convoId);
    },
    enabled: !!convoId,
  });
}

// Messages
export function useMessages(convoId: string, limit = 50) {
  return useQuery({
    queryKey: queryKeys.messages(convoId),
    queryFn: async (): Promise<Event[]> => {
      return await window.electronAPI.invoke('messages:list', convoId, limit);
    },
    enabled: !!convoId,
  });
}

export function useSendMessage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (messageData: {
      accountId: string;
      convoId: string;
      messageType: 'text' | 'image' | 'audio' | 'video' | 'document';
      content: any;
      replyTo?: string;
    }) => {
      return await window.electronAPI.invoke('messages:send', messageData);
    },
    onSuccess: (_, variables) => {
      // Invalidate and refetch messages for this conversation
      queryClient.invalidateQueries({
        queryKey: queryKeys.messages(variables.convoId),
      });
      
      // Invalidate conversations to update last message
      queryClient.invalidateQueries({
        queryKey: queryKeys.conversations(variables.accountId),
      });
    },
  });
}

// Sync status
export function useSyncStatus(accountId: string) {
  return useQuery({
    queryKey: queryKeys.syncStatus(accountId),
    queryFn: async () => {
      return await window.electronAPI.invoke('sync:status', accountId);
    },
    enabled: !!accountId,
    refetchInterval: 5000, // Check sync status every 5 seconds
  });
}

// Media
export function useDownloadMedia() {
  return useMutation({
    mutationFn: async (contentHash: string) => {
      return await window.electronAPI.invoke('media:download', contentHash);
    },
  });
}
