// Package mail provides JSONL-based messaging for agent communication.
package mail

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Priority levels for messages.
type Priority string

const (
	// PriorityNormal is the default priority.
	PriorityNormal Priority = "normal"

	// PriorityHigh indicates an urgent message.
	PriorityHigh Priority = "high"
)

// Message represents a mail message between agents.
type Message struct {
	// ID is a unique message identifier.
	ID string `json:"id"`

	// From is the sender address (e.g., "gastown/Toast" or "mayor/").
	From string `json:"from"`

	// To is the recipient address.
	To string `json:"to"`

	// Subject is a brief summary.
	Subject string `json:"subject"`

	// Body is the full message content.
	Body string `json:"body"`

	// Timestamp is when the message was sent.
	Timestamp time.Time `json:"timestamp"`

	// Read indicates if the message has been read.
	Read bool `json:"read"`

	// Priority is the message priority.
	Priority Priority `json:"priority"`
}

// NewMessage creates a new message with a generated ID.
func NewMessage(from, to, subject, body string) *Message {
	return &Message{
		ID:        generateID(),
		From:      from,
		To:        to,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now(),
		Read:      false,
		Priority:  PriorityNormal,
	}
}

// generateID creates a random message ID.
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "msg-" + hex.EncodeToString(b)
}
