package in_memory

import (
	"errors"
	"github.com/google/uuid"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/model"
)

var (
	ErrUserAlreadyExists         = errors.New("user already exists")
	ErrTelegramUserDoesNotExists = errors.New("telegram user doesn't exists")
	ErrUserDoesNotExists         = errors.New("user doesn't exists")
)

type UserStorage struct {
	users            map[uuid.UUID]*model.User
	telegramUsersIDs map[int64]uuid.UUID
}

func NewUserStorage() *UserStorage {
	return &UserStorage{
		users:            make(map[uuid.UUID]*model.User),
		telegramUsersIDs: make(map[int64]uuid.UUID),
	}
}

func (u *UserStorage) CreateNewTelegramUser(userTelegramID int64, roles []model.UserRole) (uuid.UUID, error) {
	if _, ok := u.telegramUsersIDs[userTelegramID]; ok {
		return uuid.Nil, ErrUserAlreadyExists
	}
	userID := uuid.New()
	u.telegramUsersIDs[userTelegramID] = userID

	newUserRoles := []model.UserRole{
		model.UserRoleDefault,
	}
	if roles != nil && len(roles) > 0 {
		newUserRoles = append(newUserRoles, roles...)
	}
	user := &model.User{
		TelegramID: userTelegramID,
		UserID:     userID,
		Roles:      newUserRoles,
	}
	u.users[userID] = user
	return userID, nil
}

func (u *UserStorage) UpdateUserLastAIChat(userID uuid.UUID, aiChatID uuid.UUID) error {
	user, ok := u.users[userID]
	if !ok {
		return ErrUserDoesNotExists
	}
	user.LastAIChat = aiChatID
	return nil
}

func (u *UserStorage) GetUserInfo(userID uuid.UUID) (model.User, error) {
	user, ok := u.users[userID]
	if !ok {
		return model.User{}, ErrUserDoesNotExists
	}
	return *user, nil
}

func (u *UserStorage) GetUserIDForTelegramUser(userTelegramID int64) (uuid.UUID, error) {
	userID, ok := u.telegramUsersIDs[userTelegramID]
	if !ok {
		return uuid.Nil, ErrTelegramUserDoesNotExists
	}
	return userID, nil
}
