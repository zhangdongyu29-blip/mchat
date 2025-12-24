package model

import "time"

// Role defines a chat persona.
type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `json:"name"`
	Background  string    `json:"background"`
	Style       string    `json:"style"`
	PersonaHint string    `json:"persona_hint"`
	CallMe      string    `json:"call_me"`
}

// Conversation groups messages under a role.
type Conversation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	RoleID    uint      `json:"role_id"`
	Title     string    `json:"title"`
}

// ChatMessage stores conversation messages.
type ChatMessage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	RoleID         uint      `json:"role_id"`
	ConversationID uint      `json:"conversation_id"`
	Sender         string    `json:"sender"` // user or ai or system
	Content        string    `json:"content"`
}

// Memory keeps summarized context for long chats.
type Memory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	RoleID    uint      `json:"role_id"`
	Summary   string    `json:"summary"`
}
