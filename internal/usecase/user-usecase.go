package usecase

import (
	"github.com/google/uuid"
	"github.com/iamvkosarev/chatgpt-telegram-bot/config"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/model"
)

type UserStorage interface {
	GetUserIDForTelegramUser(userTelegramID int64) (uuid.UUID, error)
	CreateNewTelegramUser(userTelegramID int64, roles []model.UserRole) (uuid.UUID, error)
	GetUserInfo(userID uuid.UUID) (model.User, error)
	UpdateUserLastAIChat(userID uuid.UUID, aiChatID uuid.UUID) error
}

type UserUsecaseDeps struct {
	UserStorage UserStorage
}

type UserUsecase struct {
	UserUsecaseDeps
	telegramCfg config.Telegram
}

func NewUserUsecase(deps UserUsecaseDeps, telegramCfg config.Telegram) *UserUsecase {
	return &UserUsecase{
		UserUsecaseDeps: deps,
		telegramCfg:     telegramCfg,
	}
}

func (u *UserUsecase) GetUserInfoForTelegramUser(userTelegramID int64) (model.User, error) {
	userID, err := u.UserStorage.GetUserIDForTelegramUser(userTelegramID)
	if err != nil {
		userID, err = u.UserStorage.CreateNewTelegramUser(userTelegramID, u.getTelegramUserRoles(userTelegramID))
	}
	return u.UserStorage.GetUserInfo(userID)
}

func (u *UserUsecase) GetUserInfo(userID uuid.UUID) (model.User, error) {
	user, err := u.UserStorage.GetUserInfo(userID)
	if err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (u *UserUsecase) UpdateUserLastAIChat(userID, aiChatID uuid.UUID) error {
	return u.UserStorage.UpdateUserLastAIChat(userID, aiChatID)
}

func (u *UserUsecase) getTelegramUserRoles(userTelegramID int64) []model.UserRole {
	roles := []model.UserRole{
		model.UserRoleDefault,
	}
	for _, userWithRoleID := range u.telegramCfg.AdminTelegramIDList {
		if userWithRoleID == userTelegramID {
			roles = append(roles, model.UserRoleAdmin)
			break
		}
	}
	for _, userWithRoleID := range u.telegramCfg.PremiumTelegramIDList {
		if userWithRoleID == userTelegramID {
			roles = append(roles, model.UserRolePremium)
			break
		}
	}
	return roles
}
