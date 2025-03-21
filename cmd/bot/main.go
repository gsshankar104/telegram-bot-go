package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/gsshankar104/telegram-bot/internal/bot"
    "github.com/gsshankar104/telegram-bot/internal/config"
    "github.com/gsshankar104/telegram-bot/internal/storage/drive"
)

func main() {
    // Load config
    //if err := config.Load("config/config.yaml"); err != nil {
    //    log.Fatalf("Failed to load config: %v", err)
    //}

    // Create storage
    storage, err := drive.NewDriveStorage(context.Background(), "credentials.json")
    if err != nil {
        log.Fatalf("Failed to create storage: %v", err)
    }

    // Create bot
    bot, err := bot.New(storage, config.Get())
    if err != nil {
        log.Fatalf("Failed to create bot: %v", err)
    }

    // Setup signal handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigChan
        fmt.Println("\nReceived shutdown signal. Shutting down gracefully...")
        cancel()
    }()

    // Start bot
    if err := bot.Start(ctx); err != nil && err != context.Canceled {
        log.Printf("Bot stopped with error: %v", err)
    }
}
