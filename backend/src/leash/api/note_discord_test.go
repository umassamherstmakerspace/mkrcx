package leash_backend_api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mkrcx/mkrcx/src/shared/models"
)

func TestParseNoteDiscordWebhookIsStrict(t *testing.T) {
	valid := "https://discord.com/api/webhooks/123/test-token"
	if _, err := parseNoteDiscordWebhook(valid); err != nil {
		t.Fatalf("valid webhook rejected: %v", err)
	}
	for _, candidate := range []string{
		"http://discord.com/api/webhooks/123/token",
		"https://example.com/api/webhooks/123/token",
		"https://discord.com.evil.example/api/webhooks/123/token",
		"https://discord.com/api/webhooks/123/token?leak=1",
		"https://user:pass@discord.com/api/webhooks/123/token",
	} {
		if _, err := parseNoteDiscordWebhook(candidate); err == nil {
			t.Fatalf("unsafe webhook accepted: %s", candidate)
		}
	}
}

func TestRenderNoteDiscordPayloadDisablesMentionsAndEscapesMarkdown(t *testing.T) {
	note := models.NoteSubmission{
		ID: 42, CreatedAt: time.Date(2026, 9, 1, 12, 30, 0, 0, time.UTC),
		SubmitterName: "Student *Name*", SubmitterEmail: "student@example.edu",
		Content: "@everyone **look** <@123> https://example.edu",
	}
	payload := renderNoteDiscordPayload(note)
	if len(payload.AllowedMentions.Parse) != 0 || len(payload.Embeds) != 1 {
		t.Fatalf("unsafe payload settings: %+v", payload)
	}
	embed := payload.Embeds[0]
	if strings.Contains(embed.Description, "**look**") || !strings.Contains(embed.Description, `\*\*look\*\*`) {
		t.Fatalf("markdown was not escaped: %q", embed.Description)
	}
	if !strings.Contains(embed.Footer.Text, "#42") || embed.Timestamp != note.CreatedAt.Format(time.RFC3339) {
		t.Fatalf("missing durable identity: %+v", embed)
	}
}

func TestNoteDiscordDeliveryMarksAcceptedMessage(t *testing.T) {
	db := noteTestDatabase(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	note := models.NoteSubmission{
		CreatedAt: now, UpdatedAt: now, SubmittedBy: 1, SubmitterName: "Student",
		SubmitterEmail: "student@example.edu", Content: "A test note", IdempotencyKey: "discord-test",
		DiscordNextAttemptAt: now,
	}
	if err := db.Create(&note).Error; err != nil {
		t.Fatal(err)
	}

	var received noteDiscordPayload
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Query().Get("wait") != "true" {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.String())
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"id":"987654321"}`))
	}))
	defer server.Close()
	endpoint, _ := url.Parse(server.URL)
	worker := &noteDiscordWorker{
		db: db, webhookURL: endpoint, httpClient: server.Client(), pollInterval: time.Second,
		now: func() time.Time { return now }, logger: log.New(io.Discard, "", 0),
	}
	claimed, err := worker.claimNext(context.Background())
	if err != nil || claimed == nil {
		t.Fatalf("claim failed: note=%v err=%v", claimed, err)
	}
	if err := worker.deliver(context.Background(), *claimed); err != nil {
		t.Fatal(err)
	}
	var stored models.NoteSubmission
	if err := db.First(&stored, note.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DiscordDeliveredAt == nil || stored.DiscordMessageID != "987654321" || stored.DiscordLastStatus != http.StatusOK {
		t.Fatalf("delivery was not recorded: %+v", stored)
	}
	if len(received.Embeds) != 1 || received.Embeds[0].Description != note.Content {
		t.Fatalf("unexpected webhook payload: %+v", received)
	}
}

func TestNoteDiscordRateLimitIsDurablyRescheduled(t *testing.T) {
	db := noteTestDatabase(t)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	note := models.NoteSubmission{
		CreatedAt: now, UpdatedAt: now, SubmittedBy: 1, SubmitterName: "Student",
		SubmitterEmail: "student@example.edu", Content: "A test note", IdempotencyKey: "discord-rate-limit",
		DiscordNextAttemptAt: now,
	}
	if err := db.Create(&note).Error; err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusTooManyRequests)
		_, _ = response.Write([]byte(`{"retry_after":1.5}`))
	}))
	defer server.Close()
	endpoint, _ := url.Parse(server.URL)
	worker := &noteDiscordWorker{
		db: db, webhookURL: endpoint, httpClient: server.Client(), pollInterval: time.Second,
		now: func() time.Time { return now }, logger: log.New(io.Discard, "", 0),
	}
	claimed, err := worker.claimNext(context.Background())
	if err != nil || claimed == nil {
		t.Fatalf("claim failed: note=%v err=%v", claimed, err)
	}
	if err := worker.deliver(context.Background(), *claimed); err != nil {
		t.Fatal(err)
	}
	var stored models.NoteSubmission
	if err := db.First(&stored, note.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DiscordDeliveredAt != nil || stored.DiscordAttemptCount != 1 || stored.DiscordLastStatus != http.StatusTooManyRequests {
		t.Fatalf("rate limit state mismatch: %+v", stored)
	}
	if stored.DiscordNextAttemptAt.Before(now.Add(1500*time.Millisecond)) || stored.DiscordClaimedUntil != nil || stored.DiscordClaimToken != "" {
		t.Fatalf("retry was not durably scheduled: %+v", stored)
	}
}
