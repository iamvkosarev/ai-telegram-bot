package key_value

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/iamvkosarev/ai-telegram-bot/internal/model"
	"github.com/redis/go-redis/v9"
)

var (
	ErrChatDoesNotExist       = errors.New("chat does not exist")
	ErrUserChatsIDsDoNotExist = errors.New("user chat ids does not exist")
)

type messageInternal struct {
	Source model.MessageSource `json:"source"`
	Body   string              `json:"body"`
}

type chatInternal struct {
	ChatID           string            `json:"chat_id"`
	UserID           string            `json:"user_id"`
	Messages         []messageInternal `json:"messages"`
	Model            string            `json:"model"`
	ModelTemperature float32           `json:"model_temperature"`
}

type userChatsIDs struct {
	Chats []string `json:"chats"`
}

type AIChatStorage struct {
	rdb *redis.Client
}

func NewAIChatStorage(rdb *redis.Client) *AIChatStorage {
	return &AIChatStorage{
		rdb: rdb,
	}
}

func (a *AIChatStorage) CreateChat(
	ctx context.Context, userID uuid.UUID, chatModel string,
	temperature float32,
) (
	model.AIChat,
	error,
) {
	chatID := uuid.New()
	chatInt := chatInternal{
		UserID:           userID.String(),
		ChatID:           chatID.String(),
		Model:            chatModel,
		Messages:         make([]messageInternal, 0),
		ModelTemperature: temperature,
	}

	if err := a.setChatInt(ctx, chatID, chatInt); err != nil {
		return model.AIChat{}, fmt.Errorf("failed to set chat internal %s: %w", chatID.String(), err)
	}
	userChatsIDsInt, err := a.getUserChatsIDs(ctx, userID)
	if err != nil {
		if !errors.Is(err, ErrUserChatsIDsDoNotExist) {
			return model.AIChat{}, fmt.Errorf("failed to get user chats ids: %w", err)
		}
		userChatsIDsInt = userChatsIDs{
			Chats: make([]string, 0),
		}
	}
	userChatsIDsInt.Chats = append(userChatsIDsInt.Chats, chatID.String())
	if err = a.setUserChatsIDs(ctx, userID, userChatsIDsInt); err != nil {
		return model.AIChat{}, fmt.Errorf("failed to set user chats ids: %w", err)
	}

	chat := model.AIChat{
		ChatID:           chatID,
		UserID:           userID,
		Model:            chatModel,
		Messages:         make([]model.Message, 0),
		ModelTemperature: temperature,
	}
	return chat, nil
}

func (a *AIChatStorage) ListUserChats(ctx context.Context, userID uuid.UUID) ([]model.AIChat, error) {
	userChatsIDsInt, err := a.getUserChatsIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user chats ids: %w", err)
	}
	chats := make([]model.AIChat, 0, len(userChatsIDsInt.Chats))
	for _, chatIDStr := range userChatsIDsInt.Chats {
		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse chatID %s: %w", chatIDStr, err)
		}
		chat, err := a.GetChat(ctx, chatID)
		chats = append(chats, chat)
	}
	return chats, nil
}

func (a *AIChatStorage) GetChat(ctx context.Context, chatID uuid.UUID) (model.AIChat, error) {
	chatInt, err := a.getChatInt(ctx, chatID)
	if err != nil {
		return model.AIChat{}, fmt.Errorf("failed to get chat %s: %w", chatID, err)
	}
	userID, err := uuid.Parse(chatInt.UserID)
	if err != nil {
		return model.AIChat{}, fmt.Errorf("failed to parse chat %s: %w", chatID, err)
	}

	messages := make([]model.Message, 0, len(chatInt.Messages))
	for _, msg := range chatInt.Messages {
		messages = append(
			messages, model.Message{
				Source: msg.Source,
				Body:   msg.Body,
			},
		)
	}

	chat := model.AIChat{
		ChatID:           chatID,
		UserID:           userID,
		Model:            chatInt.Model,
		ModelTemperature: chatInt.ModelTemperature,
		Messages:         messages,
	}
	return chat, nil
}

func (a *AIChatStorage) AddMessageToChat(
	ctx context.Context,
	chatID uuid.UUID,
	messageText string,
	messageSource model.MessageSource,
) error {
	chatInt, err := a.getChatInt(ctx, chatID)
	if err != nil {
		return err
	}
	chatInt.Messages = append(
		chatInt.Messages, messageInternal{
			Source: messageSource,
			Body:   messageText,
		},
	)
	err = a.setChatInt(ctx, chatID, chatInt)
	if err != nil {
		return fmt.Errorf("failed to set internal chat %s: %w", chatID.String(), err)
	}

	return nil
}

func (a *AIChatStorage) getChatInt(ctx context.Context, chatID uuid.UUID) (chatInternal, error) {
	chatIDKey := getChatIDKey(chatID)
	chatIntRaw, err := a.rdb.Get(ctx, chatIDKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return chatInternal{}, ErrChatDoesNotExist
		} else {
			return chatInternal{}, fmt.Errorf("failed to get chat %s: %w", chatID, err)
		}
	}
	var chatInt chatInternal
	if err = json.Unmarshal([]byte(chatIntRaw), &chatInt); err != nil {
		return chatInternal{}, fmt.Errorf("failed to unmarshal chat %s: %w", chatID, err)
	}
	return chatInt, nil
}

func (a *AIChatStorage) setChatInt(ctx context.Context, chatID uuid.UUID, chatInt chatInternal) error {
	chatIDKey := getChatIDKey(chatID)
	chatIntJSON, err := json.Marshal(chatInt)
	if err != nil {
		return fmt.Errorf("failed to marshal internal chat: %w", err)
	}
	if err = a.rdb.Set(ctx, chatIDKey, chatIntJSON, 0).Err(); err != nil {
		return fmt.Errorf("failed to save chatInternal %s: %w", chatIDKey, err)
	}
	return nil
}

func (a *AIChatStorage) getUserChatsIDs(ctx context.Context, userID uuid.UUID) (userChatsIDs, error) {
	userChatsKey := getUserChatsKey(userID)
	userChatsRaw, err := a.rdb.Get(ctx, userChatsKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return userChatsIDs{}, ErrUserChatsIDsDoNotExist
		} else {
			return userChatsIDs{}, fmt.Errorf("failed to get userChatsIDs %s: %w", userID, err)
		}
	}
	var userChats userChatsIDs
	if err = json.Unmarshal([]byte(userChatsRaw), &userChats); err != nil {
		return userChatsIDs{}, fmt.Errorf("failed to unmarshal userChatsIDs %s: %w", userID, err)
	}
	return userChats, nil
}

func (a *AIChatStorage) setUserChatsIDs(ctx context.Context, userID uuid.UUID, userChatsIDsInt userChatsIDs) error {
	userChatsIDsIntJSON, err := json.Marshal(userChatsIDsInt)
	if err != nil {
		return fmt.Errorf("failed to marshal user chats ids: %w", err)
	}
	userChatsKey := getUserChatsKey(userID)
	if err = a.rdb.Set(ctx, userChatsKey, userChatsIDsIntJSON, 0).Err(); err != nil {
		return fmt.Errorf("failed to save user chats ids %s: %w", userChatsKey, err)
	}
	return nil
}

func getChatIDKey(chatID uuid.UUID) string {
	return fmt.Sprintf("chat_%v", chatID.String())
}

func getUserChatsKey(userID uuid.UUID) string {
	return fmt.Sprintf("user_chats_%v", userID.String())
}
