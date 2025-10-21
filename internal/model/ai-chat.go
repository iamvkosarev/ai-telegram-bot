package model

import "github.com/google/uuid"

type MessageSource string

const (
	MessageSourceUser      = MessageSource("user")
	MessageSourceAssistant = MessageSource("assistant")
)

type Message struct {
	Source MessageSource
	Body   string
}

type AIChat struct {
	ChatID           uuid.UUID
	UserID           uuid.UUID
	Messages         []Message
	Model            string
	ModelTemperature float32
}
