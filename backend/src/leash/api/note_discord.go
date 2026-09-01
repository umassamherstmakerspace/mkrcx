package leash_backend_api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const (
	noteDiscordPollInterval = 15 * time.Second
	noteDiscordHTTPTimeout  = 10 * time.Second
	noteDiscordClaimTime    = 45 * time.Second
	noteDiscordBatchSize    = 10
	noteDiscordErrorBodyMax = 16 * 1024
	noteDiscordColor        = 0x881C1C
)

var noteDiscordWebhookPath = regexp.MustCompile(`^/api/webhooks/[0-9]+/[^/]+/?$`)

type noteDiscordAllowedMentions struct {
	Parse []string `json:"parse"`
}

type noteDiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type noteDiscordEmbedFooter struct {
	Text string `json:"text"`
}

type noteDiscordEmbed struct {
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Color       int                     `json:"color"`
	Fields      []noteDiscordEmbedField `json:"fields"`
	Footer      noteDiscordEmbedFooter  `json:"footer"`
	Timestamp   string                  `json:"timestamp"`
}

type noteDiscordPayload struct {
	Username        string                     `json:"username"`
	AllowedMentions noteDiscordAllowedMentions `json:"allowed_mentions"`
	Embeds          []noteDiscordEmbed         `json:"embeds"`
}

type noteDiscordResponse struct {
	ID string `json:"id"`
}

type noteDiscordRateLimitResponse struct {
	RetryAfter float64 `json:"retry_after"`
}

type noteDiscordWorker struct {
	db           *gorm.DB
	webhookURL   *url.URL
	httpClient   *http.Client
	pollInterval time.Duration
	now          func() time.Time
	logger       *log.Logger
}

func parseNoteDiscordWebhook(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, errors.New("DISCORD_NOTES_WEBHOOK_URL is invalid")
	}
	if parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "discord.com") ||
		(parsed.Port() != "" && parsed.Port() != "443") || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.Fragment != "" || !noteDiscordWebhookPath.MatchString(parsed.EscapedPath()) {
		return nil, errors.New("DISCORD_NOTES_WEBHOOK_URL must be an HTTPS discord.com /api/webhooks/{id}/{token} URL without query parameters")
	}
	return parsed, nil
}

func escapeNoteDiscordText(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`*`, `\*`,
		`_`, `\_`,
		`~`, `\~`,
		"`", "\\`",
		`|`, `\|`,
		`>`, `\>`,
		`#`, `\#`,
		`[`, `\[`,
		`]`, `\]`,
		`<`, `\<`,
	)
	return replacer.Replace(value)
}

func truncateNoteDiscordText(value string, maximum int) string {
	if utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	if maximum <= 1 {
		return string(runes[:maximum])
	}
	return string(runes[:maximum-1]) + "…"
}

func renderNoteDiscordPayload(note models.NoteSubmission) noteDiscordPayload {
	name := strings.TrimSpace(note.SubmitterName)
	if name == "" {
		name = "Makerspace user"
	}
	identity := escapeNoteDiscordText(name)
	if email := strings.TrimSpace(note.SubmitterEmail); email != "" {
		identity += "\n" + escapeNoteDiscordText(email)
	}
	return noteDiscordPayload{
		Username:        "mkr.cx Notes",
		AllowedMentions: noteDiscordAllowedMentions{Parse: []string{}},
		Embeds: []noteDiscordEmbed{{
			Title:       "New Makerspace note",
			Description: truncateNoteDiscordText(escapeNoteDiscordText(note.Content), 4000),
			Color:       noteDiscordColor,
			Fields: []noteDiscordEmbedField{{
				Name:   "From",
				Value:  truncateNoteDiscordText(identity, 1000),
				Inline: false,
			}},
			Footer:    noteDiscordEmbedFooter{Text: fmt.Sprintf("mkr.cx note #%d", note.ID)},
			Timestamp: note.CreatedAt.UTC().Format(time.RFC3339),
		}},
	}
}

func (worker *noteDiscordWorker) claimNext(ctx context.Context) (*models.NoteSubmission, error) {
	for attempts := 0; attempts < 3; attempts++ {
		now := worker.now().UTC()
		var candidate models.NoteSubmission
		result := worker.db.WithContext(ctx).
			Where("discord_delivered_at IS NULL").
			Where("discord_next_attempt_at <= ?", now).
			Where("discord_claimed_until IS NULL OR discord_claimed_until < ?", now).
			Order("id ASC").First(&candidate)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		if result.Error != nil {
			return nil, result.Error
		}

		token := uuid.NewString()
		claimUntil := now.Add(noteDiscordClaimTime)
		updated := worker.db.WithContext(ctx).Model(&models.NoteSubmission{}).
			Where("id = ? AND discord_delivered_at IS NULL", candidate.ID).
			Where("discord_next_attempt_at <= ?", now).
			Where("discord_claimed_until IS NULL OR discord_claimed_until < ?", now).
			Updates(map[string]interface{}{
				"discord_claim_token":   token,
				"discord_claimed_until": claimUntil,
			})
		if updated.Error != nil {
			return nil, updated.Error
		}
		if updated.RowsAffected == 0 {
			continue
		}
		candidate.DiscordClaimToken = token
		candidate.DiscordClaimedUntil = &claimUntil
		return &candidate, nil
	}
	return nil, nil
}

func noteDiscordBackoff(attempt uint) time.Duration {
	exponent := math.Min(float64(attempt), 8)
	delay := time.Duration(math.Pow(2, exponent)) * 15 * time.Second
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}

func (worker *noteDiscordWorker) markFailed(ctx context.Context, note models.NoteSubmission, status int, retryAfter time.Duration) error {
	attempt := note.DiscordAttemptCount + 1
	if retryAfter <= 0 {
		retryAfter = noteDiscordBackoff(attempt)
	}
	if retryAfter > time.Hour {
		retryAfter = time.Hour
	}
	next := worker.now().UTC().Add(retryAfter)
	return worker.db.WithContext(ctx).Model(&models.NoteSubmission{}).
		Where("id = ? AND discord_claim_token = ? AND discord_delivered_at IS NULL", note.ID, note.DiscordClaimToken).
		Updates(map[string]interface{}{
			"discord_attempt_count":   attempt,
			"discord_next_attempt_at": next,
			"discord_claim_token":     "",
			"discord_claimed_until":   nil,
			"discord_last_status":     status,
		}).Error
}

func (worker *noteDiscordWorker) markDelivered(ctx context.Context, note models.NoteSubmission, messageID string, status int) error {
	now := worker.now().UTC()
	return worker.db.WithContext(ctx).Model(&models.NoteSubmission{}).
		Where("id = ? AND discord_claim_token = ? AND discord_delivered_at IS NULL", note.ID, note.DiscordClaimToken).
		Updates(map[string]interface{}{
			"discord_delivered_at":  now,
			"discord_message_id":    messageID,
			"discord_attempt_count": note.DiscordAttemptCount + 1,
			"discord_claim_token":   "",
			"discord_claimed_until": nil,
			"discord_last_status":   status,
		}).Error
}

func discordRetryAfter(response *http.Response, body []byte) time.Duration {
	var payload noteDiscordRateLimitResponse
	if json.Unmarshal(body, &payload) == nil && payload.RetryAfter > 0 {
		return time.Duration(math.Ceil(payload.RetryAfter*1000)) * time.Millisecond
	}
	if header := strings.TrimSpace(response.Header.Get("Retry-After")); header != "" {
		if seconds, err := time.ParseDuration(header + "s"); err == nil {
			return seconds
		}
	}
	return 15 * time.Second
}

func (worker *noteDiscordWorker) deliver(ctx context.Context, note models.NoteSubmission) error {
	payload, err := json.Marshal(renderNoteDiscordPayload(note))
	if err != nil {
		return worker.markFailed(ctx, note, 0, 0)
	}
	endpoint := *worker.webhookURL
	query := endpoint.Query()
	query.Set("wait", "true")
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return worker.markFailed(ctx, note, 0, 0)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "mkrcx-note-delivery/1")
	response, err := worker.httpClient.Do(request)
	if err != nil {
		if markErr := worker.markFailed(ctx, note, 0, 0); markErr != nil {
			return markErr
		}
		return nil
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, noteDiscordErrorBodyMax))

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		var result noteDiscordResponse
		_ = json.Unmarshal(body, &result)
		return worker.markDelivered(ctx, note, result.ID, response.StatusCode)
	}
	retryAfter := time.Duration(0)
	if response.StatusCode == http.StatusTooManyRequests {
		retryAfter = discordRetryAfter(response, body)
	} else if response.StatusCode >= 400 && response.StatusCode < 500 {
		retryAfter = 15 * time.Minute
	}
	return worker.markFailed(ctx, note, response.StatusCode, retryAfter)
}

func (worker *noteDiscordWorker) drain(ctx context.Context) error {
	for delivered := 0; delivered < noteDiscordBatchSize; delivered++ {
		note, err := worker.claimNext(ctx)
		if err != nil {
			return err
		}
		if note == nil {
			return nil
		}
		if err := worker.deliver(ctx, *note); err != nil {
			return err
		}
	}
	return nil
}

func (worker *noteDiscordWorker) run(ctx context.Context) {
	ticker := time.NewTicker(worker.pollInterval)
	defer ticker.Stop()
	for {
		if err := worker.drain(ctx); err != nil && !errors.Is(err, context.Canceled) {
			worker.logger.Printf("note Discord delivery pass failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// StartNoteDiscordDelivery begins the durable Discord outbox worker. An empty
// webhook disables delivery without disabling note storage, which keeps local
// development and staged configuration fail-safe.
func StartNoteDiscordDelivery(db *gorm.DB, rawWebhook string) (func(), error) {
	if strings.TrimSpace(rawWebhook) == "" {
		log.Println("Note Discord delivery disabled; DISCORD_NOTES_WEBHOOK_URL is not set")
		return func() {}, nil
	}
	webhookURL, err := parseNoteDiscordWebhook(rawWebhook)
	if err != nil {
		return nil, err
	}
	worker := &noteDiscordWorker{
		db:           db,
		webhookURL:   webhookURL,
		httpClient:   &http.Client{Timeout: noteDiscordHTTPTimeout},
		pollInterval: noteDiscordPollInterval,
		now:          time.Now,
		logger:       log.Default(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wait sync.WaitGroup
	wait.Add(1)
	go func() {
		defer wait.Done()
		worker.run(ctx)
	}()
	log.Println("Note Discord delivery enabled")
	return func() {
		cancel()
		wait.Wait()
	}, nil
}
