package main

import (
	"flag"
	"github.com/iamvkosarev/chatgpt-telegram-bot/config"
	"github.com/iamvkosarev/chatgpt-telegram-bot/internal/app"
	"github.com/joho/godotenv"
	"log"
)

func main() {
	var cfgPath, envPath string
	flag.StringVar(&cfgPath, "config", "./config/config.yaml", "path to config")
	flag.StringVar(&envPath, "env", "./.env", "path to config")
	flag.Parse()

	err := godotenv.Load(envPath)
	if err != nil {
		log.Fatalf("loading .env file error %s\n", err)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("loading config error: %s\n", err)
	}
	if err = app.Run(cfg); err != nil {
		log.Fatalf("app error: %s\n", err)
	}

}
