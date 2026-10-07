package handlers

import (
	"context"
	"fmt"
	"html"
	"log"
	"strconv"
	"strings"

	"mtuunibot/internal/constants"
	"mtuunibot/internal/db"
	"mtuunibot/internal/notes"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// NotesHandler handles /notes and /notes <search query>
func NotesHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil || update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	text := strings.TrimSpace(update.Message.Text)

	// Check if user provided search query: /notes <query>
	parts := strings.SplitN(text, " ", 2)
	if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
		query := strings.TrimSpace(parts[1])
		handleNotesSearch(ctx, b, chatID, query)
		return
	}

	// If no search argument, check if user has profile set
	user, err := db.GetUser(userID)
	if err == nil && user != nil && user.College != "" && user.Department != "" && user.Level > 0 {
		showUserCourses(ctx, b, chatID, user, 0)
		return
	}

	// Otherwise, start the college selection menu
	showCollegeMenu(ctx, b, chatID, 0)
}

func showUserCourses(ctx context.Context, b *bot.Bot, chatID int64, user *db.User, messageID int) {
	courses, err := db.GetCoursesByDeptAndLevel(user.Department, fmt.Sprintf("%d", user.Level))

	var text string
	var rows [][]models.InlineKeyboardButton

	if err != nil || len(courses) == 0 {
		text = fmt.Sprintf("▎<b>AVAILABLE COURSES</b>\n\n• <b>Department:</b> %s\n• <b>Level:</b> %d Level\n\nNo courses or notes have been uploaded yet for this level.\nUse <code>/notes &lt;topic&gt;</code> to search all notes.",
			html.EscapeString(user.Department), user.Level)

		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "Change College / Level", CallbackData: "nav:colleges"},
		})
	} else {
		text = fmt.Sprintf("▎<b>AVAILABLE COURSES</b>\n\n• <b>Department:</b> %s\n• <b>Level:</b> %d Level\n\nSelect a course to view available files:",
			html.EscapeString(user.Department), user.Level)

		// Group course buttons in rows of 2
		var currentRow []models.InlineKeyboardButton
		for _, c := range courses {
			cTrimmed := strings.TrimSpace(c)
			if cTrimmed == "" {
				continue
			}
			btn := models.InlineKeyboardButton{
				Text:         cTrimmed,
				CallbackData: fmt.Sprintf("course:%s:0", cTrimmed),
			}
			currentRow = append(currentRow, btn)
			if len(currentRow) == 2 {
				rows = append(rows, currentRow)
				currentRow = []models.InlineKeyboardButton{}
			}
		}
		if len(currentRow) > 0 {
			rows = append(rows, currentRow)
		}

		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "Change College / Level", CallbackData: "nav:colleges"},
		})
	}

	kb := &models.InlineKeyboardMarkup{InlineKeyboard: rows}

	if messageID == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: kb,
		})
	} else {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: kb,
		})
	}
}

func handleNotesSearch(ctx context.Context, b *bot.Bot, chatID int64, query string) {
	results, err := db.SearchNotes(query)
	if err != nil || len(results) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      fmt.Sprintf("▎<b>NO RESULTS</b>\n\nNo notes found matching: <code>%s</code>\nTry searching with a course code (e.g. <code>SEN209</code>) or keyword.", html.EscapeString(query)),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	kb := notes.BuildSearchResultsKeyboard(results)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf("▎<b>SEARCH RESULTS</b>\n\nFound <b>%d</b> file(s) for <code>%s</code>:\nSelect a file to download:", len(results), html.EscapeString(query)),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: kb,
	})
}

func showCollegeMenu(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	var rows [][]models.InlineKeyboardButton
	for _, col := range constants.Colleges {
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: col, CallbackData: "college:" + col},
		})
	}
	kb := &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	text := "▎<b>SELECT COLLEGE</b>\n\nChoose your college to continue:"

	if messageID == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: kb,
		})
	} else {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:      chatID,
			MessageID:   messageID,
			Text:        text,
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: kb,
		})
	}
}

// NotesCollegeCallbackHandler handles college:<name>
func NotesCollegeCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})

	college := strings.TrimPrefix(cb.Data, "college:")
	depts := constants.CollegeDepartments[college]

	var rows [][]models.InlineKeyboardButton
	for _, dept := range depts {
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: dept, CallbackData: fmt.Sprintf("dept:%s:%s", college, dept)},
		})
	}

	// Back button to colleges
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "< Back to Colleges", CallbackData: "nav:colleges"},
	})

	text := fmt.Sprintf("▎<b>SELECT DEPARTMENT</b>\n\nCollege: <b>%s</b>\nChoose your department:", html.EscapeString(college))

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      cb.Message.Message.Chat.ID,
		MessageID:   cb.Message.Message.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
	})
}

// NotesDeptCallbackHandler handles dept:<college>:<dept>
func NotesDeptCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})

	parts := strings.SplitN(strings.TrimPrefix(cb.Data, "dept:"), ":", 2)
	if len(parts) < 2 {
		return
	}
	college := parts[0]
	dept := parts[1]

	var rows [][]models.InlineKeyboardButton
	for _, lvl := range constants.Levels {
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: fmt.Sprintf("%d Level", lvl), CallbackData: fmt.Sprintf("level:%s:%s:%d", college, dept, lvl)},
		})
	}

	// Back button to department list for this college
	rows = append(rows, []models.InlineKeyboardButton{
		{Text: "< Back to Departments", CallbackData: "college:" + college},
	})

	text := fmt.Sprintf("▎<b>SELECT LEVEL</b>\n\n• <b>College:</b> %s\n• <b>Department:</b> %s\n\nChoose your academic level:",
		html.EscapeString(college), html.EscapeString(dept))

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      cb.Message.Message.Chat.ID,
		MessageID:   cb.Message.Message.ID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows},
	})
}

// NotesLevelCallbackHandler handles level:<college>:<dept>:<level>
func NotesLevelCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})

	parts := strings.Split(strings.TrimPrefix(cb.Data, "level:"), ":")
	if len(parts) < 3 {
		return
	}
	college := parts[0]
	dept := parts[1]
	lvlInt, _ := strconv.Atoi(parts[2])

	userID := cb.From.ID
	chatID := cb.Message.Message.Chat.ID
	msgID := cb.Message.Message.ID

	// Save profile to database
	err := db.UpdateUserProfile(userID, college, dept, lvlInt)
	if err != nil {
		log.Println("[PROFILE] Error updating user profile:", err)
	}

	user, _ := db.GetUser(userID)
	if user == nil {
		user = &db.User{College: college, Department: dept, Level: lvlInt}
	}

	// Directly show available courses for this level
	showUserCourses(ctx, b, chatID, user, msgID)
}

// NotesCourseCallbackHandler handles course:<course_code>:<page> and page:<course_code>:<page>
func NotesCourseCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})

	data := cb.Data
	var courseCode string
	var page int

	if strings.HasPrefix(data, "course:") {
		parts := strings.Split(strings.TrimPrefix(data, "course:"), ":")
		courseCode = parts[0]
		if len(parts) > 1 {
			page, _ = strconv.Atoi(parts[1])
		}
	} else if strings.HasPrefix(data, "page:") {
		parts := strings.Split(strings.TrimPrefix(data, "page:"), ":")
		courseCode = parts[0]
		if len(parts) > 1 {
			page, _ = strconv.Atoi(parts[1])
		}
	}

	userID := cb.From.ID
	chatID := cb.Message.Message.Chat.ID
	msgID := cb.Message.Message.ID

	user, _ := db.GetUser(userID)
	var notesList []db.Note
	if user != nil && user.Department != "" {
		notesList, _ = db.GetNotesByCourse(user.Department, fmt.Sprintf("%d", user.Level), courseCode)
	}

	// If no notes by strict dept, search across course code
	if len(notesList) == 0 {
		notesList, _ = db.SearchNotes(courseCode)
	}

	if len(notesList) == 0 {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: msgID,
			Text:      fmt.Sprintf("▎<b>NO FILES</b>\n\nNo files found for course: <code>%s</code>", html.EscapeString(courseCode)),
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: "< Back to Courses", CallbackData: "nav:courses"}},
				},
			},
		})
		return
	}

	kb, text := notes.BuildPaginatedNotesKeyboard(courseCode, notesList, page, 3)

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   msgID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: kb,
	})
}

// NotesNavCallbackHandler handles navigation back buttons (nav:colleges, nav:courses)
func NotesNavCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})

	chatID := cb.Message.Message.Chat.ID
	msgID := cb.Message.Message.ID
	userID := cb.From.ID

	action := strings.TrimPrefix(cb.Data, "nav:")
	switch action {
	case "colleges":
		showCollegeMenu(ctx, b, chatID, msgID)
	case "courses":
		user, _ := db.GetUser(userID)
		if user != nil && user.Department != "" {
			showUserCourses(ctx, b, chatID, user, msgID)
		} else {
			showCollegeMenu(ctx, b, chatID, msgID)
		}
	}
}

// NotesDownloadCallbackHandler handles dl:<note_id>
func NotesDownloadCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	cb := update.CallbackQuery
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID, Text: "Sending file..."})

	idStr := strings.TrimPrefix(cb.Data, "dl:")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return
	}

	notes.SendNoteDocument(ctx, b, cb.Message.Message.Chat.ID, uint(id))
}

// NoopCallbackHandler handles dummy/indicator buttons
func NoopCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: update.CallbackQuery.ID})
	}
}
