package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type aiRoundTrip func(*http.Request) (*http.Response, error)

func (f aiRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fakeAITransport(t *testing.T, fn aiRoundTrip) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = fn
	t.Cleanup(func() { http.DefaultTransport = previous })
}
func aiResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
func TestGeminiUsesConfiguredCurrentModelAndHeaderCredential(t *testing.T) {
	t.Setenv("SPLIT_BOT_MODEL", "gemini-3.5-flash-lite")
	calls := 0
	fakeAITransport(t, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("key") != "" || r.Header.Get("x-goog-api-key") != "tiny-key" {
			t.Errorf("API credential must be in its dedicated header")
		}
		if r.URL.Path != "/v1beta/models/gemini-3.5-flash-lite:generateContent" {
			t.Errorf("obsolete/unconfigured Gemini endpoint %s", r.URL.Path)
		}
		return aiResponse(200, `{"candidates":[{"content":{"parts":[{"text":"A useful reply"}]}}]}`), nil
	})
	result, err := AskGemini("tiny-key", "public mention")
	if err != nil || result != "A useful reply" || calls != 1 {
		t.Fatalf("Gemini result %q err=%v calls=%d", result, err, calls)
	}
}
func TestAIProviderErrorsNeverExposeResponseBodiesOrCredentials(t *testing.T) {
	fakeAITransport(t, func(*http.Request) (*http.Response, error) {
		return aiResponse(401, "provider-secret credential-rejected private-content"), nil
	})
	for _, call := range []func() (string, error){func() (string, error) { return AskGemini("test-secret", "public mention") }, func() (string, error) {
		return AskOpenAI("test-secret", "public mention", "https://api.openai.com/v1/chat/completions", "gpt-4o-mini")
	}} {
		result, err := call()
		if err == nil || result != "" {
			t.Fatalf("provider failure treated as success: %q %v", result, err)
		}
		if strings.Contains(err.Error(), "provider-secret") || strings.Contains(err.Error(), "private-content") {
			t.Fatalf("provider response leaked: %v", err)
		}
	}
}
func TestAIResponseBodyIsBounded(t *testing.T) {
	fakeAITransport(t, func(*http.Request) (*http.Response, error) {
		return aiResponse(200, `{"choices":[{"message":{"content":"`+strings.Repeat("x", 128*1024)+`"}}]}`), nil
	})
	result, err := AskOpenAI("test-key", "public mention", "https://api.openai.com/v1/chat/completions", "gpt-4o-mini")
	if err == nil || result != "" {
		t.Fatalf("oversized provider response accepted (%d bytes)", len(result))
	}
}

type fakeBotStore struct {
	reserveErr error
	reserved   atomic.Int32
	saved      atomic.Int32
}

func (s *fakeBotStore) Reserve(context.Context, string, *string) (splitBotTarget, error) {
	s.reserved.Add(1)
	return splitBotTarget{UserDID: "did:alice", PostID: "public-post", Depth: 1}, s.reserveErr
}
func (s *fakeBotStore) Save(context.Context, splitBotTarget, string) error {
	s.saved.Add(1)
	return nil
}
func awaitBotIdle(t *testing.T, w *splitBotWorker) {
	t.Helper()
	select {
	case w.slot <- struct{}{}:
		<-w.slot
	case <-time.After(time.Second):
		t.Fatal("bot work did not finish")
	}
}
func TestSplitBotOnlyDispatchesExactMentionsWithConfiguredProvider(t *testing.T) {
	for _, content := range []string{"hello world", "hello @splitter", "alice@split.example"} {
		s := &fakeBotStore{}
		w := &splitBotWorker{slot: make(chan struct{}, 1), store: s, generate: func(context.Context, string) (string, error) { t.Error("non-mention reached provider"); return "", nil }}
		if w.try(content, "post", nil) {
			t.Errorf("non-mention dispatched: %q", content)
		}
	}
	s := &fakeBotStore{}
	w := &splitBotWorker{slot: make(chan struct{}, 1), store: s}
	if w.try("hello @split", "post", nil) {
		t.Fatal("unconfigured provider dispatched")
	}
}
func TestSplitBotConcurrencyOneAndErrorsNeverBecomeReplies(t *testing.T) {
	s := &fakeBotStore{}
	entered, release := make(chan struct{}), make(chan struct{})
	w := &splitBotWorker{slot: make(chan struct{}, 1), store: s, generate: func(context.Context, string) (string, error) {
		close(entered)
		<-release
		return "", errors.New("provider-secret failure")
	}}
	if !w.try("@split hello", "post", nil) {
		t.Fatal("valid mention did not dispatch")
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("provider not called")
	}
	if w.try("@split another", "post2", nil) {
		t.Error("parallel mention exceeded single request concurrency")
	}
	close(release)
	awaitBotIdle(t, w)
	if s.saved.Load() != 0 || s.reserved.Load() != 1 {
		t.Fatalf("error reply or extra reservation: saved=%d reserved=%d", s.saved.Load(), s.reserved.Load())
	}
}
func TestSplitBotPrivateConversationNeverReachesProvider(t *testing.T) {
	s := &fakeBotStore{reserveErr: errors.New("private conversation")}
	var calls atomic.Int32
	w := &splitBotWorker{slot: make(chan struct{}, 1), store: s, generate: func(context.Context, string) (string, error) { calls.Add(1); return "response", nil }}
	if !w.try("@split private mention", "post", nil) {
		t.Fatal("eligible mention not checked")
	}
	awaitBotIdle(t, w)
	if calls.Load() != 0 || s.saved.Load() != 0 {
		t.Fatal("private content was sent to the provider or a reply was saved")
	}
}
func TestSplitBotSuccessfulPublicMentionSavesOneReply(t *testing.T) {
	s := &fakeBotStore{}
	w := &splitBotWorker{slot: make(chan struct{}, 1), store: s, generate: func(_ context.Context, prompt string) (string, error) {
		if strings.Contains(prompt, "@split") {
			t.Errorf("mention marker not stripped: %q", prompt)
		}
		return "A helpful answer", nil
	}}
	if !w.try("@split explain this", "post", nil) {
		t.Fatal("valid mention did not dispatch")
	}
	awaitBotIdle(t, w)
	if s.saved.Load() != 1 {
		t.Fatalf("successful public mention saved %d replies", s.saved.Load())
	}
}
