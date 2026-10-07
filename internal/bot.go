package internal

import (
	"context"
	"fmt"
	"log"

	"mtuunibot/internal/handlers"
	"mtuunibot/internal/notes"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func getMessageFromUpdate(update *models.Update) *models.Message {
	if update.Message != nil {
		return update.Message
	}
	if update.ChannelPost != nil {
		return update.ChannelPost
	}
	if update.EditedMessage != nil {
		return update.EditedMessage
	}
	if update.EditedChannelPost != nil {
		return update.EditedChannelPost
	}
	return nil
}

func StartBot(ctx context.Context, botToken, aiApiKey, archiveChannelID string) {
	log.Println("Starting Bot...")

	loggingMiddleware := func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			msg := getMessageFromUpdate(update)
			if msg != nil {
				sender := "anonymous/channel"
				if msg.From != nil {
					sender = msg.From.Username
					if sender == "" {
						sender = msg.From.FirstName
					}
				}
				docInfo := ""
				if msg.Document != nil {
					docInfo = fmt.Sprintf(" | 📄 Document: %s (%d bytes)", msg.Document.FileName, msg.Document.FileSize)
				}
				log.Printf("[LIVE LOG] Chat %d (%s, type=%s) | From: %s | Text: %q%s",
					msg.Chat.ID, msg.Chat.Title, msg.Chat.Type, sender, msg.Text, docInfo)
			} else if update.CallbackQuery != nil {
				log.Printf("[LIVE LOG] Callback query from %s: %s",
					update.CallbackQuery.From.Username, update.CallbackQuery.Data)
			} else {
				log.Printf("[LIVE LOG] Received update ID: %d", update.ID)
			}
			next(ctx, b, update)
		}
	}

	opts := []bot.Option{
		bot.WithMiddlewares(loggingMiddleware),
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			msg := getMessageFromUpdate(update)
			if msg == nil {
				return
			}

			// Watch for uploaded documents (.pdf, .pptx, etc.) in groups, supergroups, and channels
			if msg.Document != nil {
				chatIDStr := fmt.Sprintf("%d", msg.Chat.ID)
				isTargetChat := archiveChannelID == "" || chatIDStr == archiveChannelID

				if isTargetChat {
					log.Printf("[WATCHER] Handing off document to processor: %s", msg.Document.FileName)
					go notes.ProcessUploadedDocument(ctx, b, botToken, aiApiKey, msg)
					return
				}
			}
		}),
	}

	b, err := bot.New(botToken, opts...)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	_, err = b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{
				Command:     "start",
				Description: "Start the BOT",
			},
			{
				Command:     "notes",
				Description: "Search or browse notes",
			},
			{
				Command:     "help",
				Description: "Print out help",
			},
			{
				Command:     "clear",
				Description: "Clear messages",
			},
		},
	})
	if err != nil {
		log.Println("Warning setting commands:", err)
	}

	// Command Handlers
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeContains, handlers.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeContains, handlers.HelpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/notes", bot.MatchTypeContains, handlers.NotesHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/clear", bot.MatchTypeContains, handlers.ClearHandler)

	// Callback Query Handlers
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "college:", bot.MatchTypePrefix, handlers.NotesCollegeCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "dept:", bot.MatchTypePrefix, handlers.NotesDeptCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "level:", bot.MatchTypePrefix, handlers.NotesLevelCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "course:", bot.MatchTypePrefix, handlers.NotesCourseCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "page:", bot.MatchTypePrefix, handlers.NotesCourseCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "nav:", bot.MatchTypePrefix, handlers.NotesNavCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "dl:", bot.MatchTypePrefix, handlers.NotesDownloadCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "noop", bot.MatchTypeExact, handlers.NoopCallbackHandler)

	log.Println("Bot Ready and listening for messages, channel posts, and callbacks...")
	b.Start(ctx)
}
