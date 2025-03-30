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

