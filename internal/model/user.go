package model

import (
	"github.com/google/uuid"
)

type User struct {
	UserID     uuid.UUID
	TelegramID int64
	Roles      []UserRole
	LastAIChat uuid.UUID
}
