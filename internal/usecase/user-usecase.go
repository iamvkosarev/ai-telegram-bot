package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/iamvkosarev/chatgpt-telegram-bot/config"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/model"
)

type UserStorage interface {
	GetUserIDForTelegramUser(ctx context.Context, userTelegramID int64) (uuid.UUID, error)
	CreateNewTelegramUser(ctx context.Context, userTelegramID int64, roles []model.UserRole) (uuid.UUID, error)
	GetUserInfo(ctx context.Context, userID uuid.UUID) (model.User, error)
	UpdateUserLastAIChat(ctx context.Context, userID uuid.UUID, aiChatID uuid.UUID) error
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

func (u *UserUsecase) GetUserInfoForTelegramUser(ctx context.Context, userTelegramID int64) (model.User, error) {
	userID, err := u.UserStorage.GetUserIDForTelegramUser(ctx, userTelegramID)
	if err != nil {
		if errors.Is(err, model.ErrTelegramUserDoesNotExists) {
			userID, err = u.UserStorage.CreateNewTelegramUser(
				ctx, userTelegramID, u.getTelegramUserRoles(userTelegramID),
			)
			if err != nil {
				return model.User{}, fmt.Errorf("failed to create telegram user: %w", err)
			}
		} else {
			return model.User{}, fmt.Errorf("failed to get telegram user: %w", err)
		}
	}
	return u.UserStorage.GetUserInfo(ctx, userID)
}

func (u *UserUsecase) GetUserInfo(ctx context.Context, userID uuid.UUID) (model.User, error) {
	user, err := u.UserStorage.GetUserInfo(ctx, userID)
	if err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (u *UserUsecase) UpdateUserLastAIChat(ctx context.Context, userID, aiChatID uuid.UUID) error {
	return u.UserStorage.UpdateUserLastAIChat(ctx, userID, aiChatID)
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
