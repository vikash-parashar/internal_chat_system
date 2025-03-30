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

-- Index for querying unread messages
CREATE INDEX idx_chat_unread ON messages (receiver_user_id, is_read);
