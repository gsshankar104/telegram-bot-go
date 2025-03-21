package main

import (
    "fmt"
    "log"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
    botToken := "7537227589:AAFMQy0LYUYVd2m5EDGJtZyrSh6SfTfdxn0" // Replace with your actual bot token

    bot, err := tgbotapi.NewBotAPI(botToken)
    if err != nil {
        log.Fatalf("Error creating bot: %v", err)
    }

    fmt.Printf("Bot created successfully. Bot username: %s\n", bot.Self.UserName)
}
