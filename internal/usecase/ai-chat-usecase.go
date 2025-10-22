package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/iamvkosarev/chatgpt-telegram-bot/config"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/model"
)

var (
	ErrUserRoleHasNotAnyAvailableModels = errors.New("user role has not any available models")
	ErrUserRoleHasNotAccessToModel      = errors.New("user has not access to model")
)

type AiChatStorage interface {
	GetChat(ctx context.Context, chatID uuid.UUID) (model.AIChat, error)
	CreateChat(
		ctx context.Context, userID uuid.UUID, model string,
		temperature float32,
	) (model.AIChat, error)
	AddMessageToChat(ctx context.Context, chatID uuid.UUID, messageText string, messageSource model.MessageSource) error
	ListUserChats(ctx context.Context, userID uuid.UUID) ([]model.AIChat, error)
}

type AiChatUsecaseDeps struct {
	AiChatStorage AiChatStorage
	User          *UserUsecase
}

type AiChatUsecase struct {
	AiChatUsecaseDeps
	cfg                  config.AIChat
	userRoleToChatModels map[model.UserRole][]string
}

func NewAiChatUsecase(deps AiChatUsecaseDeps, cfg config.AIChat) *AiChatUsecase {
	userRoleToChatModels := make(map[model.UserRole][]string)
	for _, roleToModels := range cfg.AccessModelsPerRoles {
		userRoleToChatModels[model.ParseUserRole(roleToModels.Role)] = roleToModels.Models
	}
	return &AiChatUsecase{
		AiChatUsecaseDeps:    deps,
		cfg:                  cfg,
		userRoleToChatModels: userRoleToChatModels,
	}
}

func (a *AiChatUsecase) GetChat(ctx context.Context, chatID uuid.UUID) (model.AIChat, error) {
	return a.AiChatStorage.GetChat(ctx, chatID)
}

func (a *AiChatUsecase) CreateChat(ctx context.Context, userID uuid.UUID, aiModel string) (model.AIChat, error) {
	user, err := a.User.GetUserInfo(ctx, userID)
	if err != nil {
		return model.AIChat{}, fmt.Errorf("failed get user info: %w", err)
	}
	availableModels := a.GetAvailableForUserModels(user)
	if len(availableModels) == 0 {
		return model.AIChat{}, ErrUserRoleHasNotAnyAvailableModels
	}
	if _, ok := availableModels[aiModel]; !ok {
		return model.AIChat{}, ErrUserRoleHasNotAccessToModel
	}
	return a.AiChatStorage.CreateChat(ctx, userID, aiModel, 1)
}

func (a *AiChatUsecase) ListUserChats(ctx context.Context, userID uuid.UUID) ([]model.AIChat, error) {
	return a.AiChatStorage.ListUserChats(ctx, userID)
}

func (a *AiChatUsecase) AddMessageToChat(
	ctx context.Context,
	chatID uuid.UUID,
	messageText string,
	messageSource model.MessageSource,
) error {
	return a.AiChatStorage.AddMessageToChat(ctx, chatID, messageText, messageSource)
}

func (a *AiChatUsecase) GetAvailableForUserModels(user model.User) map[string]struct{} {
	availableModels := make(map[string]struct{})
	for _, role := range user.Roles {
		for _, aiModel := range a.userRoleToChatModels[role] {
			availableModels[aiModel] = struct{}{}
		}
	}
	return availableModels
}
