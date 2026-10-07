package notes

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"context"
	"crypto/md5"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"mtuunibot/internal/ai"
	"mtuunibot/internal/constants"
	"mtuunibot/internal/db"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var tagRegex = regexp.MustCompile(`<[^>]*>`)
var wordRegex = regexp.MustCompile(`\s+`)

// ExtractSnippet grabs the first 50-100 words from supported document formats using standard Go.
func ExtractSnippet(fileName string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	var rawText string

	switch ext {
	case ".pptx":
		rawText = extractPPTXText(data)
	case ".docx":
		rawText = extractDOCXText(data)
	case ".pdf":
		rawText = extractPDFText(data)
	case ".txt", ".md":
		rawText = string(data)
	default:
		rawText = fileName
	}

	rawText = strings.TrimSpace(rawText)
	if rawText == "" {
		return fileName
	}

	words := wordRegex.Split(rawText, -1)
	if len(words) > 100 {
		return strings.Join(words[:100], " ")
	}
	return strings.Join(words, " ")
}

func extractPPTXText(data []byte) string {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ""
	}

	var sb strings.Builder
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, _ := io.ReadAll(rc)
			rc.Close()

			clean := tagRegex.ReplaceAllString(string(content), " ")
			sb.WriteString(clean + " ")
			if len(strings.Fields(sb.String())) >= 120 {
				break
			}
		}
	}
	return sb.String()
}

func extractDOCXText(data []byte) string {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ""
	}

	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			content, _ := io.ReadAll(rc)
			rc.Close()
			return tagRegex.ReplaceAllString(string(content), " ")
		}
	}
	return ""
}

func extractPDFText(data []byte) string {
	var sb strings.Builder

	// Scan for stream ... endstream blocks in PDF
	streamStart := []byte("stream")
	streamEnd := []byte("endstream")

	pos := 0
	for {
		startIdx := bytes.Index(data[pos:], streamStart)
		if startIdx == -1 {
			break
		}
		actualStart := pos + startIdx + len(streamStart)
		if actualStart < len(data) && (data[actualStart] == '\r' || data[actualStart] == '\n') {
			if data[actualStart] == '\r' && actualStart+1 < len(data) && data[actualStart+1] == '\n' {
				actualStart += 2
			} else {
				actualStart++
			}
		}

		endIdx := bytes.Index(data[actualStart:], streamEnd)
		if endIdx == -1 {
			break
		}
		actualEnd := actualStart + endIdx

		streamData := data[actualStart:actualEnd]
		pos = actualEnd + len(streamEnd)

		// Attempt decompression via zlib
		zr, err := zlib.NewReader(bytes.NewReader(streamData))
		if err == nil {
			uncompressed, err := io.ReadAll(zr)
			zr.Close()
			if err == nil {
				extractPDFOperators(uncompressed, &sb)
			}
		} else {
			extractPDFOperators(streamData, &sb)
		}

		if len(strings.Fields(sb.String())) >= 120 {
			break
		}
	}

	result := sb.String()
	if strings.TrimSpace(result) == "" {
		// Fallback: extract visible ASCII words from raw PDF bytes
		return extractAsciiStrings(data)
	}
	return result
}

func extractPDFOperators(data []byte, sb *strings.Builder) {
	// Look for text in (...) Tj or [...] TJ
	inParen := false
	var cur bytes.Buffer
	for i := 0; i < len(data); i++ {
		c := data[i]
		if c == '(' && !inParen {
			inParen = true
			cur.Reset()
			continue
		}
		if c == ')' && inParen {
			inParen = false
			txt := cur.String()
			if len(txt) > 1 {
				sb.WriteString(txt + " ")
			}
			continue
		}
		if inParen {
			cur.WriteByte(c)
		}
	}
}

func extractAsciiStrings(data []byte) string {
	var sb strings.Builder
	var cur bytes.Buffer
	for _, b := range data {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			cur.WriteByte(b)
		} else {
			if cur.Len() >= 4 {
				sb.WriteString(cur.String() + " ")
			}
			cur.Reset()
		}
	}
	return sb.String()
}

// ProcessUploadedDocument downloads a file from Telegram, extracts text, calls AI, and saves note to DB.
func ProcessUploadedDocument(ctx context.Context, b *bot.Bot, botToken, aiApiKey string, msg *models.Message) {
	if msg == nil || msg.Document == nil {
		return
	}

	doc := msg.Document
	ext := strings.ToLower(filepath.Ext(doc.FileName))
	if ext != ".pdf" && ext != ".pptx" && ext != ".docx" && ext != ".txt" {
		log.Printf("[WATCHER] Ignoring unsupported file extension: %s", doc.FileName)
		return
	}

	chatID := msg.Chat.ID
	log.Printf("[WATCHER] File detected in chat %d (%s): %s (%d bytes)", chatID, msg.Chat.Title, doc.FileName, doc.FileSize)

	// Immediately notify chat that the file has been received
	statusMsg, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      fmt.Sprintf("▎<b>DOCUMENT RECEIVED</b>\n\nFile: <code>%s</code>\nProcessing content with AI...", html.EscapeString(doc.FileName)),
		ParseMode: models.ParseModeHTML,
	})
	if sendErr != nil {
		log.Printf("[WATCHER] Note: could not send immediate receipt message to chat %d: %v", chatID, sendErr)
	} else {
		log.Printf("[WATCHER] Sent receipt acknowledgment to chat %d", chatID)
	}

	// 1. Get file path from Telegram
	log.Printf("[WATCHER] Requesting Telegram file path for file_id: %s...", doc.FileID)
	fileInfo, err := b.GetFile(ctx, &bot.GetFileParams{FileID: doc.FileID})
	if err != nil {
		log.Printf("[WATCHER] Failed to get file info from Telegram: %v", err)
		return
	}

	// 2. Download file content
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, fileInfo.FilePath)
	log.Printf("[WATCHER] Downloading file from Telegram servers...")
	resp, err := http.Get(fileURL)
	if err != nil {
		log.Printf("[WATCHER] Failed to download file from %s: %v", fileURL, err)
		return
	}
	defer resp.Body.Close()

	fileBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[WATCHER] Failed reading file bytes: %v", err)
		return
	}
	log.Printf("[WATCHER] Downloaded %d bytes for %s", len(fileBytes), doc.FileName)

	// 3. Compute MD5 content hash & check for duplicates
	hashBytes := md5.Sum(fileBytes)
	contentHash := fmt.Sprintf("%x", hashBytes)
	log.Printf("[WATCHER] Content MD5 for %s: %s", doc.FileName, contentHash)

	existingNote, err := db.GetNoteByHash(contentHash)
	if err == nil && existingNote != nil {
		log.Printf("[WATCHER] Duplicate file detected: hash=%s (matches ID %d: %s - %s)",
			contentHash, existingNote.ID, existingNote.CourseCode, existingNote.Title)

		dupText := fmt.Sprintf("▎<b>DUPLICATE FILE DETECTED</b>\n\n"+
			"This file is already in the database:\n"+
			"• <b>Course:</b> %s\n"+
			"• <b>Title:</b> %s\n\n"+
			"Both messages will be deleted in 5 seconds.",
			html.EscapeString(existingNote.CourseCode),
			html.EscapeString(existingNote.Title))

		statusMsgID := 0
		if statusMsg != nil {
			statusMsgID = statusMsg.ID
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsgID,
				Text:      dupText,
				ParseMode: models.ParseModeHTML,
			})
		} else {
			sentMsg, _ := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatID,
				Text:      dupText,
				ParseMode: models.ParseModeHTML,
			})
			if sentMsg != nil {
				statusMsgID = sentMsg.ID
			}
		}

		// Delete both the bot's duplicate notification and the uploaded file in the chat after 5 seconds
		go func(cID int64, bMsgID, uMsgID int) {
			time.Sleep(5 * time.Second)
			if bMsgID > 0 {
				_, _ = b.DeleteMessage(context.Background(), &bot.DeleteMessageParams{
					ChatID:    cID,
					MessageID: bMsgID,
				})
			}
			if uMsgID > 0 {
				_, _ = b.DeleteMessage(context.Background(), &bot.DeleteMessageParams{
					ChatID:    cID,
					MessageID: uMsgID,
				})
			}
			log.Printf("[WATCHER] Deleted duplicate messages in chat %d (botMsg: %d, userMsg: %d)", cID, bMsgID, uMsgID)
		}(chatID, statusMsgID, msg.ID)

		return
	}

	// 4. Extract snippet (first 50-100 words) using standard Go
	log.Printf("[WATCHER] Extracting text snippet (%s)...", ext)
	snippet := ExtractSnippet(doc.FileName, fileBytes)
	log.Printf("[WATCHER] Extracted snippet (%d chars): %q", len(snippet), snippet)

	// 5. Send to Groq AI
	log.Printf("[WATCHER] Sending snippet to Groq AI for analysis...")
	meta, err := ai.AnalyzeDocumentWithGroq(aiApiKey, doc.FileName, snippet)
	if err != nil {
		log.Printf("[WATCHER] Groq AI analysis failed: %v", err)
		if statusMsg != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
				Text:      fmt.Sprintf("▎<b>ANALYSIS FAILED</b>\n\nFile: <code>%s</code>\nError: %s", html.EscapeString(doc.FileName), html.EscapeString(err.Error())),
				ParseMode: models.ParseModeHTML,
			})
		}
		return
	}
	log.Printf("[WATCHER] AI Analysis Complete: Course=%s, Title=%s, Dept=%s, Level=%s, Tags=%s",
		meta.CourseCode, meta.Title, meta.Department, meta.Level, meta.Tags)

	// 6. Validate department against registered university departments
	matchedDept, matchedCollege, ok := constants.FindMatchingDepartment(meta.Department)
	if !ok || strings.EqualFold(meta.Department, "NONE") || strings.EqualFold(meta.Department, "UNKNOWN") {
		log.Printf("[WATCHER] File discarded: No suitable department found for %s (AI returned dept=%q)", doc.FileName, meta.Department)
		discardMsg := fmt.Sprintf("▎<b>FILE DISCARDED</b>\n\n"+
			"File: <code>%s</code>\n"+
			"Reason: No matching department found among registered university departments.",
			html.EscapeString(doc.FileName))

		if statusMsg != nil {
			_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: statusMsg.ID,
				Text:      discardMsg,
				ParseMode: models.ParseModeHTML,
			})
		} else {
			_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatID,
				Text:      discardMsg,
				ParseMode: models.ParseModeHTML,
			})
		}
		return
	}

	// Use official validated department and college name
	meta.Department = matchedDept
	if matchedCollege != "" {
		meta.College = matchedCollege
	}

	// 7. Save Note into DB with FileHash
	note := db.Note{
		CourseCode: strings.ToUpper(strings.TrimSpace(meta.CourseCode)),
		Title:      meta.Title,
		College:    meta.College,
		Department: meta.Department,
		Level:      meta.Level,
		FileID:     doc.FileID,
		FileName:   doc.FileName,
		FileHash:   contentHash,
		Tags:       meta.Tags,
	}

	err = db.SaveNote(&note)
	if err != nil {
		log.Printf("[WATCHER] Failed saving note to SQLite: %v", err)
		return
	}
	log.Printf("[WATCHER] Note successfully saved to database (ID: %d)", note.ID)

	// 8. Notify success in chat
	replyText := fmt.Sprintf("▎<b>NOTE SAVED</b>\n\n"+
		"• <b>Course:</b> %s\n"+
		"• <b>Title:</b> %s\n"+
		"• <b>College:</b> %s\n"+
		"• <b>Department:</b> %s\n"+
		"• <b>Level:</b> %s\n"+
		"• <b>Tags:</b> <i>%s</i>",
		html.EscapeString(note.CourseCode),
		html.EscapeString(note.Title),
		html.EscapeString(note.College),
		html.EscapeString(note.Department),
		html.EscapeString(note.Level),
		html.EscapeString(note.Tags))

	if statusMsg != nil {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: statusMsg.ID,
			Text:      replyText,
			ParseMode: models.ParseModeHTML,
		})
	} else {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      replyText,
			ParseMode: models.ParseModeHTML,
		})
	}
	log.Printf("[WATCHER] Completed processing for %s", doc.FileName)
}
