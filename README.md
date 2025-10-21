# ChatGPT Telegram bot

Run your own ChatGPT Telegram bot!

## Setup

1. Get your OpenAI API key

   You can create an account on the OpenAI website and [generate your API key](https://platform.openai.com/account/api-keys).

2. Get your telegram bot token

   Create a bot from Telegram [@BotFather](https://t.me/BotFather) and obtain an access token.

3. Clone repository
```bash
git clone git@github.com:iamvkosarev/chatgpt-telegram-bot.git
```

4. Create your own `.env` file

   Example : 
  ```env
OPENAI_API_KEY=<your_openai_api_key>
TELEGRAM_APITOKEN=<your_telegram_bot_token>

; Optional, default is empty. Only allow these users to use the bot with Admin role.
; ADMIN_TELEGRAM_ID_LIST=<your_telegram_id>,<your_friend_telegram_id>

; Optional, default is empty. Only allow these users to use the bot with Premium role.
; PREMIUM_TELEGRAM_ID_LIST=<your_telegram_id>,<your_friend_telegram_id>
```
5. Run an application

```bash
 go run ./cmd/main.go
```
