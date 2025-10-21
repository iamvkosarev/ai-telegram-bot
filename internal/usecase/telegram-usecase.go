package usecase

import (
	"errors"
	"fmt"
	api "github.com/OvyFlash/telegram-bot-api"
	"github.com/google/uuid"
	"github.com/iamvkosarev/chatgpt-telegram-bot/config"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/model"
	"github.com/sourcegraph/conc"
	"log"
	"sort"
	"strings"
	"time"
)

const (
	MessageServerError              = "Something wrong with me. Try later"
	MessageFailedToSaveMessageError = "Failed to save your message. Try later"
	MessageContextTrimmed           = "Context was trimmed"
	MessageUserNoAccess             = "You are not allowed to use this bot"
	MessageUserModelNoAccess        = "You are not allowed to use this model"
	MessageCommandStart             = "Welcome to ChatGPT bot! Write something to start a conversation. Use /new to clear context and start a new conversation."
	MessageCommandHelp              = "Write something to start a conversation. Use /new to clear context and start a new conversation."
	MessageCommandUnknown           = "I don't know that command"
	MessageSelectModel              = "Select model to create new chat"
	MessageHaveNoAvailableModels    = "You dont have any available models"
	MessageSelectedModelFormat      = "Started new chat with %s model"
	MessageFailedToGetChats         = "Failed to get all your chats"

	CommandStart = "start"
	CommandHelp  = "help"
	CommandNew   = "new"
	CommandChats = "chats"
)

var (
	ErrAIChatNotCreatedYet = errors.New("ai-chat not created yet")
)

type TelegramUsecaseDeps struct {
	User   *UserUsecase
	AIChat *AiChatUsecase
	Bot    *api.BotAPI
	OpenAI *OpenAIUsecase
}

type TelegramUsecase struct {
	TelegramUsecaseDeps
	cfg          config.Telegram
	userRoles    map[int64]model.UserRole
	allowedUsers map[int64]struct{}
}

func NewTelegramUsecase(cfg config.Telegram, deps TelegramUsecaseDeps) (*TelegramUsecase, error) {
	prepareUserRoles := make(map[int64]model.UserRole)
	allowedRoles := make(map[model.UserRole]struct{})
	allowedUsers := make(map[int64]struct{})
	for _, role := range cfg.AvailableForRoles {
		allowedRoles[model.ParseUserRole(role)] = struct{}{}
	}

	for _, userID := range cfg.AdminTelegramIDList {
		prepareUserRoles[userID] = model.UserRoleAdmin
		if _, ok := allowedRoles[model.UserRoleAdmin]; ok {
			allowedUsers[userID] = struct{}{}
		}
	}
	for _, userID := range cfg.PremiumTelegramIDList {
		prepareUserRoles[userID] = model.UserRolePremium
		if _, ok := allowedRoles[model.UserRolePremium]; ok {
			allowedUsers[userID] = struct{}{}
		}
	}

	_, err := deps.Bot.Request(
		api.NewSetMyCommands(
			[]api.BotCommand{
				{
					Command:     CommandHelp,
					Description: "Get help",
				},
				{
					Command:     CommandNew,
					Description: "Clear context and start a new conversation",
				},
				{
					Command:     CommandChats,
					Description: "Show chats",
				},
			}...,
		),
	)
	if err != nil {
		return nil, err
	}

	return &TelegramUsecase{
		TelegramUsecaseDeps: deps,
		cfg:                 cfg,
		userRoles:           prepareUserRoles,
		allowedUsers:        allowedUsers,
	}, nil
}

func (t *TelegramUsecase) GetUserRole(userID int64) model.UserRole {
	if userRole, ok := t.userRoles[userID]; ok {
		return userRole
	}
	return model.UserRoleDefault
}

func (t *TelegramUsecase) Run() error {
	u := api.NewUpdate(0)
	u.Timeout = 60

	updates := t.Bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			if err := t.handleMessage(update); err != nil {
				fmt.Printf("error handling message: %v", err.Error())
			}
		}
		if update.CallbackQuery != nil {
			if err := t.handleCallbackQuery(update); err != nil {
				fmt.Printf("error handling callback Query: %v", err.Error())
			}
		}

	}
	return nil
}

func (t *TelegramUsecase) handleCallbackQuery(update api.Update) error {
	chatID := update.CallbackQuery.Message.Chat.ID
	callbackQueryID := update.CallbackQuery.ID
	data := update.CallbackQuery.Data
	callback := api.NewCallback(callbackQueryID, data)
	if _, err := t.Bot.Request(callback); err != nil {
		return fmt.Errorf("failed to request callback: %w", err)
	}

	user, err := t.User.GetUserInfoForTelegramUser(chatID)
	if err != nil {
		t.sendMessageAndHandleErr(chatID, MessageServerError)
		return fmt.Errorf("failed to get user info for telegram user: %w", err)
	}
	aiModelsMap := t.AIChat.GetAvailableForUserModels(user)
	if len(aiModelsMap) == 0 {
		t.sendMessageAndHandleErr(chatID, MessageHaveNoAvailableModels)
		return fmt.Errorf("failed to get user models: %w", ErrUserRoleHasNotAnyAvailableModels)
	}
	if _, ok := aiModelsMap[data]; !ok {
		t.sendMessageAndHandleErr(chatID, MessageUserModelNoAccess)
		return nil
	}
	if _, err = t.createNewAIChat(user, chatID, data); err != nil {
		t.sendMessageAndHandleErr(chatID, MessageFailedToSaveMessageError)
		return fmt.Errorf("failed to create new AI chat: %w", err)
	}
	t.sendMessageAndHandleErr(chatID, fmt.Sprintf(MessageSelectedModelFormat, data))
	return nil
}

func (t *TelegramUsecase) handleMessage(update api.Update) error {
	chatID := update.Message.Chat.ID

	if t.cfg.IsNotPublic {
		if _, ok := t.allowedUsers[chatID]; !ok {
			t.sendMessageAndHandleErr(chatID, MessageUserNoAccess)
			return nil
		}
	}

	user, err := t.User.GetUserInfoForTelegramUser(chatID)
	if err != nil {
		t.sendMessageAndHandleErr(chatID, MessageServerError)
		return fmt.Errorf("failed to get user info for telegram user: %w", err)
	}

	if update.Message.IsCommand() {
		var answerText string
		switch update.Message.Command() {
		case CommandStart:
			answerText = MessageCommandStart
		case CommandHelp:
			answerText = MessageCommandHelp
		case CommandChats:
			chats, err := t.AIChat.ListUserChats(user.UserID)
			if err != nil {
				t.sendMessageAndHandleErr(chatID, MessageFailedToGetChats)
			}
			answerText = prepareUsersChats(chats)
		case CommandNew:
			if err = t.sendSelectModelsKeyboard(user, chatID); err != nil {
				return fmt.Errorf("failed to send select models keyboard: %w", err)
			}
			return nil
		default:
			answerText = MessageCommandUnknown
		}
		t.sendMessageAndHandleErr(chatID, answerText)
		return nil
	}

	aiChat, err := t.getAIChat(user, chatID)
	if err != nil {
		if errors.Is(err, ErrAIChatNotCreatedYet) {
			return nil
		}
		return fmt.Errorf("failed to get user ai-chat: %w", err)
	}

	answerChan := make(chan string)
	throttledAnswerChan := make(chan string)
	msgText := update.Message.Text

	if err = t.AIChat.AddMessageToChat(aiChat.ChatID, msgText, model.MessageSourceUser); err != nil {
		t.sendMessageAndHandleErr(chatID, MessageFailedToSaveMessageError)
		return fmt.Errorf("failed to add message to ai chat: %w", err)
	}

	wg := conc.NewWaitGroup()
	wg.Go(
		func() {
			var contextTrimmed bool
			if contextTrimmed, err = t.OpenAI.SendMessage(msgText, aiChat, answerChan); err != nil {
				t.sendMessageAndHandleErr(chatID, MessageServerError)
				log.Printf("failed to send message to gpt: %v\n", err.Error())
				return
			}

			if contextTrimmed {
				t.sendMessageAndHandleErr(chatID, MessageContextTrimmed)
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
			_, err = t.Bot.Request(api.NewChatAction(chatID, api.ChatTyping))
			if err != nil {
				log.Printf("failed to send new action to bot: %v\n", err)
			}

			var answerMsgID int
			for currentAnswer := range throttledAnswerChan {
				if len(currentAnswer) == 0 {
					continue
				}
				if answerMsgID == 0 {
					var answerMsg api.Message
					if answerMsg, err = t.sendMessage(chatID, currentAnswer); err != nil {
						log.Printf("failed to send answer to bot: %v\n", err)
					}
					answerMsgID = answerMsg.MessageID
					if err = t.AIChat.AddMessageToChat(
						aiChat.ChatID, currentAnswer, model.MessageSourceAssistant,
					); err != nil {
						log.Printf("failed to add answer to ai chat: %v\n", err)
						return
					}
				} else {
					if _, err = t.sendEditMessage(chatID, answerMsgID, currentAnswer); err != nil {
						log.Printf("failed to send new edit message to bot: %v\n", err)
					}
				}
			}
		},
	)

	wg.Wait()
	return nil
}

func prepareUsersChats(chats []model.AIChat) string {
	result := strings.Builder{}
	result.WriteString(fmt.Sprintf("Now you have %v chats.\n", len(chats)))
	for i, chat := range chats {
		result.WriteString(
			fmt.Sprintf(
				"%v) Messages: %v, Model: %s, T: %v\n", i+1, len(chat.Messages), chat.Model, chat.ModelTemperature,
			),
		)
	}
	return result.String()
}

func (t *TelegramUsecase) getAIChat(user model.User, chatID int64) (model.AIChat, error) {
	var aiChat model.AIChat
	var err error
	if user.LastAIChat != uuid.Nil {
		aiChat, err = t.AIChat.GetChat(user.LastAIChat)
		if err != nil {
			t.sendMessageAndHandleErr(chatID, MessageServerError)
			return model.AIChat{}, fmt.Errorf("failed to get user ai-chat: %w", err)
		}
	} else {
		if err = t.sendSelectModelsKeyboard(user, chatID); err != nil {
			return model.AIChat{}, fmt.Errorf("failed to send select models keyboard: %w", err)
		}
		return model.AIChat{}, ErrAIChatNotCreatedYet
	}
	return aiChat, nil
}

func (t *TelegramUsecase) sendSelectModelsKeyboard(user model.User, chatID int64) error {
	aiModelsMap := t.AIChat.GetAvailableForUserModels(user)
	if len(aiModelsMap) == 0 {
		t.sendMessageAndHandleErr(chatID, MessageHaveNoAvailableModels)
		return fmt.Errorf("failed to get user models: %w", ErrUserRoleHasNotAnyAvailableModels)
	}
	aiModels := make([]string, 0, len(aiModelsMap))
	for aiModel := range aiModelsMap {
		aiModels = append(aiModels, aiModel)
	}
	sort.Strings(aiModels)

	msg := api.NewMessage(chatID, MessageSelectModel)
	const maxButtonsInRow = 3
	inlineRows := make([][]api.InlineKeyboardButton, 0)
	inlineButtons := make([]api.InlineKeyboardButton, 0)
	for _, aiModel := range aiModels {
		if len(inlineButtons) > maxButtonsInRow {
			inlineRows = append(inlineRows, inlineButtons)
			inlineButtons = make([]api.InlineKeyboardButton, 0)
		}
		inlineButtons = append(inlineButtons, api.NewInlineKeyboardButtonData(aiModel, aiModel))
	}
	inlineRows = append(inlineRows, inlineButtons)
	msg.ReplyMarkup = api.NewInlineKeyboardMarkup(inlineRows...)
	if _, err := t.Bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send message to bot: %w", err)
	}
	return nil
}

func (t *TelegramUsecase) createNewAIChat(user model.User, chatID int64, aiModel string) (model.AIChat, error) {
	aiChat, err := t.AIChat.CreateChat(user.UserID, aiModel)
	if err != nil {
		t.sendMessageAndHandleErr(chatID, MessageServerError)
		return model.AIChat{}, fmt.Errorf("failed to create user ai-chat: %w", err)
	}
	if err = t.User.UpdateUserLastAIChat(user.UserID, aiChat.ChatID); err != nil {
		t.sendMessageAndHandleErr(chatID, MessageServerError)
		return model.AIChat{}, fmt.Errorf("failed to update user last ai-chat: %w", err)
	}
	return aiChat, nil
}

func (t *TelegramUsecase) sendMessageAndHandleErr(chatID int64, message string) api.Message {
	msg, err := t.sendMessage(chatID, message)
	if err != nil {
		log.Printf("failed to send new message to bot: %v\n", err)
	}
	return msg
}

func (t *TelegramUsecase) sendMessage(chatID int64, message string) (api.Message, error) {
	return t.sendToBot(api.NewMessage(chatID, message))
}

func (t *TelegramUsecase) sendEditMessage(chatID int64, previousMsgID int, message string) (api.Message, error) {
	return t.sendToBot(api.NewEditMessageText(chatID, previousMsgID, message))
}

func (t *TelegramUsecase) sendToBot(c api.Chattable) (api.Message, error) {
	return t.Bot.Send(c)
}
