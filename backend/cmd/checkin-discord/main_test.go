package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func testConfig(t *testing.T, baseURL string) config {
	t.Helper()
	return config{
		feedItemsURL: mustURL(t, baseURL+"/api/feeds/7/items"),
		apiKey:       "test-api-key",
		webhookURL:   mustURL(t, baseURL+"/api/webhooks/123/test-token"),
		pollInterval: time.Millisecond,
		httpTimeout:  5 * time.Second,
	}
}

func canonicalItems() []feedItem {
	when := time.Date(2026, time.August, 4, 16, 30, 0, 0, time.UTC)
	return []feedItem{
		{ID: 1, CreatedAt: when, LogLevel: 0, UserID: 10, UserDisplayName: "Alex Rivera", Title: cardLinkedTitle, Message: docusignCompleteText},
		{ID: 2, CreatedAt: when.Add(time.Second), LogLevel: 2, UserID: 11, UserDisplayName: "Morgan Lee", Title: cardLinkedTitle, Message: docusignRequiredText},
		{ID: 3, CreatedAt: when.Add(2 * time.Second), LogLevel: 4, Title: cardNotLinkedTitle, Message: cardNotLinkedText},
	}
}

func TestValidateConfigRequiresSecureNarrowEndpoints(t *testing.T) {
	valid := config{
		feedItemsURL: mustURL(t, "https://mkr.cx/api/feeds/17/items"),
		apiKey:       "key",
		webhookURL:   mustURL(t, "https://discord.com/api/webhooks/123/token"),
		pollInterval: time.Second,
		httpTimeout:  time.Second,
	}
	if err := validateConfig(valid); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	tests := []config{
		func() config {
			value := valid
			value.feedItemsURL = mustURL(t, "http://mkr.cx/api/feeds/17/items")
			return value
		}(),
		func() config {
			value := valid
			value.feedItemsURL = mustURL(t, "https://mkr.cx/api/feeds/signin/items")
			return value
		}(),
		func() config { value := valid; value.apiKey = ""; return value }(),
		func() config {
			value := valid
			value.webhookURL = mustURL(t, "https://example.com/api/webhooks/123/token")
			return value
		}(),
		func() config {
			value := valid
			value.webhookURL = mustURL(t, "https://discord.com/api/webhooks/123/token?leak=1")
			return value
		}(),
	}
	for index, cfg := range tests {
		if err := validateConfig(cfg); err == nil {
			t.Fatalf("unsafe config %d was accepted", index)
		}
	}
}

func TestCanonicalEventAcceptsOnlyThreeExactOutcomes(t *testing.T) {
	items := canonicalItems()
	for index, item := range items {
		if _, ok := canonicalEvent(item); !ok {
			t.Fatalf("canonical item %d rejected", index)
		}
	}

	mutations := []feedItem{
		func() feedItem { value := items[0]; value.Message = "Allowed"; return value }(),
		func() feedItem { value := items[1]; value.LogLevel = 0; return value }(),
		func() feedItem { value := items[2]; value.UserID = 99; return value }(),
		func() feedItem { value := items[2]; value.PendingUserSpecifier = "opaque"; return value }(),
		func() feedItem { value := items[0]; value.UserDisplayName = ""; return value }(),
	}
	for index, item := range mutations {
		if _, ok := canonicalEvent(item); ok {
			t.Fatalf("non-canonical mutation %d accepted", index)
		}
	}
}

func TestRenderDashboardUsesLatestTenAndDisablesMentions(t *testing.T) {
	when := time.Date(2026, time.August, 4, 16, 30, 0, 0, time.UTC)
	events := make([]checkinEvent, 0, 12)
	for id := uint(1); id <= 12; id++ {
		events = append(events, checkinEvent{itemID: id, occurredAt: when.Add(time.Duration(id) * time.Second), outcome: outcomeComplete, displayName: "Name_*_ <@123>"})
	}
	trimmed, skipped := mergeCanonical(nil, func() []feedItem {
		items := make([]feedItem, 0, len(events))
		for _, event := range events {
			items = append(items, feedItem{ID: event.itemID, CreatedAt: event.occurredAt, LogLevel: 0, UserID: event.itemID, UserDisplayName: event.displayName, Title: cardLinkedTitle, Message: docusignCompleteText})
		}
		return items
	}(), 0)
	if skipped != 0 || len(trimmed) != 10 || trimmed[0].itemID != 3 || trimmed[9].itemID != 12 {
		t.Fatalf("rolling window = %+v, skipped=%d", trimmed, skipped)
	}
	payload := renderDashboard(trimmed)
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if !strings.Contains(text, `"allowed_mentions":{"parse":[]}`) {
		t.Fatalf("mentions were not disabled: %s", text)
	}
	if strings.Contains(text, `"UserID"`) || strings.Contains(text, "Name_*_") {
		t.Fatalf("internal fields or unescaped display text leaked: %s", text)
	}
	if !strings.Contains(payload.Embeds[0].Description, `Name\_\*\_ \<@123\>`) {
		t.Fatalf("display name was not escaped: %s", payload.Embeds[0].Description)
	}
}

func TestPollOnceCreatesRollingMessageAndAdvancesAfterSuccess(t *testing.T) {
	items := canonicalItems()
	var discordPayloadReceived discordPayload
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/feeds/7/items":
			if request.Header.Get("Authorization") != "API-Key test-api-key" {
				response.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(response).Encode(feedResponse{Data: items})
		case request.URL.Path == "/api/webhooks/123/test-token" && request.Method == http.MethodPost:
			if request.URL.Query().Get("wait") != "true" {
				t.Error("Discord create did not request a response")
			}
			if err := json.NewDecoder(request.Body).Decode(&discordPayloadReceived); err != nil {
				t.Error(err)
			}
			_ = json.NewEncoder(response).Encode(map[string]string{"id": "555"})
		default:
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	worker := newSubscriber(testConfig(t, server.URL), server.Client(), log.New(io.Discard, "", 0))
	skipped, err := worker.pollOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || worker.state.cursor != 3 || worker.state.messageID != "555" || len(worker.state.events) != 3 {
		t.Fatalf("unexpected state: %+v, skipped=%d", worker.state, skipped)
	}
	if len(discordPayloadReceived.AllowedMentions.Parse) != 0 || len(discordPayloadReceived.Embeds) != 1 {
		t.Fatalf("unsafe Discord payload: %+v", discordPayloadReceived)
	}
}

func TestRateLimitKeepsCursorThenDeletedMessageIsRecreated(t *testing.T) {
	items := canonicalItems()[:1]
	var mutex sync.Mutex
	patches := 0
	posts := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/feeds/7/items":
			_ = json.NewEncoder(response).Encode(feedResponse{Data: items})
		case request.URL.Path == "/api/webhooks/123/test-token/messages/111" && request.Method == http.MethodPatch:
			mutex.Lock()
			patches++
			attempt := patches
			mutex.Unlock()
			if attempt == 1 {
				response.Header().Set("Retry-After", "1.25")
				response.WriteHeader(http.StatusTooManyRequests)
				return
			}
			response.WriteHeader(http.StatusNotFound)
		case request.URL.Path == "/api/webhooks/123/test-token" && request.Method == http.MethodPost:
			mutex.Lock()
			posts++
			mutex.Unlock()
			_ = json.NewEncoder(response).Encode(map[string]string{"id": "222"})
		default:
			response.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	worker := newSubscriber(testConfig(t, server.URL), server.Client(), log.New(io.Discard, "", 0))
	worker.state.messageID = "111"
	_, err := worker.pollOnce(context.Background())
	var retry *retryAfterError
	if !errors.As(err, &retry) || retry.after != 1250*time.Millisecond {
		t.Fatalf("rate limit error = %v, retry=%v", err, retry)
	}
	if worker.state.cursor != 0 || worker.state.messageID != "111" || len(worker.state.events) != 0 {
		t.Fatalf("state advanced after failed Discord update: %+v", worker.state)
	}

	if _, err := worker.pollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if worker.state.cursor != 1 || worker.state.messageID != "222" || posts != 1 {
		t.Fatalf("deleted message was not recreated: %+v posts=%d", worker.state, posts)
	}
}

func TestRedirectsAreNotFollowedForFeedOrDiscord(t *testing.T) {
	var leakHits int
	mode := "feed"
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/leak" {
			leakHits++
			response.WriteHeader(http.StatusNoContent)
			return
		}
		if request.URL.Path == "/api/feeds/7/items" {
			if mode == "feed" {
				http.Redirect(response, request, "/leak", http.StatusFound)
				return
			}
			_ = json.NewEncoder(response).Encode(feedResponse{Data: canonicalItems()[:1]})
			return
		}
		if strings.HasPrefix(request.URL.Path, "/api/webhooks/") {
			http.Redirect(response, request, "/leak", http.StatusTemporaryRedirect)
			return
		}
		response.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	worker := newSubscriber(testConfig(t, server.URL), server.Client(), log.New(io.Discard, "", 0))
	if _, err := worker.pollOnce(context.Background()); err == nil || leakHits != 0 {
		t.Fatalf("feed redirect followed or accepted: err=%v leakHits=%d", err, leakHits)
	}
	mode = "discord"
	if _, err := worker.pollOnce(context.Background()); err == nil || leakHits != 0 {
		t.Fatalf("Discord redirect followed or accepted: err=%v leakHits=%d", err, leakHits)
	}
}

func TestErrorsDoNotExposeSecretsBodiesOrDisplayNames(t *testing.T) {
	mode := "feed"
	privateBody := "0011223344556677 Private Person"
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/feeds/7/items" {
			if mode == "feed" {
				response.WriteHeader(http.StatusInternalServerError)
				_, _ = response.Write([]byte(privateBody))
				return
			}
			item := canonicalItems()[0]
			item.UserDisplayName = "Private Person"
			_ = json.NewEncoder(response).Encode(feedResponse{Data: []feedItem{item}})
			return
		}
		if strings.HasPrefix(request.URL.Path, "/api/webhooks/") {
			response.WriteHeader(http.StatusInternalServerError)
			_, _ = response.Write([]byte(privateBody))
			return
		}
		response.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	worker := newSubscriber(testConfig(t, server.URL), server.Client(), log.New(io.Discard, "", 0))
	for _, stage := range []string{"feed", "discord"} {
		mode = stage
		_, err := worker.pollOnce(context.Background())
		if err == nil {
			t.Fatalf("%s error was not returned", stage)
		}
		message := err.Error()
		for _, private := range []string{"test-api-key", "test-token", privateBody, "Private Person", "0011223344556677"} {
			if strings.Contains(message, private) {
				t.Fatalf("%s error exposed private data", stage)
			}
		}
	}
}
