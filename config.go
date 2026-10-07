package main

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken         string
	AIApiKey         string
	ArchiveChannelID string
	Port             string
}

func LoadConfig() *Config {
	_ = godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	botToken := os.Getenv("BOT_TOKEN")
	aiApiKey := os.Getenv("AI_API_KEY")
	if aiApiKey == "" {
		aiApiKey = os.Getenv("GROQ_API_KEY")
	}

	if botToken == "" {
		botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	archiveChannelID := os.Getenv("ARCHIVE_CHANNEL_ID")
	if archiveChannelID == "" {
		archiveChannelID = os.Getenv("GROUP_ID")
	}

	return &Config{
		BotToken:         botToken,
		AIApiKey:         aiApiKey,
		ArchiveChannelID: archiveChannelID,
		Port:             port,
	}
}