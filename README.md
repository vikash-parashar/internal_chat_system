# 🏥 Internal Chat System (Go, PostgreSQL, Redis, WebSocket)

A real-time internal messaging system built for EHR platforms using Golang, PostgreSQL, Redis, and WebSockets. It allows secure communication between doctors (`user_id`) and patients (`contact_id`) scoped to a hospital or clinic (`location_id`).

---

## 🛠 Tech Stack

- **Backend:** Golang (chi router)
- **Database:** PostgreSQL
- **Real-time:** WebSockets
- **Message Broker:** Redis (Pub/Sub)
- **Auth:** JWT (via middleware)
- **Frontend:** Test with HTML/JS or Postman

---

## 📦 Features

- Real-time messaging via WebSocket
- Message persistence in PostgreSQL
- Redis Pub/Sub for scalability
- JWT-based authentication middleware
- Read receipts and delivery tracking
- Modular, production-grade architecture

---

## 📂 API Endpoints

### 🔐 Requires `Authorization: Bearer <token>`

#### 1. Send a Message
```
POST /chat/send
```
**Payload:**
```json
{
  "location_id": "loc1",
  "sender_user_id": "doc123",
  "receiver_contact_id": "pat456",
  "content": "Hello patient!"
}
```

#### 2. Get Message History
```
GET /chat/history?location_id=loc1&user_id=doc123&contact_id=pat456
```

#### 3. Mark Messages as Read
```
PUT /chat/read
```
**Payload:**
```json
{
  "message_ids": ["uuid1", "uuid2"]
}
```

#### 4. WebSocket Endpoint
```
ws://localhost:8080/ws?location_id=loc1&user_id=doc123&contact_id=pat456
```

---

## 🧪 Testing Instructions

### ✅ Postman
- Import the `internal-chat.postman_collection.json`
- Set `jwt_token` variable in environment

### 🌐 WebSocket Test (Smart WebSocket Client / HTML)
- Connect to `ws://localhost:8080/ws?...`
- Send messages from REST API and observe real-time chat

### 🧰 Generate JWTs
Use `jwt.io` or a Go script with the same signing key used in `jwt.go`

---

## 🗄 Database Schema (PostgreSQL)
```sql
CREATE TABLE messages (
    id UUID PRIMARY KEY,
    location_id UUID NOT NULL,
    sender_user_id UUID,
    receiver_user_id UUID,
    sender_contact_id UUID,
    receiver_contact_id UUID,
    content TEXT NOT NULL,
    sent_at TIMESTAMP DEFAULT now(),
    read_at TIMESTAMP,
    is_read BOOLEAN DEFAULT FALSE
);
```

---

## 📜 License
MIT License

---

## 👨‍⚕️ Built by Pros, for Healthcare Teams
Production-ready chat infrastructure with compliance, security, and extensibility at its core.














# 📬 Internal Chat System for EHR Platform

A HIPAA-compliant real-time chat system for secure communication between patients and doctors within a specific hospital (location).

---

## ✅ Features Implemented

### 💬 Messaging Core
- Two-way real-time messaging using WebSockets (doctor ↔ patient)
- Message storage in PostgreSQL with UUIDs
- Chat session management (per contact + doctor + location)
- Support for:
  - Text messages
  - File messages (PDFs, images, etc.)
  - Replies / threaded messages
  - Reactions (❤️, ✅, ❗)
  - Message editing
  - Pin/unpin messages
  - Message deletion (soft-delete)

### 🔄 Realtime & Offline Support
- Redis Pub/Sub for scalable real-time messaging
- WebSocket connection registry (hub)
- Offline message queue using Redis lists
- Delivery + read tracking (with timestamps)
- Typing indicators
- Online/last seen presence tracking

### 🔔 Push Notifications
- Firebase Cloud Messaging (FCM) integration
- Device token management with upsert support
- Send push notifications to offline users

### 🔍 Search & Filtering
- Message search by content (ILIKE)
- Pagination support for chat sessions
- Filter sessions by location

### 🧑 Auth & Access Control
- JWT middleware with user context
- Role-based permissions (DOCTOR / PATIENT)
- Session creation/reuse logic

### 📁 File Support
- Support for file previews in messages
- File message type metadata (`message_type: file`)
- Validation of file types (TBD)

---

## 🚧 To Be Implemented

### 🔐 Auth Enhancements
- [ ] User impersonation support for Admins
- [ ] Refresh token flow

### 📥 Upload & File Controls
- [ ] Restrict allowed file types
- [ ] S3-backed upload support for large files
- [ ] File virus scanning (ClamAV integration)

### 📊 Admin Tools & Analytics
- [ ] Admin dashboard for chat usage (total messages, active users)
- [ ] Message moderation tools
- [ ] Archived session viewer

### 🔁 Additional Message Features
- [ ] Forwarding messages to other users
- [ ] Mark conversation as "resolved"
- [ ] Conversation tags/labels

### 🔍 Advanced Search & Filters
- [ ] Filter by unread / pinned / attachments
- [ ] Sort messages by relevance / timestamp
- [ ] Date range filtering

### 🛎 Push Notifications (Next Phase)
- [ ] Push to topic/group
- [ ] Silent push for badge updates
- [ ] Push preview (message + sender name)

### 🧪 QA / Testing
- [ ] WebSocket disconnect reconnect logic
- [ ] Mobile responsiveness (if embedded)
- [ ] Rate limiting (per user)

---

## ✨ Prompts for AI-Assisted Implementation

Use these prompts to accelerate development using ChatGPT:

### 🔐 Auth
> "Generate a Go middleware for JWT validation with support for roles: ADMIN, DOCTOR, PATIENT."

### 📥 Upload
> "Create an S3-backed file upload API with MIME type validation and size limit."

### 🔁 Message Forwarding
> "Add support for forwarding a message to another session or user in Golang."

### 📊 Analytics
> "Build a Go API to aggregate message count by day for the past 30 days grouped by location."

### 🔍 Search
> "Extend message search to support date range and message type filters in SQL."

### 🔔 Push Notifications
> "Send push notification to an FCM topic from Go when a message is sent to a group."

### 🧪 Testing
> "Write Go tests to simulate 10 WebSocket clients sending messages concurrently."

---

## 🧠 Contributors
- Backend Lead: You 😎
- Frontend: Handled via Next.js (Craft.js)
- Push Infra: Firebase Cloud Messaging
- Storage: PostgreSQL, Redis, AWS S3

---

Let me know when you're ready to build the Admin tools, silent push notifications, or WebSocket reconnection handler!

