import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useState } from 'react';
import { useAuthStore } from './stores/authStore.js';
import { LoginForm } from './components/auth/LoginForm.js';
import { RegisterForm } from './components/auth/RegisterForm.js';
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
  return (
    <QueryClientProvider client={queryClient}>
      <div className="h-screen bg-background text-foreground">
        <AppContent />
      </div>
    </QueryClientProvider>
  );
}

function AppContent() {
  const { isAuthenticated, user } = useAuthStore();

  if (isAuthenticated && user) {
    return <MainInterface user={user} />;
  }

  return <AuthInterface />;
}

function AuthInterface() {
  const [showRegister, setShowRegister] = useState(false);

  return (
    <div className="flex items-center justify-center h-full bg-gradient-to-br from-background to-muted/20">
      <div className="w-full max-w-md mx-auto p-6">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold bg-gradient-to-r from-primary to-primary/70 bg-clip-text text-transparent">
            Tennex
          </h1>
          <p className="text-muted-foreground mt-2">
            Your local-first WhatsApp client
          </p>
        </div>

        {showRegister ? (
          <RegisterForm onSwitchToLogin={() => setShowRegister(false)} />
        ) : (
          <LoginForm onSwitchToRegister={() => setShowRegister(true)} />
        )}
      </div>
    </div>
  );
}

function MainInterface({ user }: { user: any }) {
  const logout = useAuthStore((state) => state.logout);

  return (
    <div className="flex h-full">
      {/* Sidebar */}
      <div className="w-80 border-r border-border bg-muted/30">
        <div className="p-4 border-b border-border">
          <div className="flex items-center justify-between">
            <h2 className="font-semibold">Conversations</h2>
            <button
              onClick={logout}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Logout
            </button>
          </div>
        </div>
        {/* ConversationList would go here */}
        <div className="p-4 text-sm text-muted-foreground">
          No conversations yet
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 flex flex-col">
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center space-y-4">
            <div className="text-6xl">👋</div>
            <h1 className="text-2xl font-bold">
              Hello, {user.full_name || user.username}!
            </h1>
            <p className="text-muted-foreground">
              Welcome to your local-first WhatsApp client
            </p>
            <div className="text-sm text-muted-foreground bg-muted/50 p-4 rounded-lg max-w-md">
              <p><strong>Username:</strong> {user.username}</p>
              <p><strong>Email:</strong> {user.email}</p>
              {user.full_name && <p><strong>Name:</strong> {user.full_name}</p>}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}


export default App;
