package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"time"
)

type GPT struct {
	OpenAIAPIKey            string        `env:"OPENAI_API_KEY,required"`
	OPENAIModel             string        `yaml:"openai_model" env:"OPENAI_MODEL"`
	OpenAIBaseURL           string        `yaml:"open_ai_base_url" env:"OPENAI_BASE_URL"`
	ModelTemperature        float32       `yaml:"model_temperature" env:"MODEL_TEMPERATURE"`
	ConversationIdleTimeout time.Duration `yaml:"conversation_idle_timeout_seconds"`
}

type Telegram struct {
	TelegramAPIToken                    string  `env:"TELEGRAM_APITOKEN,required"`
	AllowedTelegramID                   []int64 `env:"ALLOWED_TELEGRAM_ID" envSeparator:","`
	NotifyUserOnConversationIdleTimeout bool    `yaml:"notify_user_on_conversation_idle_timeout" envDefault:"false"`
}

type Config struct {
	GPT      GPT      `yaml:"gpt"`
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
