package in_memory

import (
	"errors"
	"github.com/google/uuid"
	"github.com/iamvkosarev/ai-telegram-bot/internal/model"
)

var (
	ErrChatDoesNotExist = errors.New("chat does not exist")
)

type AIChatStorage struct {
	chats map[uuid.UUID]*model.AIChat
}

func NewAIChatStorage() *AIChatStorage {
	return &AIChatStorage{
		chats: make(map[uuid.UUID]*model.AIChat),
	}
}

func (a *AIChatStorage) ListUserChats(userID uuid.UUID) ([]model.AIChat, error) {
	chats := make([]model.AIChat, 0)
	for _, chat := range a.chats {
		if chat.UserID == userID {
			chats = append(chats, *chat)
		}
	}
	return chats, nil
}

func (a *AIChatStorage) CreateChat(userID uuid.UUID, chatModel string) (model.AIChat, error) {
	chatID := uuid.New()
	chat := model.AIChat{
		UserID:           userID,
		ChatID:           chatID,
		Model:            chatModel,
		Messages:         make([]model.Message, 0),
		ModelTemperature: 1,
	}
	a.chats[chatID] = &chat
	return chat, nil
}

func (a *AIChatStorage) GetChat(chatID uuid.UUID) (model.AIChat, error) {
	chat, ok := a.chats[chatID]
	if !ok {
		return model.AIChat{}, ErrChatDoesNotExist
	}
	return *chat, nil
}

func (a *AIChatStorage) AddMessageToChat(
	chatID uuid.UUID,
	messageText string,
	messageSource model.MessageSource,
) error {
	chat, ok := a.chats[chatID]
	if !ok {
		return ErrChatDoesNotExist
	}
	chat.Messages = append(
		chat.Messages, model.Message{
			Source: messageSource,
			Body:   messageText,
		},
	)
	a.chats[chatID] = chat
	return nil
}
