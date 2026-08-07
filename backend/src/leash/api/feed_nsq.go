package leash_backend_api

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mkrcx/mkrcx/src/shared/models"
	"github.com/nsqio/go-nsq"
	"gorm.io/gorm"
)

const feedNSQTopic = "mkrcx-feed-items-v1"

var invalidNSQChannelCharacters = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type feedNSQEvent struct {
	Version uint `json:"version"`
	FeedID  uint `json:"feed_id"`
	ItemID  uint `json:"item_id"`
}

type feedNSQHandler struct {
	db      *gorm.DB
	runtime *FeedRuntime
}

func (h *feedNSQHandler) HandleMessage(message *nsq.Message) error {
	var event feedNSQEvent
	if err := json.Unmarshal(message.Body, &event); err != nil || event.Version != 1 || event.FeedID == 0 || event.ItemID == 0 {
		// Invalid events cannot become valid on retry, so acknowledge and log them.
		log.Printf("discarding invalid %s event: %q", feedNSQTopic, string(message.Body))
		return nil
	}

	var item models.FeedMessage
	result := h.db.Where("id = ? AND feed_id = ?", event.ItemID, event.FeedID).First(&item)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// The database commit should precede publish. A missing row is therefore poison.
		log.Printf("discarding %s event for missing feed item %d", feedNSQTopic, event.ItemID)
		return nil
	}
	if result.Error != nil {
		return result.Error
	}
	var feed models.Feed
	if result := h.db.Select("id", "name").Where("id = ?", event.FeedID).First(&feed); errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil
	} else if result.Error != nil {
		return result.Error
	}
	if feed.Name == checkinFeedName {
		if err := hydrateFeedUserDisplayName(h.db, &item); err != nil {
			return err
		}
	}
	return h.runtime.hub.broadcast(item)
}

func feedNSQChannel(instanceID string) (string, error) {
	originalID := strings.TrimSpace(instanceID)
	if originalID == "" {
		return "", errors.New("feed NSQ instance ID is empty")
	}
	instanceID = strings.Trim(invalidNSQChannelCharacters.ReplaceAllString(originalID, "-"), "-._")
	const prefix = "leash-feed-"
	const suffix = "#ephemeral"
	const nsqNameLimit = 64
	maxInstanceLength := nsqNameLimit - len(prefix) - len(suffix)
	if instanceID == "" || instanceID != originalID || len(instanceID) > maxInstanceLength {
		digest := fmt.Sprintf("%x", sha256.Sum256([]byte(originalID)))[:12]
		hashSuffix := "-" + digest
		if instanceID == "" {
			instanceID = "instance"
		}
		maxReadableLength := maxInstanceLength - len(hashSuffix)
		if len(instanceID) > maxReadableLength {
			instanceID = strings.TrimRight(instanceID[:maxReadableLength], "-._")
		}
		instanceID += hashSuffix
	}
	return prefix + instanceID + suffix, nil
}

func pingNSQLookupd(address string) error {
	endpoint := (&url.URL{Scheme: "http", Host: address, Path: "/ping"}).String()
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 16))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "OK" {
		return fmt.Errorf("unexpected lookupd health response: %s %q", response.Status, string(body))
	}
	return nil
}

// NewNSQFeedRuntime inserts NSQ between committed feed items and each pod's
// process-local WebSocket hub. instanceID must uniquely identify this pod.
func NewNSQFeedRuntime(db *gorm.DB, nsqdAddress, lookupAddress, instanceID string) (*FeedRuntime, error) {
	if db == nil {
		return nil, errors.New("feed NSQ database is nil")
	}
	if nsqdAddress == "" {
		return nil, errors.New("NSQD_HOST is not set")
	}
	if lookupAddress == "" {
		return nil, errors.New("NSQLOOKUP_HOST is not set")
	}
	channel, err := feedNSQChannel(instanceID)
	if err != nil {
		return nil, err
	}
	if err := pingNSQLookupd(lookupAddress); err != nil {
		return nil, fmt.Errorf("connect feed NSQ lookupd: %w", err)
	}

	producer, err := nsq.NewProducer(nsqdAddress, nsq.NewConfig())
	if err != nil {
		return nil, fmt.Errorf("create feed NSQ producer: %w", err)
	}
	if err := producer.Ping(); err != nil {
		producer.Stop()
		return nil, fmt.Errorf("connect feed NSQ producer: %w", err)
	}
	consumer, err := nsq.NewConsumer(feedNSQTopic, channel, nsq.NewConfig())
	if err != nil {
		producer.Stop()
		return nil, fmt.Errorf("create feed NSQ consumer: %w", err)
	}

	runtime := NewLocalFeedRuntime()
	runtime.publisher = func(item models.FeedMessage) error {
		event, err := json.Marshal(feedNSQEvent{Version: 1, FeedID: item.FeedID, ItemID: item.ID})
		if err != nil {
			return err
		}
		return producer.Publish(feedNSQTopic, event)
	}
	consumer.AddHandler(&feedNSQHandler{db: db, runtime: runtime})
	// Connect directly to the same nsqd used by the producer. This call performs
	// the TCP handshake and SUB synchronously, so startup cannot claim consumer
	// readiness while lookup discovery is still failing asynchronously.
	if err := consumer.ConnectToNSQD(nsqdAddress); err != nil {
		consumer.Stop()
		producer.Stop()
		return nil, fmt.Errorf("connect feed NSQ consumer to nsqd: %w", err)
	}
	if err := consumer.ConnectToNSQLookupd(lookupAddress); err != nil {
		consumer.Stop()
		producer.Stop()
		return nil, fmt.Errorf("connect feed NSQ consumer: %w", err)
	}
	runtime.close = func() {
		consumer.Stop()
		producer.Stop()
	}
	log.Printf("feed live delivery subscribed to NSQ topic %s on channel %s", feedNSQTopic, channel)
	return runtime, nil
}
