package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"splitter/internal/config"
	"splitter/internal/models"
	"splitter/internal/repository"
)

const geminiUrl = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=%s"

// --- Gemini Structs ---
type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig map[string]any  `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// --- OpenAI Structs ---
type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AskOpenAI calls the standard OpenAI API with a prompt
func AskOpenAI(apiKey, prompt string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("missing API key")
	}

	reqBody := openAIRequest{
		Model: "gpt-4o-mini", // Cost-effective model for bots
		Messages: []openAIMessage{
			{Role: "system", Content: "You are 'Split', a helpful, fun, and concise AI reply bot on a social media app called Splitter. Please answer the following prompt in 1-3 short sentences. Make it engaging."},
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		return strings.TrimSpace(result.Choices[0].Message.Content), nil
	}

	return "I couldn't process that request, sorry!", nil
}

// AskGemini calls the Google Gemini API with a prompt
func AskGemini(apiKey, prompt string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("missing API key")
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: map[string]any{
			"temperature":     0.8,
			"maxOutputTokens": 150,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf(geminiUrl, apiKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
	}

	return "I couldn't process that request, sorry!", nil
}

// CheckAndHandleSplitBot triggers the AI in the background if @split is mentioned.
func CheckAndHandleSplitBot(originalContent, postID string, parentID *string, cfg *config.Config, replyRepo *repository.ReplyRepository) {
	// Only trigger if @split is case-insensitively found
	if !strings.Contains(strings.ToLower(originalContent), "@split") {
		return
	}

	apiKey := cfg.Bot.ApiKey
	if apiKey == "" {
		log.Println("[SplitBot] Mention detected but SPLIT_BOT_API_KEY is not configured")
		return
	}

	log.Printf("[SplitBot] Mention detected! Triggering AI... (PostID: %s)", postID)

	// Remove @split from the text for a cleaner prompt
	re := regexp.MustCompile(`(?i)@split\b`)
	promptText := strings.TrimSpace(re.ReplaceAllString(originalContent, ""))

	if promptText == "" {
		promptText = "The user mentioned you without saying anything else. Greet them."
	}

	var replyStr string
	var err error

	// Automatically decide between OpenAI and Gemini
	if strings.HasPrefix(apiKey, "sk-") {
		// It's an OpenAI key (sk-...)
		log.Printf("[SplitBot] Detected OpenAI key, calling GPT-4o-mini...")
		replyStr, err = AskOpenAI(apiKey, promptText)
	} else {
		// Fallback to Gemini
		systemPrompt := "You are 'Split', a helpful, fun, and concise AI reply bot on a social media app called Splitter. Please answer the following prompt in 1-3 short sentences. Make it engaging. Prompt: " + promptText
		
		log.Printf("[SplitBot] Calling Gemini with key prefix: %s...", apiKey[:8])
		replyStr, err = AskGemini(apiKey, systemPrompt)
	}

	if err != nil {
		log.Printf("[SplitBot] ERROR - AI call failed: %v", err)
		replyStr = fmt.Sprintf("⚠️ SplitBot error: %v", err)
	} else {
		log.Printf("[SplitBot] AI responded successfully: %s", replyStr[:min(len(replyStr), 50)])
	}

	replyCreate := &models.ReplyCreate{
		PostID:   postID,
		ParentID: parentID,
		Content:  replyStr,
	}

	authorDID := "did:key:bot_split"

	ctx := context.Background()
	_, err = replyRepo.Create(ctx, authorDID, replyCreate, 1)
	if err != nil {
		log.Printf("[SplitBot] Failed to save AI reply to DB: %v", err)
		return
	}
	log.Printf("[SplitBot] Successfully responded to post/reply %s", postID)
}
