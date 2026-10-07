package notes

import (
	"context"
	"fmt"
	"html"
	"log"

	"mtuunibot/internal/db"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// SendNoteDocument sends the actual file to user by FileID
func SendNoteDocument(ctx context.Context, b *bot.Bot, chatID int64, noteID uint) {
	note, err := db.GetNoteByID(noteID)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      "▎<b>NOTE NOT FOUND</b>\n\nThe requested file is no longer available.",
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	caption := fmt.Sprintf("▎<b>%s</b> (%s)\n\n• <b>Department:</b> %s\n• <b>Level:</b> %s",
		html.EscapeString(note.Title),
		html.EscapeString(note.CourseCode),
		html.EscapeString(note.Department),
		html.EscapeString(note.Level))

	_, err = b.SendDocument(ctx, &bot.SendDocumentParams{
		ChatID: chatID,
		Document: &models.InputFileString{
			Data: note.FileID,
		},
		Caption:   caption,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		log.Printf("[DOWNLOAD] Failed to send note document %d: %v", noteID, err)
	}
}

// BuildPaginatedNotesKeyboard builds a 3-per-page paginated keyboard with navigation buttons
func BuildPaginatedNotesKeyboard(courseCode string, notesList []db.Note, page, pageSize int) (*models.InlineKeyboardMarkup, string) {
	totalNotes := len(notesList)
	if pageSize <= 0 {
		pageSize = 3
	}

	totalPages := (totalNotes + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * pageSize
	end := start + pageSize
	if end > totalNotes {
		end = totalNotes
	}

	pageItems := notesList[start:end]

	var text string
	text = fmt.Sprintf("▎<b>COURSE: %s</b>\n\nPage %d of %d (%d file(s) available)\nSelect a file to download:",
		html.EscapeString(courseCode), page+1, totalPages, totalNotes)

	var rows [][]models.InlineKeyboardButton

	// Add button for each file in current page (max 3 files)
	for i, n := range pageItems {
		itemNum := start + i + 1
		btnText := n.FileName
		if btnText == "" {
			btnText = n.Title
		}
		if len(btnText) > 36 {
			btnText = btnText[:33] + "..."
		}
		label := fmt.Sprintf("%d. %s", itemNum, btnText)

		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         label,
				CallbackData: fmt.Sprintf("dl:%d", n.ID),
			},
		})
	}

	// Pagination row if multiple pages exist
	if totalPages > 1 {
		var navRow []models.InlineKeyboardButton

		if page > 0 {
			navRow = append(navRow, models.InlineKeyboardButton{
				Text:         "<< Prev",
				CallbackData: fmt.Sprintf("page:%s:%d", courseCode, page-1),
			})
		}

		navRow = append(navRow, models.InlineKeyboardButton{
			Text:         fmt.Sprintf("[%d/%d]", page+1, totalPages),
			CallbackData: "noop",
		})

		if page < totalPages-1 {
			navRow = append(navRow, models.InlineKeyboardButton{
				Text:         "Next >>",
				CallbackData: fmt.Sprintf("page:%s:%d", courseCode, page+1),
			})
		}

		rows = append(rows, navRow)
	}

	// Bottom navigation row
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "< Back to Courses", CallbackData: "nav:courses"},
		{Text: "Change Profile", CallbackData: "nav:colleges"},
	})

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}, text
}

// BuildSearchResultsKeyboard builds a list keyboard for search results
func BuildSearchResultsKeyboard(noteList []db.Note) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton

	for _, n := range noteList {
		btnText := fmt.Sprintf("%s - %s", n.CourseCode, n.Title)
		if len(btnText) > 40 {
			btnText = btnText[:37] + "..."
		}
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         btnText,
				CallbackData: fmt.Sprintf("dl:%d", n.ID),
			},
		})
	}

	rows = append(rows, []models.InlineKeyboardButton{
		{
			Text:         "Change College / Level",
			CallbackData: "nav:colleges",
		},
	})

	return &models.InlineKeyboardMarkup{InlineKeyboard: rows}
}