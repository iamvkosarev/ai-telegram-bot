# AI Telegram bot

Run your own Telegram bot with AI!

## Description

The bot has a few commands:

- `/new` - to create new chat with selected model
- `/help` - to get help info
- `/chats` - to print chats info
- `/select_chat` - to change current working chat

For managing available models there are two main (admin, premium) and default user roles.
To assign a role edit `ADMIN_TELEGRAM_ID_LIST` or `PREMIUM_TELEGRAM_ID_LIST` field at `.env` file. Example of `.env`
file contains down below at [Setup](#setup) section.

Available models for user roles can be managed in config file (`./config/config.yaml`).

If you want to make your bot public edit config's field `telegram/is_not_public` to `false`

## Setup

0. You should have these installed dependencies to run correctly:
    1. Docker
    2. make

1. Get your OpenAI API key

   You can create an account on the OpenAI website
   and [generate your API key](https://platform.openai.com/account/api-keys).

2. Get your telegram bot token

   Create a bot from Telegram [@BotFather](https://t.me/BotFather) and obtain an access token.

3. Clone repository

```bash
git clone git@github.com:iamvkosarev/ai-telegram-bot.git
```

4. Create your own `.env` file

   Example :

  ```env
OPENAI_API_KEY=<your_openai_api_key>
TELEGRAM_APITOKEN=<your_telegram_bot_token>

# Optional, default is empty. Only allow these users to use the bot with Admin role.
# ADMIN_TELEGRAM_ID_LIST=<your_telegram_id>,<your_friend_telegram_id>

# Optional, default is empty. Only allow these users to use the bot with Premium role.
# PREMIUM_TELEGRAM_ID_LIST=<your_telegram_id>,<your_friend_telegram_id>
```

5. Run an application

```bash
make run
```
