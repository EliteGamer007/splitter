package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/models"
	"splitter/internal/repository"
)

const splitBotDID = "did:key:bot_split"
const defaultGeminiModel = "gemini-3.5-flash-lite"
const botSystemPrompt = "You are Split, a friendly and occasionally playful reply bot on Splitter. Answer the public mention in 1-3 short sentences, at most 500 bytes. Do not invent account actions or private information."
const maxAIResponse = 64 << 10

var splitMention = regexp.MustCompile(`(?i)(^|[\s(])@split\b`)
var modelName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:/-]{0,127}$`)
var splitBotSlot = make(chan struct{}, 1)

// SplitBotCapabilities exposes configuration and finite local budgets, never credentials.
type SplitBotCapabilities struct {
	Enabled           bool   `json:"enabled"`
	Reason            string `json:"reason,omitempty"`
	Provider          string `json:"provider,omitempty"`
	Model             string `json:"model,omitempty"`
	DailyLimit        int    `json:"daily_limit"`
	PerUserDailyLimit int    `json:"per_user_daily_limit"`
}

func botLimit(name string, fallback, maximum int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}
	if value > maximum {
		return maximum
	}
	return value
}
func SplitBotCapability(cfg *config.Config) SplitBotCapabilities {
	c := SplitBotCapabilities{DailyLimit: botLimit("SPLIT_BOT_DAILY_LIMIT", 10, 100), PerUserDailyLimit: botLimit("SPLIT_BOT_USER_DAILY_LIMIT", 3, 10)}
	if cfg == nil || cfg.Bot.ApiKey == "" {
		c.Reason = "Configure an AI provider key to enable @split replies"
		return c
	}
	c.Provider = strings.ToLower(strings.TrimSpace(os.Getenv("SPLIT_BOT_PROVIDER")))
	if c.Provider == "" {
		switch {
		case strings.HasPrefix(cfg.Bot.ApiKey, "sk-"):
			c.Provider = "openai"
		case strings.HasPrefix(cfg.Bot.ApiKey, "gsk_"):
			c.Provider = "groq"
		case strings.HasPrefix(cfg.Bot.ApiKey, "http"):
			c.Provider = "ollama"
		default:
			c.Provider = "gemini"
		}
	}
	c.Model = strings.TrimSpace(os.Getenv("SPLIT_BOT_MODEL"))
	if c.Model == "" {
		switch c.Provider {
		case "gemini":
			c.Model = defaultGeminiModel
		case "openai":
			c.Model = "gpt-4o-mini"
		case "groq":
			c.Model = "qwen/qwen3.8-27b"
		case "ollama":
			c.Model = "llama3.2"
		}
	}
	switch c.Provider {
	case "gemini", "openai", "groq", "ollama":
	default:
		c.Reason = "Unsupported AI provider; configure gemini, openai, groq, or ollama"
		return c
	}
	if !modelName.MatchString(c.Model) || strings.HasPrefix(c.Model, "gemini-1.") || strings.HasPrefix(c.Model, "gemini-2.0") {
		c.Reason = "Configure a supported current AI model"
		return c
	}
	if c.Provider == "ollama" {
		endpoint := os.Getenv("SPLIT_BOT_ENDPOINT")
		if endpoint == "" {
			endpoint = cfg.Bot.ApiKey
		}
		if !validProviderEndpoint(endpoint) {
			c.Reason = "Configure a valid HTTPS or loopback Ollama endpoint"
			return c
		}
	}
	c.Enabled = true
	return c
}

func validProviderEndpoint(endpoint string) bool {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	return u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")
}
func providerJSON(ctx context.Context, endpoint, credentialHeader, credential string, payload any) ([]byte, error) {
	if !validProviderEndpoint(endpoint) {
		return nil, errors.New("invalid AI provider endpoint")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("cannot prepare AI request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("cannot prepare AI request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(credentialHeader, credential)
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("AI provider is temporarily unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI provider returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAIResponse+1))
	if err != nil || len(data) > maxAIResponse {
		return nil, errors.New("AI provider response exceeds its limit or is unreadable")
	}
	return data, nil
}

// AskOpenAI retains the existing OpenAI-compatible API for configured cloud/local providers.
func AskOpenAI(apiKey, prompt, endpoint, model string) (string, error) {
	return askOpenAI(context.Background(), apiKey, prompt, endpoint, model)
}
func askOpenAI(ctx context.Context, apiKey, prompt, endpoint, model string) (string, error) {
	if apiKey == "" || !modelName.MatchString(model) {
		return "", errors.New("missing AI key or supported model")
	}
	payload := map[string]any{"model": model, "messages": []map[string]string{{"role": "system", "content": botSystemPrompt}, {"role": "user", "content": prompt}}, "temperature": 0.9, "max_tokens": 150}
	data, err := providerJSON(ctx, endpoint, "Authorization", "Bearer "+apiKey, payload)
	if err != nil {
		return "", err
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &result) != nil || len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return "", errors.New("AI provider returned no usable reply")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}
func AskGemini(apiKey, prompt string) (string, error) {
	model := strings.TrimSpace(os.Getenv("SPLIT_BOT_MODEL"))
	if model == "" {
		model = defaultGeminiModel
	}
	return askGemini(context.Background(), apiKey, prompt, model)
}
func askGemini(ctx context.Context, apiKey, prompt, model string) (string, error) {
	if apiKey == "" || !modelName.MatchString(model) {
		return "", errors.New("missing AI key or supported model")
	}
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":generateContent"
	payload := map[string]any{"contents": []any{map[string]any{"parts": []any{map[string]string{"text": botSystemPrompt + " Public mention: " + prompt}}}}, "generationConfig": map[string]any{"temperature": 0.9, "maxOutputTokens": 150}}
	data, err := providerJSON(ctx, endpoint, "x-goog-api-key", apiKey, payload)
	if err != nil {
		return "", err
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
	if json.Unmarshal(data, &result) != nil || len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 || strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text) == "" {
		return "", errors.New("AI provider returned no usable reply")
	}
	return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
}

// CheckAndHandleSplitBot schedules at most one actual public mention. There is
// no idle worker, provider retry loop, seeded posting, or unbounded goroutine queue.
func CheckAndHandleSplitBot(content, postID string, parentID *string, cfg *config.Config, replyRepo *repository.ReplyRepository) {
	if !splitMention.MatchString(content) {
		return
	}
	capability := SplitBotCapability(cfg)
	if !capability.Enabled {
		return
	}
	generate := func(ctx context.Context, prompt string) (string, error) {
		switch capability.Provider {
		case "gemini":
			return askGemini(ctx, cfg.Bot.ApiKey, prompt, capability.Model)
		case "openai":
			return askOpenAI(ctx, cfg.Bot.ApiKey, prompt, "https://api.openai.com/v1/chat/completions", capability.Model)
		case "groq":
			return askOpenAI(ctx, cfg.Bot.ApiKey, prompt, "https://api.groq.com/openai/v1/chat/completions", capability.Model)
		default:
			endpoint := os.Getenv("SPLIT_BOT_ENDPOINT")
			if endpoint == "" {
				endpoint = cfg.Bot.ApiKey
			}
			return askOpenAI(ctx, "ollama", prompt, strings.TrimRight(endpoint, "/")+"/v1/chat/completions", capability.Model)
		}
	}
	worker := splitBotWorker{slot: splitBotSlot, store: databaseSplitBotStore{domain: cfg.Federation.Domain, appLimit: capability.DailyLimit, userLimit: capability.PerUserDailyLimit, replyRepo: replyRepo}, generate: generate}
	worker.try(content, postID, parentID)
}

type splitBotTarget struct {
	UserDID  string
	PostID   string
	ParentID *string
	Depth    int
}
type splitBotStore interface {
	Reserve(context.Context, string, *string) (splitBotTarget, error)
	Save(context.Context, splitBotTarget, string) error
}
type splitBotWorker struct {
	slot     chan struct{}
	store    splitBotStore
	generate func(context.Context, string) (string, error)
}

func (w *splitBotWorker) try(content, postID string, parentID *string) bool {
	if w.generate == nil || w.store == nil || !splitMention.MatchString(content) {
		return false
	}
	select {
	case w.slot <- struct{}{}:
	default:
		return false
	}
	if parentID != nil {
		copyID := *parentID
		parentID = &copyID
	}
	go func() {
		defer func() { <-w.slot }()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		target, err := w.store.Reserve(ctx, postID, parentID)
		if err != nil {
			return
		}
		prompt := strings.TrimSpace(splitMention.ReplaceAllString(content, "$1"))
		if prompt == "" {
			prompt = "The user mentioned you. Greet them briefly."
		}
		reply, err := w.generate(ctx, prompt)
		if err != nil {
			log.Print("[SplitBot] Provider reply unavailable; no reply published")
			return
		}
		reply = strings.TrimSpace(reply)
		if len(reply) > 500 {
			reply = reply[:500]
			for !utf8.ValidString(reply) {
				reply = reply[:len(reply)-1]
			}
		}
		if reply == "" {
			return
		}
		if err := w.store.Save(ctx, target, reply); err != nil {
			log.Print("[SplitBot] Reply could not be published")
		}
	}()
	return true
}

type databaseSplitBotStore struct {
	domain              string
	appLimit, userLimit int
	replyRepo           *repository.ReplyRepository
}

func (s databaseSplitBotStore) Reserve(ctx context.Context, postID string, parentID *string) (splitBotTarget, error) {
	target := splitBotTarget{PostID: postID, ParentID: parentID, Depth: 1}
	if db.GetDB() == nil {
		return target, errors.New("AI database unavailable")
	}
	allowed, err := repository.NewPostRepository().CanView(ctx, postID, "")
	if err != nil || !allowed {
		return target, errors.New("AI only processes public conversations")
	}
	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return target, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(827416017)); err != nil {
		return target, err
	}
	mentionID := "post:" + postID
	if parentID == nil {
		err = tx.QueryRow(ctx, `SELECT p.author_did FROM posts p JOIN users u ON u.did=p.author_did WHERE p.id=$1 AND p.visibility='public' AND p.deleted_at IS NULL AND (p.expires_at IS NULL OR p.expires_at>NOW()) AND NOT u.is_suspended`, postID).Scan(&target.UserDID)
	} else {
		mentionID = "reply:" + *parentID
		var parentDepth int
		var grandparent *string
		err = tx.QueryRow(ctx, `SELECT r.author_did,r.depth,r.parent_id FROM replies r JOIN posts p ON p.id=r.post_id JOIN users u ON u.did=r.author_did WHERE r.id=$1 AND r.post_id=$2 AND r.deleted_at IS NULL AND p.visibility='public' AND p.deleted_at IS NULL AND (p.expires_at IS NULL OR p.expires_at>NOW()) AND NOT u.is_suspended`, *parentID, postID).Scan(&target.UserDID, &parentDepth, &grandparent)
		target.Depth = parentDepth + 1
		if target.Depth > 3 {
			target.Depth = 3
			target.ParentID = grandparent
		}
	}
	if err != nil || target.UserDID == splitBotDID {
		return target, errors.New("AI mention is unavailable")
	}
	since := time.Now().UTC().Truncate(24 * time.Hour)
	var appCount, userCount int
	if err := tx.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE user_did=$1) FROM ai_bot_requests WHERE requested_at >= $2`, target.UserDID, since).Scan(&appCount, &userCount); err != nil {
		return target, err
	}
	if appCount >= s.appLimit || userCount >= s.userLimit {
		return target, errors.New("AI daily request budget reached")
	}
	if err := db.CheckGrowthBudget(ctx, tx, 2048); err != nil {
		return target, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ai_bot_requests(user_did,post_id,mention_id)VALUES($1,$2,$3)`, target.UserDID, postID, mentionID); err != nil {
		return target, err
	}
	var botUsername, hash string
	var suspended bool
	err = tx.QueryRow(ctx, `SELECT username,COALESCE(password_hash,''),COALESCE(is_suspended,false) FROM users WHERE did=$1`, splitBotDID).Scan(&botUsername, &hash, &suspended)
	if errors.Is(err, pgx.ErrNoRows) {
		var taken bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE username='split')").Scan(&taken); err != nil {
			return target, err
		}
		if taken {
			return target, errors.New("bot username belongs to an existing account")
		}
		domain := s.domain
		if domain == "" {
			domain = "localhost"
		}
		if _, err := tx.Exec(ctx, `INSERT INTO users(username,instance_domain,did,display_name,bio,public_key,role)VALUES('split',$1,$2,'Split','AI replies to public @split mentions','','user')`, domain, splitBotDID); err != nil {
			return target, err
		}
	} else if err != nil || botUsername != "split" || hash != "" || suspended {
		return target, errors.New("configured bot account is unavailable")
	}
	// Keep only the short-lived counter metadata; never remove user content.
	if _, err := tx.Exec(ctx, `DELETE FROM ai_bot_requests WHERE id IN(SELECT id FROM ai_bot_requests WHERE requested_at < NOW()-INTERVAL '7 days' ORDER BY requested_at LIMIT 100)`); err != nil {
		return target, err
	}
	if err := tx.Commit(ctx); err != nil {
		return target, err
	}
	return target, nil
}
func (s databaseSplitBotStore) Save(ctx context.Context, target splitBotTarget, content string) error {
	if s.replyRepo == nil {
		return errors.New("reply repository unavailable")
	}
	allowed, err := repository.NewPostRepository().CanView(ctx, target.PostID, "")
	if err != nil || !allowed {
		return errors.New("conversation is no longer public")
	}
	_, err = s.replyRepo.Create(ctx, splitBotDID, &models.ReplyCreate{PostID: target.PostID, ParentID: target.ParentID, Content: content}, target.Depth)
	return err
}
