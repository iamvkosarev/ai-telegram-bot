package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/iamvkosarev/ai-telegram-bot/config"
	"github.com/iamvkosarev/ai-telegram-bot/internal/model"
	openai_tools "github.com/iamvkosarev/ai-telegram-bot/pkg/openai-tools"
	"io"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

const (
	OpenAIRoleUser      = "user"
	OpenAIRoleAssistant = "assistant"
	OpenAIRoleUnknown   = "unknown"
)

type OpenAIUsecase struct {
	cfg config.OpenAI
}

type UserState struct {
	TelegramID     int64
	LastActiveTime time.Time
	HistoryMessage []openai.ChatCompletionMessage
	SelectedModel  string
}

func NewOpenAIUsecase(cfg config.OpenAI) *OpenAIUsecase {
	return &OpenAIUsecase{
		cfg: cfg,
	}
}

func (gpt *OpenAIUsecase) SendMessage(msg string, chat model.AIChat, answerChan chan<- string) (bool, error) {
	messageHistory := make([]openai.ChatCompletionMessage, 0, len(chat.Messages)+1)
	for _, message := range chat.Messages {
		messageHistory = append(
			messageHistory, openai.ChatCompletionMessage{
				Role:    parseMessageSourceToRole(message.Source),
				Content: message.Body,
			},
		)
	}
	messageHistory = append(
		messageHistory, openai.ChatCompletionMessage{
			Role:    OpenAIRoleUser,
			Content: msg,
		},
	)

	trimHistory := func() {
		messageHistory = messageHistory[1:]
		fmt.Println("History trimmed due to token limit")
	}
	for len(messageHistory) > 0 {
		tokenCount, err := openai_tools.CountToken(messageHistory, chat.Model)
		if err != nil {
			fmt.Println("count token error:", err)

			// How should we deal with this error?
			trimHistory()
			continue
		}

		if tokenCount < 3500 {
			break
		}
		trimHistory()
	}

	clientConfig := openai.DefaultConfig(gpt.cfg.OpenAIAPIKey)
	clientConfig.BaseURL = gpt.cfg.OpenAIBaseURL
	c := openai.NewClientWithConfig(clientConfig)
	ctx := context.Background()

	req := openai.ChatCompletionRequest{
		Model:       chat.Model,
		Temperature: chat.ModelTemperature,
		TopP:        1,
		N:           1,
		// TODO: add to config or Roles
		// PresencePenalty:  0.2,
		// FrequencyPenalty: 0.2,
		Messages: messageHistory,
		Stream:   true,
	}

	stream, err := c.CreateChatCompletionStream(ctx, req)
	if err != nil {
		log.Print(err)
		return false, err
	}

	var currentAnswer string

	defer stream.Close()
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			close(answerChan)
			break
		}

		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			close(answerChan)
			break
		}

		currentAnswer += response.Choices[0].Delta.Content
		answerChan <- currentAnswer
	}
	return false, nil
}

func parseMessageSourceToRole(source model.MessageSource) string {
	switch source {
	case model.MessageSourceUser:
		return OpenAIRoleUser
	case model.MessageSourceAssistant:
		return OpenAIRoleAssistant
	default:
		return OpenAIRoleUnknown
	}
}
