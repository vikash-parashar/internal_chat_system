-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    location_id UUID NOT NULL,

    sender_user_id UUID,
    receiver_user_id UUID,

    sender_contact_id UUID,
    receiver_contact_id UUID,

    content TEXT NOT NULL,
    
    sent_at TIMESTAMP NOT NULL DEFAULT now(),
    read_at TIMESTAMP,
    is_read BOOLEAN DEFAULT FALSE
);


-- Index for retrieving conversations quickly
CREATE INDEX idx_chat_conversation ON messages (
    location_id,
    sender_user_id,
    receiver_user_id,
    sender_contact_id,
    receiver_contact_id
);

CREATE TABLE chat_sessions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  contact_id UUID NOT NULL,
  user_id UUID NOT NULL,
  location_id UUID NOT NULL,
  started_at TIMESTAMP DEFAULT now(),
  last_message_at TIMESTAMP,
  UNIQUE(contact_id, user_id, location_id)
);


-- Unique constraint to avoid duplicate sessions
CREATE UNIQUE INDEX uniq_chat_session ON chat_sessions (contact_id, user_id, location_id);

ALTER TABLE messages ADD COLUMN session_id UUID REFERENCES chat_sessions(id);



ALTER TABLE messages ADD COLUMN delivered_at TIMESTAMP;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_messages_content_trgm ON messages USING gin (content gin_trgm_ops);
