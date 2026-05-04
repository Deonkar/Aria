package models

import "time"

type Conversation struct {
	ID            string
	UserID        string
	SessionToken  string
	Title         *string
	MessageCount  int
	StartedAt     time.Time
	LastMessageAt *time.Time
	EndedAt       *time.Time
}

type ChatMessage struct {
	ID               string
	ConversationID   string
	UserID           string
	Role             string
	Content          string
	GeneratedSQL     *string
	SQLRowCount      *int
	SQLExecutionMs   *int
	TokenCountInput  *int
	TokenCountOutput *int
	ModelUsed        *string
	WasCached        bool
	WasSQLCorrected  bool
	CreatedAt        time.Time
}
