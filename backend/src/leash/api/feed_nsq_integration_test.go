//go:build nsq_integration

package leash_backend_api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const nsqIntegrationDeadline = 10 * time.Second

type nsqStatsResponse struct {
	Topics []struct {
		TopicName string `json:"topic_name"`
		Channels  []struct {
			ChannelName string `json:"channel_name"`
			ClientCount int    `json:"client_count"`
		} `json:"channels"`
	} `json:"topics"`
}

type feedCreatedEnvelope struct {
	Type   string             `json:"type"`
	FeedID uint               `json:"feed_id"`
	Item   models.FeedMessage `json:"item"`
}

func requiredNSQTestEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s must be set by scripts/test-feed-nsq.ps1", name)
	}
	return value
}

func openNSQTestDB(t *testing.T, databasePath string) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(databasePath) + "?cache=shared&_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func addNSQTestSocket(runtime *FeedRuntime, feedID uint) *feedSocket {
	socket := &feedSocket{
		id:       uuid.New(),
		feedID:   feedID,
		outbound: make(chan []byte, feedSocketQueueSize),
		done:     make(chan struct{}),
	}
	runtime.hub.add(socket)
	return socket
}

func waitForNSQChannels(t *testing.T, statsURL string, channels []string) {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(nsqIntegrationDeadline)
	for {
		response, err := client.Get(statsURL + "/stats?format=json")
		if err == nil {
			var stats nsqStatsResponse
			decodeErr := json.NewDecoder(response.Body).Decode(&stats)
			_ = response.Body.Close()
			if decodeErr == nil {
				ready := make(map[string]bool, len(channels))
				for _, topic := range stats.Topics {
					if topic.TopicName != feedNSQTopic {
						continue
					}
					for _, channel := range topic.Channels {
						if channel.ClientCount > 0 {
							ready[channel.ChannelName] = true
						}
					}
				}
				allReady := true
				for _, channel := range channels {
					allReady = allReady && ready[channel]
				}
				if allReady {
					return
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("NSQ channels did not become ready: %v", channels)
		}
		timer := time.NewTimer(50 * time.Millisecond)
		<-timer.C
	}
}

func receiveFeedCreated(t *testing.T, socket *feedSocket, deadline time.Time) feedCreatedEnvelope {
	t.Helper()
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	select {
	case payload := <-socket.outbound:
		var envelope feedCreatedEnvelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatal(err)
		}
		return envelope
	case <-timer.C:
		t.Fatal("timed out waiting for feed item")
		return feedCreatedEnvelope{}
	}
}

func TestNSQFeedRuntimeFansOutAcrossInstances(t *testing.T) {
	nsqdAddress := requiredNSQTestEnv(t, "NSQ_TEST_NSQD")
	lookupAddress := requiredNSQTestEnv(t, "NSQ_TEST_LOOKUPD")
	nsqdHTTP := requiredNSQTestEnv(t, "NSQ_TEST_NSQD_HTTP")

	databasePath := filepath.Join(t.TempDir(), "feeds.db")
	databases := []*gorm.DB{
		openNSQTestDB(t, databasePath),
		openNSQTestDB(t, databasePath),
		openNSQTestDB(t, databasePath),
	}
	if err := databases[0].AutoMigrate(&models.Feed{}, &models.FeedMessage{}); err != nil {
		t.Fatal(err)
	}
	feeds := []models.Feed{{Name: "signin"}, {Name: "machine-access"}}
	if err := databases[0].Create(&feeds).Error; err != nil {
		t.Fatal(err)
	}

	instanceIDs := []string{"integration-a", "integration-b", "integration-c"}
	runtimes := make([]*FeedRuntime, 0, len(instanceIDs))
	channels := make([]string, 0, len(instanceIDs))
	primarySockets := make([]*feedSocket, 0, len(instanceIDs))
	isolationSockets := make([]*feedSocket, 0, len(instanceIDs))
	for index, instanceID := range instanceIDs {
		runtime, err := NewNSQFeedRuntime(databases[index], nsqdAddress, lookupAddress, instanceID)
		if err != nil {
			t.Fatalf("start runtime %s: %v", instanceID, err)
		}
		t.Cleanup(runtime.Close)
		runtimes = append(runtimes, runtime)
		primarySockets = append(primarySockets, addNSQTestSocket(runtime, feeds[0].ID))
		isolationSockets = append(isolationSockets, addNSQTestSocket(runtime, feeds[1].ID))
		channel, err := feedNSQChannel(instanceID)
		if err != nil {
			t.Fatal(err)
		}
		channels = append(channels, channel)
	}
	waitForNSQChannels(t, nsqdHTTP, channels)

	const itemCount = 8
	items := make([]models.FeedMessage, 0, itemCount)
	for index := 0; index < itemCount; index++ {
		item := models.FeedMessage{
			FeedID:  feeds[0].ID,
			Title:   "Card tap",
			Message: fmt.Sprintf("integration item %d", index+1),
		}
		if err := databases[0].Create(&item).Error; err != nil {
			t.Fatal(err)
		}
		items = append(items, item)
	}

	started := time.Now()
	for index, item := range items {
		if err := runtimes[index%len(runtimes)].publish(item); err != nil {
			t.Fatalf("publish item %d: %v", item.ID, err)
		}
	}

	deadline := time.Now().Add(nsqIntegrationDeadline)
	for socketIndex, socket := range primarySockets {
		for itemIndex, expected := range items {
			event := receiveFeedCreated(t, socket, deadline)
			if event.Type != "feed_item.created" || event.FeedID != feeds[0].ID || event.Item.ID != expected.ID {
				t.Fatalf("runtime %d item %d: unexpected event %+v", socketIndex, itemIndex, event)
			}
		}
	}
	latency := time.Since(started)

	for index, socket := range isolationSockets {
		select {
		case payload := <-socket.outbound:
			t.Fatalf("runtime %d leaked signin event to machine-access feed: %s", index, payload)
		default:
		}
	}
	t.Logf("three NSQ instances each received %d ordered items in %s", itemCount, latency)
}
