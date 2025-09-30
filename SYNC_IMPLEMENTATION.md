# WhatsApp Data Sync Implementation

This document describes the complete implementation of the local-first WhatsApp data syncing system between the backend and Electron client.

---

## 🎯 **Overview**

We've implemented a **sequence-based, cursor-paginated sync system** that allows the Electron client to:

1. **Pull complete history** from the backend on first sync
2. **Incrementally sync** new data on subsequent syncs
3. **Store all data locally** in SQLite for offline access
4. **Track sync state** to avoid re-syncing old data

---

## 🏗️ **Architecture**

```
┌─────────────────────────────────────────────────────┐
│                  Electron Client                     │
│                                                      │
│  ┌─────────────┐        IPC         ┌────────────┐  │
│  │  Renderer   │ ◄─────────────────► │    Main    │  │
│  │  (React)    │    contextBridge    │  Process   │  │
│  └─────────────┘                     └─────┬──────┘  │
│                                             │         │
│                                             ▼         │
│                                      ┌──────────────┐ │
│                                      │   SQLite DB  │ │
│                                      │ (better-     │ │
│                                      │  sqlite3)    │ │
│                                      └──────────────┘ │
└─────────────────────────────────────────────────────┘
                        ▲
                        │ HTTP REST API
                        │ (Seq-based pagination)
                        ▼
┌─────────────────────────────────────────────────────┐
│                   Backend Service                    │
│                                                      │
│  ┌──────────────┐           ┌─────────────────┐     │
│  │ HTTP Handlers│           │  PostgreSQL     │     │
│  │ (Chi Router) │ ◄────────►│  (SQLC Queries) │     │
│  └──────────────┘           └─────────────────┘     │
└─────────────────────────────────────────────────────┘
```

---

## 📦 **Backend Implementation**

### **1. Database Schema Changes**

**File:** `pkg/db/schema/006_add_sync_sequences.sql`

Added `seq` columns to all syncable tables:

- `conversations.seq`
- `messages.seq`
- `contacts.seq`
- `message_media.seq`
- `conversation_participants.seq`

These are **BIGSERIAL** (auto-incrementing integers) that provide a monotonically increasing sequence number for cursor-based pagination.

### **2. SQLC Queries**

**Files:**

- `pkg/db/queries/messages.sql`
- `pkg/db/queries/conversations.sql`
- `pkg/db/queries/contacts.sql`

Added queries for fetching data **since a sequence number**:

```sql
-- Example: Fetch messages since seq X
SELECT * FROM messages
WHERE integration_id = $1 AND seq > $2
ORDER BY seq ASC
LIMIT $3;
```

### **3. REST API Endpoints**

**File:** `pkg/api/openapi.yaml`

Added sync endpoints:

- `GET /sync/conversations/{integration_id}?since_seq=X&limit=N`
- `GET /sync/messages/{integration_id}?since_seq=X&limit=N`
- `GET /sync/contacts/{integration_id}?since_seq=X&limit=N`
- `GET /sync/status/{integration_id}` - Returns latest seq numbers

**Response Format:**

```json
{
  "conversations": [...],
  "latest_seq": 12345,
  "has_more": true,
  "total_count": 150
}
```

### **4. HTTP Handlers**

**File:** `services/backend/internal/http/handlers/api_handler.go`

Implemented handlers that:

- Use SQLC queries to fetch paginated data
- Return `latest_seq` for cursor-based pagination
- Return `has_more` to indicate if more pages exist

---

## 💾 **Electron Client Implementation**

### **1. SQLite Schema**

**File:** `clients/electron/src/db/schema.ts`

Defined Drizzle ORM schema mirroring backend tables:

- `sync_state` - Tracks last synced seq numbers
- `conversations`
- `messages`
- `message_media`
- `contacts`
- `conversation_participants`

**Key differences from backend:**

- Uses SQLite instead of PostgreSQL
- Timestamps stored as Unix timestamps (integers)
- JSON stored as TEXT with `mode: 'json'`

### **2. Database Service**

**File:** `clients/electron/src/db/database.ts`

Main process database service that:

- Creates SQLite connection using `better-sqlite3`
- Initializes tables on first run
- Enables WAL mode for better concurrency
- Provides `getStats()` for database metrics

**Database Location:**

- **macOS:** `~/Library/Application Support/Minimal UI/tennex.db`
- **Windows:** `C:\Users\<username>\AppData\Roaming\Minimal UI\tennex.db`
- **Linux:** `~/.config/Minimal UI/tennex.db`

### **3. Database Operations**

**File:** `clients/electron/src/db/operations.ts`

Exposes functions for:

- `upsertConversations()` - Bulk upsert conversations
- `upsertMessages()` - Bulk upsert messages
- `upsertContacts()` - Bulk upsert contacts
- `getSyncState()` - Get last sync state
- `upsertSyncState()` - Update sync state
- `getStats()` - Get database statistics

All operations use **idempotent upserts** (INSERT ON CONFLICT DO UPDATE).

### **4. IPC Communication**

**File:** `clients/electron/src/main.ts`

Main process sets up IPC handlers:

```typescript
ipcMain.handle("db:upsertConversations", async (_, conversations) => {
  return await dbOps.upsertConversations(conversations);
});
```

**File:** `clients/electron/src/preload.ts`

Exposes safe API to renderer via `contextBridge`:

```typescript
contextBridge.exposeInMainWorld("electronDB", {
  upsertConversations: (conversations) =>
    ipcRenderer.invoke("db:upsertConversations", conversations),
  // ... other methods
});
```

### **5. Sync UI**

**File:** `clients/electron/src/renderer/sections/whatsapp/view/whatsapp-sync-view.tsx`

React component that:

- Shows database statistics (conversations, messages, contacts, size)
- Provides "Sync All Data" button
- Shows real-time progress bars during sync
- Handles errors gracefully

**Sync Flow:**

```
1. Fetch conversations (100 per page)
   └─ Store in SQLite via IPC
2. Fetch messages (1500 per page)
   └─ Store in SQLite via IPC
3. Fetch contacts (500 per page)
   └─ Store in SQLite via IPC
4. Update sync_state with latest seq numbers
5. Refresh database stats
```

---

## 🔄 **Sync Strategies**

### **Initial Sync (First Time)**

```typescript
// 1. Get current sync state (all seq = 0 for first time)
const syncState = await window.electronDB.getSyncState(integrationId);
const sinceSeq = syncState?.lastConvSeq || 0;

// 2. Fetch all conversations since seq=0
while (hasMore) {
  const response = await axios.get(`/sync/conversations/${integrationId}`, {
    params: { since_seq: sinceSeq, limit: 100 },
  });

  // 3. Store in SQLite
  await window.electronDB.upsertConversations(response.data.conversations);

  // 4. Update cursor
  sinceSeq = response.data.latest_seq;
  hasMore = response.data.has_more;
}

// 5. Update sync state
await window.electronDB.upsertSyncState({
  integrationId,
  lastConvSeq: sinceSeq,
});
```

### **Incremental Sync (Subsequent Syncs)**

Same flow, but starts from `lastConvSeq` instead of 0.

### **Gap Filling (After Offline Period)**

Just use the same incremental sync - it will fetch everything since the last `lastMessageSeq`.

---

## 🎨 **UI Features**

### **Database Stats Card**

Shows:

- Number of conversations
- Number of messages
- Number of contacts
- Database size in MB
- Database file path

### **Sync Progress**

Real-time progress bars showing:

- Current stage (conversations/messages/contacts)
- Number of items synced
- Visual progress indicator

### **Error Handling**

- Network errors displayed in alerts
- Sync can be retried
- Database errors logged to console

---

## 🔧 **Tools & Libraries Used**

### **Backend**

- **PostgreSQL** - Main database
- **SQLC** - Type-safe SQL query generation
- **Chi** - HTTP router
- **Zap** - Structured logging

### **Frontend**

- **better-sqlite3** - Native SQLite bindings for Node.js
- **Drizzle ORM** - TypeScript ORM for SQLite
- **Electron IPC** - Inter-process communication
- **React** - UI framework
- **Material-UI** - Component library

---

## 📊 **Performance Characteristics**

### **Page Sizes**

- **Conversations:** 100 per page
- **Messages:** 1,500 per page
- **Contacts:** 500 per page

### **Database Indexes**

All `seq` columns are indexed for fast lookups:

```sql
CREATE INDEX idx_messages_seq ON messages(seq);
CREATE INDEX idx_conversations_seq ON conversations(seq);
```

### **Transaction Batching**

All upserts use transactions for atomicity:

```typescript
const insertMany = sqlite.transaction((messages) => {
  for (const msg of messages) {
    stmt.run(...msg);
  }
});
```

---

## 🚀 **Future Enhancements**

### **1. Real-Time Event Streaming**

- Use NATS/WebSocket for live updates
- Event bus publishes changes to all connected clients
- Client applies events incrementally

### **2. Conflict Resolution**

- Last-write-wins for now
- Future: Operational Transform (OT) or CRDTs

### **3. Selective Sync**

- Allow users to choose which conversations to sync
- Archive old conversations locally

### **4. Media Download**

- Queue media downloads separately
- Store files in user's Downloads folder
- Show download progress per media item

### **5. Full-Text Search**

- Add FTS5 virtual table in SQLite
- Search across all message content
- Highlight search results

---

## 🧪 **Testing**

### **Manual Testing**

1. Start Electron app
2. Navigate to `/dashboard/whatsapp/sync`
3. Click "Sync All Data"
4. Verify:
   - Progress bars update
   - Database stats increase
   - No errors in console

### **Database Verification**

```bash
# On macOS
sqlite3 ~/Library/Application\ Support/Minimal\ UI/tennex.db

# Check data
SELECT COUNT(*) FROM conversations;
SELECT COUNT(*) FROM messages;
SELECT * FROM sync_state;
```

---

## 📝 **Key Decisions**

1. **Why sequence numbers instead of timestamps?**

   - Timestamps can have clock skew issues
   - Sequences are guaranteed to be monotonically increasing
   - Easier to handle edge cases (same timestamp for multiple records)

2. **Why idempotent upserts?**

   - Allows safe re-syncing without duplicates
   - Handles network failures gracefully
   - Simplifies sync logic (no need to track "already synced")

3. **Why separate sync_state table?**

   - Clean separation of sync metadata from data
   - Easy to reset sync (just delete sync_state row)
   - Can track multiple integrations independently

4. **Why IPC instead of direct SQLite access?**
   - Security: Renderer process is sandboxed
   - Best practice: Main process owns database
   - Enables future optimizations (e.g., caching)

---

## 🎉 **What's Working Now**

✅ Backend sync endpoints working  
✅ SQLite database created on app launch  
✅ IPC communication established  
✅ Full sync flow implemented  
✅ Database stats displayed  
✅ Progress indicators working  
✅ Error handling in place  
✅ Idempotent upserts  
✅ Cursor-based pagination  
✅ Sync state tracking

**Ready to sync WhatsApp data!** 🚀
