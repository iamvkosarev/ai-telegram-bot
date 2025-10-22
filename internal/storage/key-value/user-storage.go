package key_value

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/model"
	"github.com/redis/go-redis/v9"
)

var (
	ErrUserAlreadyExists = errors.New("userInternal already exists")
	ErrUserDoesNotExists = errors.New("userInternal doesn't exists")
)

type userInternal struct {
	UserID     string           `json:"user_id"`
	TelegramID int64            `json:"telegram_id"`
	Roles      []model.UserRole `json:"roles"`
	LastAIChat string           `json:"last_ai_chat"`
}

type UserStorage struct {
	rdb *redis.Client
}

func NewUserStorage(rdb *redis.Client) *UserStorage {
	return &UserStorage{
		rdb: rdb,
	}
}

func (u *UserStorage) CreateNewTelegramUser(
	ctx context.Context,
	userTelegramID int64,
	roles []model.UserRole,
) (uuid.UUID, error) {
	userTelegramIDKey := getUserTelegramIDKey(userTelegramID)
	_, err := u.rdb.Get(ctx, userTelegramIDKey).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return uuid.Nil, fmt.Errorf("failed to get userInternal %s: %w", userTelegramIDKey, err)
		}
	} else {
		return uuid.Nil, ErrUserAlreadyExists
	}
	userID := uuid.New()
	if err = u.rdb.Set(ctx, userTelegramIDKey, userID.String(), 0).Err(); err != nil {
		return uuid.Nil, fmt.Errorf("failed to save userInternal %s: %w", userID, err)
	}

	user := userInternal{
		TelegramID: userTelegramID,
		UserID:     userID.String(),
		Roles:      roles,
	}
	err = u.setUser(ctx, userID, user)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to set user: %w", err)
	}
	return userID, nil
}

func (u *UserStorage) UpdateUserLastAIChat(ctx context.Context, userID uuid.UUID, aiChatID uuid.UUID) error {
	user, err := u.getUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	user.LastAIChat = aiChatID.String()
	if err = u.setUser(ctx, userID, user); err != nil {
		return fmt.Errorf("failed to set user: %w", err)
	}
	return nil
}

func (u *UserStorage) GetUserInfo(ctx context.Context, userID uuid.UUID) (model.User, error) {
	userInt, err := u.getUser(ctx, userID)
	if err != nil {
		return model.User{}, fmt.Errorf("failed to get user: %w", err)
	}
	lastAIChat, err := uuid.Parse(userInt.LastAIChat)
	if err != nil {
		return model.User{}, fmt.Errorf("failed to parse userID %s: %w", userID, err)
	}

	user := model.User{
		TelegramID: userInt.TelegramID,
		UserID:     userID,
		LastAIChat: lastAIChat,
		Roles:      userInt.Roles,
	}
	return user, nil
}

func (u *UserStorage) GetUserIDForTelegramUser(ctx context.Context, userTelegramID int64) (uuid.UUID, error) {
	userTelegramIDKey := getUserTelegramIDKey(userTelegramID)
	userIDStr, err := u.rdb.Get(ctx, userTelegramIDKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return uuid.Nil, model.ErrTelegramUserDoesNotExists
		} else {
			return uuid.Nil, fmt.Errorf("failed to get telegram user id %s: %w", userTelegramIDKey, err)
		}
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse userID %s: %w", userIDStr, err)
	}
	return userID, nil
}

func (u *UserStorage) getUser(ctx context.Context, userID uuid.UUID) (userInternal, error) {
	userIDKey := getUserIDKey(userID)
	userRaw, err := u.rdb.Get(ctx, userIDKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return userInternal{}, ErrUserDoesNotExists
		} else {
			return userInternal{}, fmt.Errorf("failed to get userInternal %s: %w", userID, err)
		}
	}
	var user userInternal
	if err = json.Unmarshal([]byte(userRaw), &user); err != nil {
		return userInternal{}, fmt.Errorf("failed to unmarshal userInternal %s: %w", userID, err)
	}
	return user, nil
}

func (u *UserStorage) setUser(
	ctx context.Context, userID uuid.UUID, userInt userInternal,
) error {
	userIDKey := getUserIDKey(userID)
	newUserJSON, err := json.Marshal(userInt)
	if err != nil {
		return fmt.Errorf("failed to marshal internal user: %w", err)
	}
	if err = u.rdb.Set(ctx, userIDKey, newUserJSON, 0).Err(); err != nil {
		return fmt.Errorf("failed to save userInternal %s: %w", userID, err)
	}
	return nil
}

func getUserTelegramIDKey(id int64) string {
	return fmt.Sprintf("telegram_%d", id)
}

func getUserIDKey(id uuid.UUID) string {
	return fmt.Sprintf("user_%d", id)
}
