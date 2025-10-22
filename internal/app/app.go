package app

import (
	"fmt"
	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/iamvkosarev/chatgpt-telegram-bot/config"
	key_value "github.com/iamvkosarev/chatgpt-telegram-bot/internal/storage/key-value"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/usecase"
	"github.com/redis/go-redis/v9"
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

	rdb := redis.NewClient(
		&redis.Options{
			Addr: cfg.Redis.Endpoint,
		},
	)

	openAIUsecase := usecase.NewOpenAIUsecase(cfg.OpenAI)

	userStorage := key_value.NewUserStorage(rdb)

	userUsecase := usecase.NewUserUsecase(
		usecase.UserUsecaseDeps{
			UserStorage: userStorage,
		},
		cfg.Telegram,
	)

	aiChatStorage := key_value.NewAIChatStorage(rdb)

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
