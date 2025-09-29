# Tennex Docker Development Guide

## ✅ Current Status: ALL SERVICES RUNNING!

### 🚀 Services Running:

- ✅ **Backend** (Air hot-reload) - `http://localhost:8000` | gRPC: `localhost:6001`
- ✅ **Bridge** (Air hot-reload) - `http://localhost:6003`
- ✅ **PostgreSQL** - `localhost:5432`
- ✅ **NATS** - `localhost:4222`
- ✅ **MinIO** - `localhost:9000` (Console: `localhost:9001`)
- ✅ **PgAdmin** - `http://localhost:8080`

---

## 🔄 Quick Commands

### Start All Services:

```bash
cd /Users/yotam/projects/tennex/deployments/local
docker-compose --profile full up -d
```

### Stop All Services:

```bash
cd /Users/yotam/projects/tennex/deployments/local
docker-compose down
```

### View Logs (Live):

```bash
# Backend
docker-compose logs -f backend

# Bridge
docker-compose logs -f bridge

# All services
docker-compose logs -f
```

### Rebuild Services (after code changes to dependencies):

```bash
docker-compose build backend bridge
docker-compose --profile full up -d
```

---

## 🔥 Hot Reloading with Air

**Both backend and bridge have Air installed and will automatically reload when you edit code!**

### What Gets Hot-Reloaded:

- ✅ All `.go` files in `services/backend/`
- ✅ All `.go` files in `services/bridge/`
- ✅ All `.go` files in `shared/`
- ✅ All `.go` files in `pkg/`

### Watch the Reloading:

```bash
# In one terminal, watch backend logs
docker-compose logs -f backend

# In another terminal, edit a file
# You'll see: "building..." → "running..." → Service restarts!
```

---

## 🧪 Testing the WhatsApp Integration

### 1. Check Services Are Healthy:

```bash
curl http://localhost:8000/health
curl http://localhost:6003/health
```

### 2. Create a Test User (if needed):

```bash
curl -X POST http://localhost:8000/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'
```

### 3. Login and Get JWT:

```bash
curl -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

### 4. Connect WhatsApp:

```bash
# Replace YOUR_JWT_TOKEN with the token from step 3
curl -X POST http://localhost:6003/whatsapp/connect \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Expected Response:**

```json
{
  "qr_code": "2@ABC123...",
  "session_id": "uuid-here",
  "expires_at": "2025-09-29T21:40:00Z",
  "instructions": "Open WhatsApp on your phone..."
}
```

### 5. Scan QR Code with WhatsApp Mobile

- Open WhatsApp on your phone
- Tap Menu → Linked Devices → Link a Device
- Scan the QR code from the response

### 6. Watch the Magic Happen! 🪄

**Bridge Logs:**

```bash
docker-compose logs -f bridge
```

You'll see:

```
🎉 QR scan successful! Session established
👤 User ID: <user-id>
📱 WhatsApp JID: <phone>@s.whatsapp.net
✅ User integration created: ID=<integration-id>
✅ Backend notified of WhatsApp connection
🔗 WhatsApp Connected!
🔄 History Sync: type=RECENT, conversations=<count>
✅ Synced <count> conversations from history
✅ Synced <count> messages for conversation <id>
```

**Backend Logs:**

```bash
docker-compose logs -f backend
```

You'll see:

```
📥 CreateUserIntegration: user_id=<id>, type=whatsapp
✅ User integration created: integration_id=<id>
📥 SyncConversations: batch 1/1 (50 conversations)
✅ Conversations sync completed: 50 processed
📥 SyncMessages: batch 1/5 (200 messages)
✅ Messages sync completed: 1000 processed
```

---

## 🗄️ Check the Database

### Connect to PostgreSQL:

```bash
docker exec -it tennex-postgres psql -U tennex -d tennex
```

### Check Synced Data:

```sql
-- User integrations
SELECT * FROM user_integrations;

-- Conversations
SELECT id, platform_id, name, type, is_archived, is_pinned
FROM conversations
ORDER BY created_at DESC
LIMIT 10;

-- Messages
SELECT id, platform_id, conversation_id, message_type, content, is_from_me, timestamp
FROM messages
ORDER BY timestamp DESC
LIMIT 10;

-- Contacts
SELECT * FROM contacts ORDER BY created_at DESC LIMIT 10;

-- Exit
\q
```

### Or Use PgAdmin:

1. Open http://localhost:8080
2. Login: `admin@tennex.com` / `admin123`
3. Navigate to Servers → Tennex → Databases → tennex → Schemas → public → Tables

---

## 🐛 Debugging

### Service Won't Start?

**Check logs:**

```bash
docker-compose logs backend
docker-compose logs bridge
```

**Restart a specific service:**

```bash
docker-compose restart backend
docker-compose restart bridge
```

**Rebuild from scratch:**

```bash
docker-compose down
docker-compose build backend bridge
docker-compose --profile full up -d
```

### Database Issues?

**Check PostgreSQL:**

```bash
docker-compose logs postgres
```

**Reset database (⚠️ DESTRUCTIVE):**

```bash
# From project root
source deployments/local/shell_shortcuts.sh
txdbreset
```

### Code Changes Not Picked Up?

**Check if Air is watching:**

```bash
docker-compose logs backend | grep watching
docker-compose logs bridge | grep watching
```

**Force rebuild if dependencies changed:**

```bash
docker-compose build backend bridge
```

### Port Already in Use?

**Check what's using the port:**

```bash
lsof -ti:8000  # Backend HTTP
lsof -ti:6001  # Backend gRPC
lsof -ti:6003  # Bridge HTTP
```

**Kill the process:**

```bash
lsof -ti:8000 | xargs kill -9
```

---

## 🔧 Configuration

### Environment Variables

**Backend** (`docker-compose.yml`):

- `TENNEX_HTTP_PORT=8000`
- `TENNEX_GRPC_PORT=6001`
- `TENNEX_DATABASE_URL=postgres://tennex:tennex123@postgres:5432/tennex?sslmode=disable`
- `TENNEX_NATS_URL=nats://nats:4222`
- `TENNEX_AUTH_JWT_SECRET=dev-jwt-secret-change-in-production`

**Bridge** (`docker-compose.yml`):

- `TENNEX_HTTP_PORT=6003`
- `TENNEX_DATABASE_URL=postgres://tennex:tennex123@postgres:5432/tennex?sslmode=disable`
- `BACKEND_GRPC_ADDR=backend:6001` ← **Important for gRPC communication**
- `JWT_SECRET=dev-jwt-secret-change-in-production`

---

## 📂 Volume Mounts (Hot Reload)

### Backend:

```yaml
- ../../services/backend:/app/services/backend
- ../../shared:/app/shared
- ../../pkg:/app/pkg
- backend_go_cache:/go/pkg/mod
```

### Bridge:

```yaml
- ../../services/bridge:/app/services/bridge
- ../../shared:/app/shared
- ../../pkg:/app/pkg
- bridge_go_cache:/go/pkg/mod
```

**This means any changes to these directories will trigger Air to rebuild!**

---

## 🎯 What's Next?

1. **Scan a WhatsApp QR code** and watch your message history sync
2. **Send a message on WhatsApp** and watch it appear in the database
3. **Edit code** and watch Air automatically reload
4. **Build your frontend** to display the synced messages

---

## 🛑 Stopping Everything

```bash
cd /Users/yotam/projects/tennex/deployments/local
docker-compose down
```

To remove volumes too (⚠️ deletes all data):

```bash
docker-compose down -v
```

---

Happy Coding! 🚀
