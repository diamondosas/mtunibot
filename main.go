package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"mtuunibot/internal"
	"mtuunibot/internal/db"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, os.Interrupt)
	defer cancel()

	cfg := LoadConfig()
	err := db.InitDB()
	if err != nil {
		log.Println("Could not init database:", err)
	}

	db.StartServer(cfg.Port)

	if cfg.BotToken == "" {
		log.Println("Please set BOT_TOKEN in .env file")
		return
	}

	if cfg.AIApiKey == "" {
		log.Println("Warning: AI_API_KEY is not set in .env file. Note watcher will fail AI parsing without it.")
	}

	internal.StartBot(ctx, cfg.BotToken, cfg.AIApiKey, cfg.ArchiveChannelID)
}
