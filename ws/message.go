package ws

type WebSocketPayload struct {
	Type      string `json:"type"` // e.g., "message", "typing"
	SessionID string `json:"session_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Typing    bool   `json:"typing,omitempty"`
}
