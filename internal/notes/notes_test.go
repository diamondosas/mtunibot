package notes

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"mtuunibot/internal/db"
)

func TestExtractSnippetPlainText(t *testing.T) {
	longText := strings.Repeat("hello world test university ", 20) // 80 words
	snippet := ExtractSnippet("test.txt", []byte(longText))
	words := strings.Fields(snippet)
	if len(words) > 100 {
		t.Errorf("Expected <= 100 words, got %d", len(words))
	}
	if len(words) == 0 {
		t.Errorf("Expected non-empty snippet")
	}
}

func TestExtractSnippetPPTX(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	slideContent := `<?xml version="1.0" encoding="UTF-8"?><p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>COURSE TITLE: SOFTWARE CONSTRUCTION</a:t></a:r></a:p><a:p><a:r><a:t>COURSE CODE: SEN 209</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:sld>`
	f, err := zw.Create("ppt/slides/slide1.xml")
	if err != nil {
		t.Fatalf("Failed creating slide in zip: %v", err)
	}
	_, _ = f.Write([]byte(slideContent))
	_ = zw.Close()

	snippet := ExtractSnippet("sample.pptx", buf.Bytes())
	if !strings.Contains(snippet, "SOFTWARE CONSTRUCTION") {
		t.Errorf("Expected snippet to contain extracted slide text, got: %q", snippet)
	}
	if !strings.Contains(snippet, "SEN 209") {
		t.Errorf("Expected snippet to contain course code, got: %q", snippet)
	}
}

func TestBuildPaginatedNotesKeyboard(t *testing.T) {
	// Create 7 notes (should result in 3 pages with pageSize 3)
	var notesList []db.Note
	for i := 1; i <= 7; i++ {
		notesList = append(notesList, db.Note{
			ID:         uint(i),
			CourseCode: "SEN209",
			Title:      fmt.Sprintf("Lecture %d", i),
			FileName:   fmt.Sprintf("Lecture_%d.pptx", i),
		})
	}

	// Page 0 (items 1, 2, 3)
	kb0, text0 := BuildPaginatedNotesKeyboard("SEN209", notesList, 0, 3)
	if !strings.Contains(text0, "Page 1 of 3") {
		t.Errorf("Expected Page 1 of 3 in text, got: %s", text0)
	}
	if len(kb0.InlineKeyboard) < 3 {
		t.Errorf("Expected at least 3 rows for notes + nav")
	}

	// Page 1 (items 4, 5, 6)
	_, text1 := BuildPaginatedNotesKeyboard("SEN209", notesList, 1, 3)
	if !strings.Contains(text1, "Page 2 of 3") {
		t.Errorf("Expected Page 2 of 3 in text, got: %s", text1)
	}

	// Page 2 (item 7)
	_, text2 := BuildPaginatedNotesKeyboard("SEN209", notesList, 2, 3)
	if !strings.Contains(text2, "Page 3 of 3") {
		t.Errorf("Expected Page 3 of 3 in text, got: %s", text2)
	}
}
