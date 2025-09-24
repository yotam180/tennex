import { useRef, useEffect } from 'react';
import { useMessages } from '../hooks/useApi.js';
import { format } from 'date-fns';
import { Avatar, AvatarFallback, AvatarImage } from './ui/Avatar.js';
import { ScrollArea } from './ui/ScrollArea.js';
import { Badge } from './ui/Badge.js';

interface MessageListProps {
  convoId: string;
}

export function MessageList({ convoId }: MessageListProps) {
  const { data: messages, isLoading, error } = useMessages(convoId);
  const scrollAreaRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (scrollAreaRef.current) {
      const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]');
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight;
      }
    }
  }, [messages]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-muted-foreground">Loading messages...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-destructive">Failed to load messages</div>
      </div>
    );
  }

  if (!messages?.length) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-sm text-muted-foreground">No messages yet</div>
      </div>
    );
  }

  return (
    <ScrollArea ref={scrollAreaRef} className="h-full">
      <div className="p-4 space-y-4">
        {messages.map((message, index) => {
          const isOutgoing = message.type.startsWith('msg_out');
          const prevMessage = index > 0 ? messages[index - 1] : null;
          const showDateSeparator = !prevMessage || 
            !isSameDay(new Date(message.timestamp), new Date(prevMessage.timestamp));

          return (
            <div key={message.id}>
              {showDateSeparator && (
                <div className="flex items-center gap-4 my-6">
                  <div className="flex-1 h-px bg-border" />
                  <span className="text-xs text-muted-foreground bg-background px-2">
                    {format(new Date(message.timestamp), 'MMMM d, yyyy')}
                  </span>
                  <div className="flex-1 h-px bg-border" />
                </div>
              )}

              <MessageBubble
                message={message}
                isOutgoing={isOutgoing}
                showSenderInfo={!isOutgoing && (!prevMessage || prevMessage.senderJid !== message.senderJid)}
              />
            </div>
          );
        })}
      </div>
    </ScrollArea>
  );
}

function MessageBubble({ 
  message, 
  isOutgoing, 
  showSenderInfo 
}: { 
  message: any; 
  isOutgoing: boolean; 
  showSenderInfo: boolean;
}) {
  const payload = message.payload as any;
  const messageStatus = getMessageStatus(message.type);

  return (
    <div className={`flex gap-3 ${isOutgoing ? 'justify-end' : 'justify-start'}`}>
      {!isOutgoing && (
        <Avatar className="h-8 w-8 mt-1">
          <AvatarImage src={payload.sender_avatar} />
          <AvatarFallback>
            {message.senderJid?.slice(0, 2).toUpperCase() || '??'}
          </AvatarFallback>
        </Avatar>
      )}

      <div className={`max-w-[70%] ${isOutgoing ? 'items-end' : 'items-start'} flex flex-col`}>
        {showSenderInfo && !isOutgoing && (
          <span className="text-xs text-muted-foreground mb-1">
            {payload.sender_name || message.senderJid}
          </span>
        )}

        <div className={`
          rounded-lg px-3 py-2 max-w-full break-words
          ${isOutgoing 
            ? 'bg-primary text-primary-foreground' 
            : 'bg-muted'
          }
        `}>
          {payload.content?.body && (
            <p className="text-sm">{payload.content.body}</p>
          )}

          {payload.content?.media && (
            <div className="mt-2">
              <MediaContent media={payload.content.media} />
            </div>
          )}

          {payload.reply_to && (
            <div className="border-l-2 border-muted-foreground/30 pl-2 mt-2 text-xs text-muted-foreground">
              Replying to a message
            </div>
          )}
        </div>

        <div className={`flex items-center gap-1 mt-1 text-xs text-muted-foreground ${isOutgoing ? 'flex-row-reverse' : ''}`}>
          <span>
            {format(new Date(message.timestamp), 'h:mm a')}
          </span>
          
          {isOutgoing && messageStatus && (
            <Badge variant="secondary" className="text-xs">
              {messageStatus}
            </Badge>
          )}
        </div>
      </div>
    </div>
  );
}

function MediaContent({ media }: { media: any }) {
  // TODO: Implement media rendering based on type
  return (
    <div className="bg-muted/50 rounded p-2 text-sm">
      📎 {media.type || 'Media'} ({media.size || 'Unknown size'})
    </div>
  );
}

function getMessageStatus(eventType: string): string | null {
  switch (eventType) {
    case 'msg_out_pending':
      return 'Sending';
    case 'msg_out_sent':
      return 'Sent';
    case 'msg_delivery':
      return 'Delivered';
    default:
      return null;
  }
}

function isSameDay(date1: Date, date2: Date): boolean {
  return date1.getFullYear() === date2.getFullYear() &&
         date1.getMonth() === date2.getMonth() &&
         date1.getDate() === date2.getDate();
}
