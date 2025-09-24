import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState } from 'react';
import './App.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      refetchOnWindowFocus: false,
    },
  },
});

function App() {
  const [currentAccount] = useState<string | null>(null);

  return (
    <QueryClientProvider client={queryClient}>
      <div className="h-screen bg-background text-foreground">
        {currentAccount ? (
          <MainInterface accountId={currentAccount} />
        ) : (
          <AuthInterface />
        )}
      </div>
    </QueryClientProvider>
  );
}

function AuthInterface() {
  return (
    <div className="flex items-center justify-center h-full">
      <div className="text-center space-y-4">
        <h1 className="text-2xl font-bold">Tennex</h1>
        <p className="text-muted-foreground">WhatsApp messaging client</p>
        <button className="px-4 py-2 bg-primary text-primary-foreground rounded-md">
          Connect Account
        </button>
      </div>
    </div>
  );
}

function MainInterface({ accountId }: { accountId: string }) {
  const [selectedConvo] = useState<string | null>(null);
  
  // TODO: Use accountId for fetching account-specific data
  console.log('Current account:', accountId);

  return (
    <div className="flex h-full">
      {/* Sidebar */}
      <div className="w-80 border-r border-border bg-muted/30">
        <div className="p-4 border-b border-border">
          <h2 className="font-semibold">Conversations</h2>
        </div>
        {/* ConversationList would go here */}
        <div className="p-4 text-sm text-muted-foreground">
          No conversations yet
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col">
        {selectedConvo ? (
          <>
            {/* Messages */}
            <div className="flex-1 p-4">
              {/* MessageList would go here */}
              <div className="text-muted-foreground">Select a conversation</div>
            </div>
            
            {/* Message input */}
            <div className="border-t border-border p-4">
              <input 
                type="text" 
                placeholder="Type a message..."
                className="w-full px-3 py-2 border border-border rounded-md bg-background"
              />
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center">
            <div className="text-center text-muted-foreground">
              <p>Select a conversation to start messaging</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;
