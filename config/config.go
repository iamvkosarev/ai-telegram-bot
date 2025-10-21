package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

type OpenAI struct {
	OpenAIAPIKey            string        `env:"OPENAI_API_KEY,required"`
	OpenAIBaseURL           string        `yaml:"open_ai_base_url" env:"OPENAI_BASE_URL"`
	ConversationIdleTimeout time.Duration `yaml:"conversation_idle_timeout_seconds"`
	StdModel                string        `env:"OPENAI_STD_MODEL" envDefault:"gpt-3.5-turbo"`
}

type Telegram struct {
	TelegramAPIToken                    string  `env:"TELEGRAM_APITOKEN,required"`
	AllowedTelegramID                   []int64 `env:"ALLOWED_TELEGRAM_ID" envSeparator:","`
	NotifyUserOnConversationIdleTimeout bool    `yaml:"notify_user_on_conversation_idle_timeout" envDefault:"false"`
}

type Config struct {
	OpenAI   OpenAI   `yaml:"open_ai"`
	Telegram Telegram `yaml:"telegram"`
}

func LoadConfig(cfgPath string) (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(cfgPath, &cfg); err != nil {
		return nil, err
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
