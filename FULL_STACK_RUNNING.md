# 🚀 Full Tennex Stack - All Services Running!

## ✅ Current Status: EVERYTHING IS LIVE!

### Backend Services (Docker with Air Hot-Reload):

- ✅ **Backend API** - `http://localhost:8000` (gRPC: `localhost:6001`)
- ✅ **Bridge Service** - `http://localhost:6003` (WhatsApp integration)
- ✅ **PostgreSQL** - `localhost:5432`
- ✅ **NATS** - `localhost:4222`
- ✅ **MinIO** - `localhost:9000`
- ✅ **PgAdmin** - `http://localhost:8080`

### Frontend:

- ✅ **Electron App** - Starting now! (Will open in a new window)

---

## 🎯 What You Can Do Now

### 1. **Login to the Frontend**

When the Electron app opens, you can:

- Sign up for a new account
- Or login with existing credentials

**Default connection:** The app automatically connects to `http://localhost:8000`

---

### 2. **Connect WhatsApp**

Once logged in:

1. Navigate to **Settings** → **Integrations**
2. Click **Connect WhatsApp**
3. Scan the QR code with your phone
4. Watch your messages sync! 📱

---

### 3. **Watch the Magic in Real-Time**

#### **Terminal 1 - Backend Logs:**

```bash
cd /Users/yotam/projects/tennex/deployments/local
docker-compose logs -f backend
```

You'll see:

```
📥 CreateUserIntegration: user_id=<id>, type=whatsapp
✅ User integration created
📥 SyncConversations: batch 1/1 (50 conversations)
✅ Conversations sync completed
📥 SyncMessages: batch 1/5 (200 messages)
✅ Messages sync completed
```

#### **Terminal 2 - Bridge Logs:**

```bash
cd /Users/yotam/projects/tennex/deployments/local
docker-compose logs -f bridge
```

You'll see:

```
🎉 QR scan successful!
📱 WhatsApp JID: <phone>@s.whatsapp.net
✅ User integration created: ID=<id>
🔗 WhatsApp Connected!
🔄 History Sync: conversations=<count>
✅ Synced conversations and messages
```

#### **Terminal 3 - Frontend (Electron):**

Already running! Check the Electron app window.

---

## 🔥 Hot Reloading Active

### Backend & Bridge (Docker):

- Edit any `.go` file in `services/backend/` or `services/bridge/`
- Air will automatically rebuild and restart
- Changes take ~2-5 seconds

### Frontend (Electron):

- Edit any `.ts`, `.tsx` file in `clients/electron/src/`
- Vite will hot-reload instantly
- Changes appear immediately (usually < 1 second)

---

## 🧪 Quick Test Flow

### Step 1: Create Account

- Open the Electron app (should open automatically)
- Click **Sign Up**
- Create a new account

### Step 2: Connect WhatsApp

- Navigate to **Settings** or **Integrations**
- Click **Connect WhatsApp**
- Scan QR code with your phone

### Step 3: See Your Data

- Messages should start appearing in the frontend
- Check the backend logs to see sync progress
- Query the database to see raw data

### Step 4: Test Real-Time

- Send a message on WhatsApp (from your phone)
- Watch it appear in the Electron app instantly! 🚀

---

## 🗄️ Database Queries

### Connect to PostgreSQL:

```bash
docker exec -it tennex-postgres psql -U tennex -d tennex
```

### Useful Queries:

```sql
-- See all users
SELECT * FROM users;

-- See user integrations (WhatsApp connections)
SELECT * FROM user_integrations;

-- See conversations
SELECT id, platform_id, name, type, unread_count, last_message_at
FROM conversations
ORDER BY last_message_at DESC
LIMIT 10;

-- See messages
SELECT id, conversation_id, content, message_type, is_from_me, timestamp
FROM messages
ORDER BY timestamp DESC
LIMIT 20;

-- Count synced data
SELECT
  (SELECT COUNT(*) FROM conversations) as conversations,
  (SELECT COUNT(*) FROM messages) as messages,
  (SELECT COUNT(*) FROM contacts) as contacts;
```

---

## 🛠️ Development Workflow

### Making Changes:

**Backend Changes:**

```bash
# Edit files in services/backend/
# Watch logs: docker-compose logs -f backend
# Air will rebuild automatically
```

**Bridge Changes:**

```bash
# Edit files in services/bridge/
# Watch logs: docker-compose logs -f bridge
# Air will rebuild automatically
```

**Frontend Changes:**

```bash
# Edit files in clients/electron/src/
# Changes hot-reload instantly in the Electron window
```

**Database Schema Changes:**

```bash
# 1. Create new migration in pkg/db/schema/
# 2. Run: source deployments/local/shell_shortcuts.sh
# 3. Run: txmigrateall
# 4. Run: txsqlc  # Regenerate Go code
```

**Protobuf Changes:**

```bash
# 1. Edit shared/proto/*.proto
# 2. Run: cd shared && buf generate
# 3. Rebuild: docker-compose build backend bridge
```

---

## 🐛 Troubleshooting

### Frontend Won't Connect?

**Check backend is running:**

```bash
curl http://localhost:8000/health
```

Should return: `{"status":"ok",...}`

**Check frontend config:**

- Default: `http://localhost:8000` (in `src/global-config.ts`)

### WhatsApp Won't Connect?

**Check bridge is running:**

```bash
curl http://localhost:6003/health
```

**Check bridge logs:**

```bash
docker-compose logs bridge
```

### Database Issues?

**Check PostgreSQL:**

```bash
docker-compose logs postgres
docker exec -it tennex-postgres psql -U tennex -d tennex -c "\dt"
```

### Electron Won't Start?

**Try:**

```bash
cd /Users/yotam/projects/tennex/clients/electron
rm -rf node_modules
npm install
npm start
```

---

## 🎬 Service Management

### View All Services:

```bash
cd /Users/yotam/projects/tennex/deployments/local
docker-compose ps
```

### Restart a Service:

```bash
docker-compose restart backend
docker-compose restart bridge
```

### Stop All Services:

```bash
docker-compose down
```

### Start All Services:

```bash
docker-compose --profile full up -d
```

### View Logs (All):

```bash
docker-compose logs -f
```

---

## 📊 Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Electron Frontend                         │
│                 (Vite + React + TypeScript)                  │
│                   http://localhost:XXXX                       │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP/REST + JWT
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    Backend Service                           │
│               (Go + Chi + gRPC + PostgreSQL)                 │
│          HTTP: localhost:8000 | gRPC: localhost:6001         │
│                   [Air Hot Reload]                           │
└──────┬──────────────────────────────────┬───────────────────┘
       │                                   │
       │ PostgreSQL                        │ gRPC
       ▼                                   ▼
┌──────────────┐                  ┌────────────────────────────┐
│  PostgreSQL  │                  │     Bridge Service         │
│   Database   │                  │  (Go + whatsmeow + gRPC)   │
│ localhost:   │                  │    http://localhost:6003   │
│    5432      │                  │     [Air Hot Reload]       │
└──────────────┘                  └──────┬─────────────────────┘
                                         │
                                         │ WhatsApp Protocol
                                         ▼
                                  ┌────────────────┐
                                  │   WhatsApp     │
                                  │    Servers     │
                                  └────────────────┘
```

---

## 🎉 You're All Set!

Everything is running and connected. Now:

1. **Open the Electron app** (should be opening)
2. **Create an account** or **login**
3. **Connect WhatsApp** and scan the QR code
4. **Watch your messages sync**!

Happy coding! 🚀

---

## 📝 Quick Reference

**Backend API:** http://localhost:8000
**Backend gRPC:** localhost:6001
**Bridge API:** http://localhost:6003
**Database:** localhost:5432 (user: tennex, pass: tennex123)
**PgAdmin:** http://localhost:8080 (admin@tennex.com / admin123)

**Logs:**

- `docker-compose logs -f backend`
- `docker-compose logs -f bridge`

**Stop Everything:**

- `docker-compose down`
- Close Electron app window

**Restart:**

- `docker-compose --profile full up -d`
- `cd clients/electron && npm start`
