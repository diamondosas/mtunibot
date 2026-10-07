package handlers

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var helpText = `▎<b>AVAILABLE COMMANDS</b>

• <b>/start</b> - Start the bot 
• <b>/notes</b> - Search available notes
• <b>/help</b> - Display commands
• <b>/clear</b> - Clear chat history

<b>Search Shortcut:</b>
Type <code>/notes &lt;topic&gt;</code> e.g /notes Java Programming to quickly search notes by  topic. e.g /notes C Programming`

func HelpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      helpText,
		ParseMode: models.ParseModeHTML,
	})

	if err != nil {
		log.Println("[HELP] Failed to send Help message:", err)
	}
}
