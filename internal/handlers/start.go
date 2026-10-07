package handlers

import (
	"context"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"

	"mtuunibot/internal/db"
	"mtuunibot/internal/notes"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}

	_ = db.DoesUserExist(update.Message.From.ID, update.Message.From.FirstName)

	// Check if deep link download payload was passed: /start dl_<id>
	text := strings.TrimSpace(update.Message.Text)
	parts := strings.Split(text, " ")
	if len(parts) > 1 && strings.HasPrefix(parts[1], "dl_") {
		idStr := strings.TrimPrefix(parts[1], "dl_")
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			notes.SendNoteDocument(ctx, b, update.Message.Chat.ID, uint(id))
			return
		}
	}

	firstName := update.Message.From.FirstName
	if firstName == "" && update.Message.Chat.FirstName != "" {
		firstName = update.Message.Chat.FirstName
	}

	startText := fmt.Sprintf("▎<b>MOUNTAIN TOP UNIVERSITY BOT</b>\n\n"+
		"Hello %s!\n\n"+
		"You can do these actions:\n\n"+
		"• Download Notes using /notes\n"+
		"• Search Notes using /notes &lt;topic&gt;\n\n"+
		"Type \"/\" or /help to see available commands.",
		html.EscapeString(firstName))

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      startText,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		log.Println("[START] Error sending start message:", err)
	}
}
