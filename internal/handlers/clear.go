package handlers

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func ClearHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Chat.Type != "private" {
		return
	}

	chatID := update.Message.Chat.ID
	lastMessageID := update.Message.ID

	for end := lastMessageID; end > 0; end -= 100 {
		start := end - 99
		if start < 1 {
			start = 1
		}

		messageIDs := make([]int, 0, end-start+1)
		for id := start; id <= end; id++ {
			messageIDs = append(messageIDs, id)
		}

		_, err := b.DeleteMessages(ctx, &bot.DeleteMessagesParams{
			ChatID:     chatID,
			MessageIDs: messageIDs,
		})

		if err != nil {
			for _, id := range messageIDs {
				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    chatID,
					MessageID: id,
				})
			}
		}
	}

	log.Println("Cleared messages for", update.Message.Chat.Username)
}