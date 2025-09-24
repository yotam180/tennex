import React from 'react';
import { useConversations } from '../hooks/useApi.js';
import { formatDistanceToNow } from 'date-fns';
import { Avatar, AvatarFallback, AvatarImage } from './ui/Avatar.js';
import { Badge } from './ui/Badge.js';
import { ScrollArea } from './ui/ScrollArea.js';

interface ConversationListProps {
  accountId: string;
  selectedConvoId?: string;
  onSelectConversation: (convoId: string) => void;
}

export function ConversationList({ 
  accountId, 
  selectedConvoId, 
  onSelectConversation 
}: ConversationListProps) {
  const { data: conversations, isLoading, error } = useConversations(accountId);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-muted-foreground">Loading conversations...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-destructive">Failed to load conversations</div>
      </div>
    );
  }

  if (!conversations?.length) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-muted-foreground">No conversations yet</div>
      </div>
    );
  }

  return (
    <ScrollArea className="h-full">
      <div className="p-2 space-y-1">
        {conversations.map((conversation) => (
          <div
            key={conversation.id}
            className={`
              flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors
              hover:bg-accent
              ${selectedConvoId === conversation.id ? 'bg-accent' : ''}
            `}
            onClick={() => onSelectConversation(conversation.id)}
          >
            <Avatar className="h-12 w-12">
              <AvatarImage src={conversation.avatarUrl || undefined} />
              <AvatarFallback>
                {conversation.displayName?.slice(0, 2).toUpperCase() || '??'}
              </AvatarFallback>
            </Avatar>

            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between">
                <h3 className="font-medium truncate">
                  {conversation.displayName || conversation.id}
                </h3>
                {conversation.lastMessageAt && (
                  <span className="text-xs text-muted-foreground">
                    {formatDistanceToNow(new Date(conversation.lastMessageAt), { 
                      addSuffix: true 
                    })}
                  </span>
                )}
              </div>

              <div className="flex items-center justify-between mt-1">
                <p className="text-sm text-muted-foreground truncate">
                  {conversation.lastMessage || 'No messages'}
                </p>
                {conversation.unreadCount > 0 && (
                  <Badge variant="default" className="ml-2">
                    {conversation.unreadCount}
                  </Badge>
                )}
              </div>
            </div>

            {conversation.isPinned && (
              <div className="text-muted-foreground">
                📌
              </div>
            )}
          </div>
        ))}
      </div>
    </ScrollArea>
  );
}
