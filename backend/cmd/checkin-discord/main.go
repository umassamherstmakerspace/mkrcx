package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

const (
	defaultPollInterval = 2 * time.Second
	defaultHTTPTimeout  = 10 * time.Second
	feedPageLimit       = 100
	rollingItemLimit    = 10
	maxResponseBytes    = 1 << 20

	cardLinkedTitle      = "Card linked"
	docusignCompleteText = "DocuSign complete"
	docusignRequiredText = "DocuSign required: ask them to complete the participation agreement."
	cardNotLinkedTitle   = "Card not linked"
	cardNotLinkedText    = "Check whether the user is registered; if so, direct them to link their UCard."
)

var (
	feedItemsPathPattern = regexp.MustCompile(`^/api/feeds/[1-9][0-9]*/items/?$`)
	webhookPathPattern   = regexp.MustCompile(`^/api/webhooks/[0-9]+/[^/]+/?$`)
	messageIDPattern     = regexp.MustCompile(`^[0-9]+$`)
)

type config struct {
	feedItemsURL *url.URL
	apiKey       string
	webhookURL   *url.URL
	pollInterval time.Duration
	httpTimeout  time.Duration
}

func loadConfig() (config, error) {
	pollInterval, err := durationEnv("CHECKIN_DISCORD_POLL_INTERVAL", defaultPollInterval)
	if err != nil {
		return config{}, err
	}
	httpTimeout, err := durationEnv("CHECKIN_DISCORD_HTTP_TIMEOUT", defaultHTTPTimeout)
	if err != nil {
		return config{}, err
	}

	feedItemsURL, err := url.Parse(strings.TrimSpace(os.Getenv("MKRCX_SIGNIN_FEED_ITEMS_URL")))
	if err != nil {
		return config{}, errors.New("MKRCX_SIGNIN_FEED_ITEMS_URL is invalid")
	}
	webhookURL, err := url.Parse(strings.TrimSpace(os.Getenv("DISCORD_SIGNIN_WEBHOOK_URL")))
	if err != nil {
		return config{}, errors.New("DISCORD_SIGNIN_WEBHOOK_URL is invalid")
	}

	cfg := config{
		feedItemsURL: feedItemsURL,
		apiKey:       strings.TrimSpace(os.Getenv("MKRCX_SIGNIN_API_KEY")),
		webhookURL:   webhookURL,
		pollInterval: pollInterval,
		httpTimeout:  httpTimeout,
	}
	if err := validateConfig(cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value < 250*time.Millisecond || value > time.Minute {
		return 0, fmt.Errorf("%s must be a duration between 250ms and 1m", name)
	}
	return value, nil
}

func validateConfig(cfg config) error {
	if cfg.feedItemsURL == nil || cfg.feedItemsURL.Scheme != "https" || cfg.feedItemsURL.Host == "" ||
		cfg.feedItemsURL.User != nil || cfg.feedItemsURL.RawQuery != "" || cfg.feedItemsURL.Fragment != "" ||
		!feedItemsPathPattern.MatchString(cfg.feedItemsURL.EscapedPath()) {
		return errors.New("MKRCX_SIGNIN_FEED_ITEMS_URL must be an HTTPS numeric /api/feeds/{id}/items URL without credentials or query parameters")
	}
	if cfg.apiKey == "" {
		return errors.New("MKRCX_SIGNIN_API_KEY is required")
	}
	if cfg.webhookURL == nil || cfg.webhookURL.Scheme != "https" || !strings.EqualFold(cfg.webhookURL.Hostname(), "discord.com") ||
		(cfg.webhookURL.Port() != "" && cfg.webhookURL.Port() != "443") ||
		cfg.webhookURL.User != nil || cfg.webhookURL.RawQuery != "" || cfg.webhookURL.Fragment != "" ||
		!webhookPathPattern.MatchString(cfg.webhookURL.EscapedPath()) {
		return errors.New("DISCORD_SIGNIN_WEBHOOK_URL must be an HTTPS discord.com /api/webhooks/{id}/{token} URL without query parameters")
	}
	if cfg.pollInterval <= 0 || cfg.httpTimeout <= 0 {
		return errors.New("poll interval and HTTP timeout must be positive")
	}
	return nil
}

type feedResponse struct {
	Data []feedItem `json:"data"`
}

type feedItem struct {
	ID                   uint      `json:"ID"`
	CreatedAt            time.Time `json:"CreatedAt"`
	LogLevel             uint      `json:"LogLevel"`
	UserID               uint      `json:"UserID"`
	UserDisplayName      string    `json:"UserDisplayName"`
	Title                string    `json:"Title"`
	Message              string    `json:"Message"`
	PendingUserSpecifier string    `json:"PendingUserSpecifier"`
}

type outcome string

const (
	outcomeComplete outcome = "complete"
	outcomeRequired outcome = "required"
	outcomeUnlinked outcome = "unlinked"
)

type checkinEvent struct {
	itemID      uint
	occurredAt  time.Time
	outcome     outcome
	displayName string
}

func canonicalEvent(item feedItem) (checkinEvent, bool) {
	if item.ID == 0 || item.CreatedAt.IsZero() || item.PendingUserSpecifier != "" {
		return checkinEvent{}, false
	}
	name := strings.TrimSpace(item.UserDisplayName)
	switch {
	case item.LogLevel == 0 && item.UserID != 0 && name != "" && item.Title == cardLinkedTitle && item.Message == docusignCompleteText:
		return checkinEvent{itemID: item.ID, occurredAt: item.CreatedAt, outcome: outcomeComplete, displayName: name}, true
	case item.LogLevel == 2 && item.UserID != 0 && name != "" && item.Title == cardLinkedTitle && item.Message == docusignRequiredText:
		return checkinEvent{itemID: item.ID, occurredAt: item.CreatedAt, outcome: outcomeRequired, displayName: name}, true
	case item.LogLevel == 4 && item.UserID == 0 && name == "" && item.Title == cardNotLinkedTitle && item.Message == cardNotLinkedText:
		return checkinEvent{itemID: item.ID, occurredAt: item.CreatedAt, outcome: outcomeUnlinked}, true
	default:
		return checkinEvent{}, false
	}
}

type allowedMentions struct {
	Parse []string `json:"parse"`
}

type discordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color"`
}

type discordPayload struct {
	Embeds          []discordEmbed  `json:"embeds"`
	AllowedMentions allowedMentions `json:"allowed_mentions"`
}

func renderDashboard(events []checkinEvent) discordPayload {
	lines := make([]string, 0, len(events)*2)
	for index := len(events) - 1; index >= 0; index-- {
		event := events[index]
		timestamp := fmt.Sprintf("<t:%d:T>", event.occurredAt.Unix())
		switch event.outcome {
		case outcomeComplete:
			lines = append(lines, fmt.Sprintf("🟢 **%s** · %s\n%s", escapeDiscordText(event.displayName), timestamp, docusignCompleteText))
		case outcomeRequired:
			lines = append(lines, fmt.Sprintf("🟡 **%s** · %s\n%s", escapeDiscordText(event.displayName), timestamp, docusignRequiredText))
		case outcomeUnlinked:
			lines = append(lines, fmt.Sprintf("🔴 **%s** · %s\n%s", cardNotLinkedTitle, timestamp, cardNotLinkedText))
		}
	}
	description := "Waiting for the next card tap."
	if len(lines) != 0 {
		description = strings.Join(lines, "\n\n")
	}
	return discordPayload{
		Embeds: []discordEmbed{{
			Title:       "Front Desk Check-in",
			Description: description,
			Color:       0x5865F2,
		}},
		AllowedMentions: allowedMentions{Parse: []string{}},
	}
}

func escapeDiscordText(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
	value = truncateRunes(value, 120)
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`*`, `\*`,
		`_`, `\_`,
		`~`, `\~`,
		"`", "\\`",
		`|`, `\|`,
		`<`, `\<`,
		`>`, `\>`,
	)
	return replacer.Replace(value)
}

func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit-1]) + "…"
}

type memoryState struct {
	messageID string
	cursor    uint
	events    []checkinEvent
}

type subscriber struct {
	cfg    config
	client *http.Client
	state  memoryState
	logger *log.Logger
}

func newSubscriber(cfg config, client *http.Client, logger *log.Logger) *subscriber {
	if client == nil {
		client = &http.Client{Timeout: cfg.httpTimeout}
	}
	copyClient := *client
	copyClient.Timeout = cfg.httpTimeout
	copyClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &subscriber{cfg: cfg, client: &copyClient, logger: logger}
}

func (s *subscriber) fetchFeedItems(ctx context.Context) ([]feedItem, uint, error) {
	endpoint := *s.cfg.feedItemsURL
	query := endpoint.Query()
	query.Set("limit", strconv.Itoa(feedPageLimit))
	if s.state.cursor != 0 {
		query.Set("after_id", strconv.FormatUint(uint64(s.state.cursor), 10))
	}
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, 0, errors.New("feed request could not be created")
	}
	request.Header.Set("Authorization", "API-Key "+s.cfg.apiKey)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "mkrcx-checkin-discord/1")

	response, err := s.client.Do(request)
	if err != nil {
		return nil, 0, errors.New("feed request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, 0, fmt.Errorf("feed returned HTTP %d", response.StatusCode)
	}

	var payload feedResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes))
	if err := decoder.Decode(&payload); err != nil {
		return nil, 0, errors.New("feed returned invalid JSON")
	}
	sort.Slice(payload.Data, func(i, j int) bool { return payload.Data[i].ID < payload.Data[j].ID })
	maxID := s.state.cursor
	for _, item := range payload.Data {
		if item.ID > maxID {
			maxID = item.ID
		}
	}
	return payload.Data, maxID, nil
}

func mergeCanonical(existing []checkinEvent, items []feedItem, cursor uint) ([]checkinEvent, int) {
	merged := append([]checkinEvent(nil), existing...)
	seen := make(map[uint]struct{}, len(merged))
	for _, event := range merged {
		seen[event.itemID] = struct{}{}
	}
	skipped := 0
	for _, item := range items {
		if item.ID <= cursor {
			continue
		}
		event, ok := canonicalEvent(item)
		if !ok {
			skipped++
			continue
		}
		if _, duplicate := seen[event.itemID]; duplicate {
			continue
		}
		seen[event.itemID] = struct{}{}
		merged = append(merged, event)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].itemID < merged[j].itemID })
	if len(merged) > rollingItemLimit {
		merged = append([]checkinEvent(nil), merged[len(merged)-rollingItemLimit:]...)
	}
	return merged, skipped
}

type retryAfterError struct {
	after time.Duration
}

func (e *retryAfterError) Error() string { return "Discord rate limited the update" }

func retryDelay(response *http.Response) time.Duration {
	raw := strings.TrimSpace(response.Header.Get("Retry-After"))
	if seconds, err := strconv.ParseFloat(raw, 64); err == nil && seconds >= 0 {
		return boundedRetry(time.Duration(seconds * float64(time.Second)))
	}
	if when, err := http.ParseTime(raw); err == nil {
		return boundedRetry(time.Until(when))
	}
	var body struct {
		RetryAfter float64 `json:"retry_after"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&body); err == nil && body.RetryAfter >= 0 {
		return boundedRetry(time.Duration(body.RetryAfter * float64(time.Second)))
	}
	return time.Second
}

func boundedRetry(delay time.Duration) time.Duration {
	if delay < 250*time.Millisecond {
		return 250 * time.Millisecond
	}
	if delay > time.Minute {
		return time.Minute
	}
	return delay
}

func (s *subscriber) sendDiscord(ctx context.Context, method string, endpoint *url.URL, payload discordPayload) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("Discord payload could not be encoded")
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.New("Discord request could not be created")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "mkrcx-checkin-discord/1")
	response, err := s.client.Do(request)
	if err != nil {
		return nil, errors.New("Discord request failed")
	}
	return response, nil
}

func (s *subscriber) createDiscordMessage(ctx context.Context, payload discordPayload) (string, error) {
	endpoint := *s.cfg.webhookURL
	query := endpoint.Query()
	query.Set("wait", "true")
	endpoint.RawQuery = query.Encode()

	response, err := s.sendDiscord(ctx, http.MethodPost, &endpoint, payload)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return "", &retryAfterError{after: retryDelay(response)}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("Discord create returned HTTP %d", response.StatusCode)
	}
	var message struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&message); err != nil || !messageIDPattern.MatchString(message.ID) {
		return "", errors.New("Discord create returned an invalid message identifier")
	}
	return message.ID, nil
}

func (s *subscriber) updateDiscordMessage(ctx context.Context, messageID string, payload discordPayload) (bool, error) {
	if !messageIDPattern.MatchString(messageID) {
		return false, errors.New("stored Discord message identifier is invalid")
	}
	endpoint := *s.cfg.webhookURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/messages/" + url.PathEscape(messageID)
	query := endpoint.Query()
	query.Set("wait", "true")
	endpoint.RawQuery = query.Encode()

	response, err := s.sendDiscord(ctx, http.MethodPatch, &endpoint, payload)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return false, &retryAfterError{after: retryDelay(response)}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return false, fmt.Errorf("Discord update returned HTTP %d", response.StatusCode)
	}
	return true, nil
}

func (s *subscriber) upsertDashboard(ctx context.Context, payload discordPayload) (string, error) {
	if s.state.messageID != "" {
		updated, err := s.updateDiscordMessage(ctx, s.state.messageID, payload)
		if err != nil {
			return "", err
		}
		if updated {
			return s.state.messageID, nil
		}
	}
	return s.createDiscordMessage(ctx, payload)
}

func (s *subscriber) pollOnce(ctx context.Context) (int, error) {
	items, nextCursor, err := s.fetchFeedItems(ctx)
	if err != nil {
		return 0, err
	}
	if s.state.messageID != "" && nextCursor == s.state.cursor {
		return 0, nil
	}
	nextEvents, skipped := mergeCanonical(s.state.events, items, s.state.cursor)
	messageID, err := s.upsertDashboard(ctx, renderDashboard(nextEvents))
	if err != nil {
		return skipped, err
	}
	// Feed progress becomes durable for this process only after Discord accepted
	// the corresponding rolling view. Failed updates are replayed on the next poll.
	s.state.messageID = messageID
	s.state.events = nextEvents
	s.state.cursor = nextCursor
	return skipped, nil
}

func (s *subscriber) run(ctx context.Context) error {
	delay := time.Duration(0)
	for {
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
		}

		skipped, err := s.pollOnce(ctx)
		if skipped != 0 {
			s.logger.Printf("ignored %d non-canonical signin feed item(s)", skipped)
		}
		if err == nil {
			delay = s.cfg.pollInterval
			continue
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) && ctx.Err() != nil {
			return nil
		}
		var retry *retryAfterError
		if errors.As(err, &retry) {
			delay = retry.after
			s.logger.Printf("Discord rate limited the dashboard update; retrying after %s", delay.Round(time.Millisecond))
			continue
		}
		s.logger.Printf("check-in dashboard cycle failed: %v", err)
		delay = s.cfg.pollInterval
	}
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	worker := newSubscriber(cfg, &http.Client{Timeout: cfg.httpTimeout}, log.Default())
	log.Printf("starting private check-in Discord subscriber")
	if err := worker.run(ctx); err != nil {
		log.Fatalf("subscriber stopped: %v", err)
	}
}
