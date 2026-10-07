package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"mtuunibot/internal/constants"
)

type NoteMetadata struct {
	CourseCode string `json:"course_code"`
	Title      string `json:"title"`
	College    string `json:"college"`
	Department string `json:"department"`
	Level      string `json:"level"`
	Tags       string `json:"tags"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqChatRequest struct {
	Model          string            `json:"model"`
	Messages       []groqMessage     `json:"messages"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

type groqChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// AnalyzeDocumentWithGroq extracts course metadata and search tags from document excerpt using Groq.
func AnalyzeDocumentWithGroq(apiKey, fileName, textSnippet string) (*NoteMetadata, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("groq API key is empty")
	}

	collegesStr := strings.Join(constants.GetAllColleges(), ", ")
	departmentsStr := strings.Join(constants.GetAllDepartments(), ", ")
	levelsStr := strings.Join(constants.GetAllLevelsFormatted(), ", ")

	systemPrompt := fmt.Sprintf(`You are an academic document parser for Mountain Top University.
Analyze the document filename and excerpt.
You must match the document against the university's official curriculum.

Valid Colleges: %s
Valid Departments: %s
Valid Levels: %s

CRITICAL RULES:
1. "department" MUST be selected from the Valid Departments list above. If the document is irrelevant, general chatter, not academic, or does NOT belong to any of these departments, set "department": "NONE" and "college": "NONE".
2. "college" MUST be the college corresponding to the chosen department (or "NONE" if discarded).
3. "level" should be one of the Valid Levels (or "NONE" if unknown).
4. "tags" must be 5-10 comma-separated lowercase topic keywords for search.

Return ONLY valid JSON matching this schema:
{
  "course_code": "e.g. CSC201 or NONE",
  "title": "Clear concise topic or course title",
  "college": "%s or NONE",
  "department": "One of the Valid Departments or NONE",
  "level": "One of the Valid Levels or NONE",
  "tags": "topic keywords"
}`, collegesStr, departmentsStr, levelsStr, collegesStr)

	userPrompt := fmt.Sprintf("File Name: %s\n\nDocument Excerpt:\n%s", fileName, textSnippet)

	// Available working models on current Groq account
	activeModels := []string{"openai/gpt-oss-20b"}
	var lastErr error

	for _, modelName := range activeModels {
		reqBody := groqChatRequest{
			Model: modelName,
			Messages: []groqMessage{
				{Role: "system", Content: systemPrompt},
				{Role: "user", Content: userPrompt},
			},
			ResponseFormat: map[string]string{"type": "json_object"},
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed calling Groq model %s: %w", modelName, err)
			continue
		}

		respBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed reading response body: %w", err)
			continue
		}

		var chatResp groqChatResponse
		if err := json.Unmarshal(respBytes, &chatResp); err != nil {
			lastErr = fmt.Errorf("failed parsing Groq response JSON: %w (raw: %s)", err, string(respBytes))
			continue
		}

		if chatResp.Error != nil && chatResp.Error.Message != "" {
			lastErr = fmt.Errorf("groq API error (%s): %s", modelName, chatResp.Error.Message)
			continue
		}

		if len(chatResp.Choices) == 0 {
			lastErr = fmt.Errorf("no response choices from Groq model %s", modelName)
			continue
		}

		rawContent := strings.TrimSpace(chatResp.Choices[0].Message.Content)
		rawContent = strings.TrimPrefix(rawContent, "```json")
		rawContent = strings.TrimPrefix(rawContent, "```")
		rawContent = strings.TrimSuffix(rawContent, "```")
		rawContent = strings.TrimSpace(rawContent)

		var meta NoteMetadata
		if err := json.Unmarshal([]byte(rawContent), &meta); err != nil {
			lastErr = fmt.Errorf("failed unmarshaling metadata JSON: %w (content: %s)", err, rawContent)
			continue
		}

		return &meta, nil
	}

	return nil, lastErr
}
