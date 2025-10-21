package app

import (
	"fmt"
	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/iamvkosarev/chatgpt-telegram-bot/config"
	in_memory "github.com/iamvkosarev/chatgpt-telegram-bot/internal/storage/in-memory"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/usecase"
	"log"
	"net/url"
)

func Run(cfg *config.Config) error {
	baseURL, err := url.JoinPath(cfg.OpenAI.OpenAIBaseURL, "/v1")
	if err != nil {
		return err
	}
	cfg.OpenAI.OpenAIBaseURL = baseURL

	bot, err := api.NewBotAPI(cfg.Telegram.TelegramAPIToken)
	if err != nil {
		return fmt.Errorf("failed to create new bot: %w", err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	openAIUsecase := usecase.NewOpenAIUsecase(cfg.OpenAI)

	userStorage := in_memory.NewUserStorage()

	userUsecase := usecase.NewUserUsecase(
		usecase.UserUsecaseDeps{
			UserStorage: userStorage,
		},
		cfg.Telegram,
	)

	aiChatStorage := in_memory.NewAIChatStorage()

	aiChatUsecase := usecase.NewAiChatUsecase(
		usecase.AiChatUsecaseDeps{
			AiChatStorage: aiChatStorage,
			User:          userUsecase,
		}, cfg.AIChat,
	)

	telegramUsecase, err := usecase.NewTelegramUsecase(
		cfg.Telegram, usecase.TelegramUsecaseDeps{
			User:   userUsecase,
			Bot:    bot,
			OpenAI: openAIUsecase,
			AIChat: aiChatUsecase,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create telegram usecase: %w", err)
	}

	return telegramUsecase.Run()
}
