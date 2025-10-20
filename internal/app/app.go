package app

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/iamvkosarev/chatgpt-telegram-bot/config"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/usecase"
	"github.com/sourcegraph/conc"
	"log"
	"net/url"
	"time"
)

func Run(cfg *config.Config) error {
	baseURL, err := url.JoinPath(cfg.GPT.OpenAIBaseURL, "/v1")
	if err != nil {
		return err
	}
	cfg.GPT.OpenAIBaseURL = baseURL

	if cfg.GPT.OPENAIModel != "gpt-3.5-turbo" && cfg.GPT.OPENAIModel != "gpt-4" {
		return fmt.Errorf("invalid OPENAI_MODEL: %s", cfg.GPT.OPENAIModel)
	}

	bot, err := tgbotapi.NewBotAPI(cfg.Telegram.TelegramAPIToken)
	if err != nil {
		return fmt.Errorf("failed to create new bot: %w", err)
	}

	gptUsecase := usecase.NewGPTUsecase(cfg.GPT)

	log.Printf("Authorized on account %s", bot.Self.UserName)

	_, _ = bot.Request(
		tgbotapi.NewSetMyCommands(
			[]tgbotapi.BotCommand{
				{
					Command:     "help",
					Description: "Get help",
				},
				{
					Command:     "new",
					Description: "Clear context and start a new conversation",
				},
			}...,
		),
	)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		if len(cfg.Telegram.AllowedTelegramID) != 0 {
			var userAllowed bool
			for _, allowedID := range cfg.Telegram.AllowedTelegramID {
				if allowedID == update.Message.Chat.ID {
					userAllowed = true
				}
			}
			if !userAllowed {
				_, err := bot.Send(
					tgbotapi.NewMessage(
						update.Message.Chat.ID,
						fmt.Sprintf("You are not allowed to use this bot. User ID: %d", update.Message.Chat.ID),
					),
				)
				if err != nil {
					log.Print(err)
				}
				continue
			}
		}

		if update.Message.IsCommand() { // ignore any non-command Messages
			// Create a new MessageConfig. We don't have text yet,
			// so we leave it empty.
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

			// Extract the command from the Message.
			switch update.Message.Command() {
			case "start":
				msg.Text = "Welcome to ChatGPT bot! Write something to start a conversation. Use /new to clear context and start a new conversation."
			case "help":
				msg.Text = "Write something to start a conversation. Use /new to clear context and start a new conversation."
			case "new":
				gptUsecase.ResetUser(update.Message.From.ID)
				msg.Text = "OK, let's start a new conversation."
			default:
				msg.Text = "I don't know that command"
			}

			if _, err := bot.Send(msg); err != nil {
				log.Print(err)
			}
		} else {
			answerChan := make(chan string)
			throttledAnswerChan := make(chan string)
			userID := update.Message.Chat.ID
			msg := update.Message.Text

			wg := conc.NewWaitGroup()
			wg.Go(
				func() {
					contextTrimmed, err := gptUsecase.SendMessage(userID, msg, answerChan)
					if err != nil {
						log.Print(err)

						_, err = bot.Send(tgbotapi.NewMessage(userID, err.Error()))
						if err != nil {
							log.Print(err)
						}

						return
					}

					if contextTrimmed {
						msg := tgbotapi.NewMessage(userID, "Context trimmed.")
						_, err = bot.Send(msg)
						if err != nil {
							log.Print(err)
						}
					}
				},
			)
			wg.Go(
				func() {
					lastUpdateTime := time.Now()
					var currentAnswer string
					for answer := range answerChan {
						currentAnswer = answer
						// Update message every 2.5 seconds to avoid hitting Telegram API limits. In the documentation,
						// Although the documentation states that the limit is one message per second, in practice, it is
						// still rate-limited.
						// https://core.telegram.org/bots/faq#my-bot-is-hitting-limits-how-do-i-avoid-this
						if lastUpdateTime.Add(time.Duration(2500) * time.Millisecond).Before(time.Now()) {
							throttledAnswerChan <- currentAnswer
							lastUpdateTime = time.Now()
						}
					}
					throttledAnswerChan <- currentAnswer
					close(throttledAnswerChan)
				},
			)
			wg.Go(
				func() {
					_, err := bot.Send(tgbotapi.NewChatAction(userID, tgbotapi.ChatTyping))
					if err != nil {
						log.Print(err)
					}

					var messageID int

					for currentAnswer := range throttledAnswerChan {
						if messageID == 0 {
							msg, err := bot.Send(tgbotapi.NewMessage(userID, currentAnswer))
							if err != nil {
								log.Print(err)
							}
							messageID = msg.MessageID
						} else {
							editedMsg := tgbotapi.NewEditMessageText(userID, messageID, currentAnswer)
							_, err := bot.Send(editedMsg)
							if err != nil {
								log.Print(err)
							}
						}
					}
				},
			)

			wg.Wait()
		}
	}
	return nil
}
