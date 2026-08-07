package leash_backend_api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

func TestFeedNSQChannelIsUniqueAndEphemeral(t *testing.T) {
	first, err := feedNSQChannel("pod-a/unsafe")
	if err != nil {
		t.Fatal(err)
	}
	second, err := feedNSQChannel("pod-b")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("pod channels must differ: %q", first)
	}
	if !strings.HasSuffix(first, "#ephemeral") {
		t.Fatalf("channel must be ephemeral: %q", first)
	}
	if len(first) > 64 {
		t.Fatalf("channel exceeds NSQ's name limit: %d", len(first))
	}
}

func TestFeedNSQChannelKeepsUniqueSuffixForLongPodNames(t *testing.T) {
	prefix := strings.Repeat("deployment-name-", 4)
	first, err := feedNSQChannel(prefix + "pod-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := feedNSQChannel(prefix + "pod-b")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("long pod identities collided after shortening: %q", first)
	}
	if len(first) > 64 || len(second) > 64 {
		t.Fatalf("shortened channel exceeds NSQ limit: %d, %d", len(first), len(second))
	}
}

func TestPingNSQLookupdRequiresHealthyHTTP(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte("OK"))
	}))
	defer healthy.Close()
	if err := pingNSQLookupd(strings.TrimPrefix(healthy.URL, "http://")); err != nil {
		t.Fatalf("healthy lookupd was rejected: %v", err)
	}

	unhealthy := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, "not ready", http.StatusServiceUnavailable)
	}))
	defer unhealthy.Close()
	if err := pingNSQLookupd(strings.TrimPrefix(unhealthy.URL, "http://")); err == nil {
		t.Fatal("unhealthy lookupd was accepted")
	}
}

func TestFeedSocketUpgradeRoutePrecedesHTTPAuthentication(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api", leash_auth.SetPermissionPrefixMiddleware("leash"))
	RegisterAPIEndpoints(api)

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/feeds/1/ws", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusUpgradeRequired {
		t.Fatalf("unauthenticated non-upgrade request returned %d, want %d; HTTP auth likely intercepted the socket route", response.StatusCode, fiber.StatusUpgradeRequired)
	}
}

func TestFeedHubScopesEventsByFeed(t *testing.T) {
	hub := newFeedHub()
	first := &feedSocket{id: uuid.New(), feedID: 1, outbound: make(chan []byte, 1), done: make(chan struct{})}
	second := &feedSocket{id: uuid.New(), feedID: 2, outbound: make(chan []byte, 1), done: make(chan struct{})}
	hub.add(first)
	hub.add(second)

	item := models.FeedMessage{ID: 42, FeedID: 1, Title: "Tap"}
	if err := hub.broadcast(item); err != nil {
		t.Fatal(err)
	}
	select {
	case payload := <-first.outbound:
		var envelope struct {
			Type   string             `json:"type"`
			FeedID uint               `json:"feed_id"`
			Item   models.FeedMessage `json:"item"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Type != "feed_item.created" || envelope.FeedID != 1 || envelope.Item.ID != 42 {
			t.Fatalf("unexpected event: %+v", envelope)
		}
	default:
		t.Fatal("matching feed did not receive the event")
	}
	select {
	case <-second.outbound:
		t.Fatal("event crossed feed boundary")
	default:
	}
}

func TestFeedArchiveRetainsItemsAndIdempotencyIsPerFeedAndProducer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Feed{}, &models.FeedMessage{}); err != nil {
		t.Fatal(err)
	}

	firstFeed := models.Feed{Name: "signin"}
	secondFeed := models.Feed{Name: "machine-access"}
	if err := db.Create(&firstFeed).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&secondFeed).Error; err != nil {
		t.Fatal(err)
	}
	key := "reader-attempt-1"
	for _, feedID := range []uint{firstFeed.ID, secondFeed.ID} {
		item := models.FeedMessage{FeedID: feedID, Title: "Tap", Message: "Allowed", IdempotencyScope: "reader-a", IdempotencyKey: &key}
		if err := db.Create(&item).Error; err != nil {
			t.Fatalf("key should be reusable across feeds: %v", err)
		}
	}
	secondReader := models.FeedMessage{FeedID: firstFeed.ID, Title: "Tap", Message: "Allowed", IdempotencyScope: "reader-b", IdempotencyKey: &key}
	if err := db.Create(&secondReader).Error; err != nil {
		t.Fatalf("key should be reusable by another producer: %v", err)
	}
	duplicate := models.FeedMessage{FeedID: firstFeed.ID, Title: "Tap", Message: "Allowed", IdempotencyScope: "reader-a", IdempotencyKey: &key}
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate feed/producer idempotency key was accepted")
	}

	if err := db.Delete(&firstFeed).Error; err != nil {
		t.Fatal(err)
	}
	var itemCount int64
	if err := db.Model(&models.FeedMessage{}).Where("feed_id = ?", firstFeed.ID).Count(&itemCount).Error; err != nil {
		t.Fatal(err)
	}
	if itemCount != 2 {
		t.Fatalf("archiving feed removed history; got %d items", itemCount)
	}
}

func TestFeedItemSerializationOmitsSensitiveProducerData(t *testing.T) {
	item := models.FeedMessage{
		ID: 1, FeedID: 2, Title: "Unknown card", Message: "Ask the visitor to see a supervisor",
		PendingUserSpecifier: "opaque", PendingUserData: "sensitive-value-must-not-leak",
		IdempotencyScope: "sensitive-producer-scope",
	}
	payload, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), item.PendingUserData) {
		t.Fatalf("sensitive producer data leaked in JSON: %s", payload)
	}
	if strings.Contains(string(payload), item.IdempotencyScope) {
		t.Fatalf("producer idempotency scope leaked in JSON: %s", payload)
	}
}

func TestFeedDisplayNameHydrationIsLimitedToSignin(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:feed-name-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.FeedMessage{}); err != nil {
		t.Fatal(err)
	}
	user := models.User{Name: "Private Display Name", Email: "display@example.test", Role: "member", Type: "other"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	for _, feedID := range []uint{1, 2} {
		if err := db.Create(&models.FeedMessage{FeedID: feedID, UserID: user.ID, Title: "Fixed title", Message: "Fixed message"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	limit := 10
	signinItems, err := queryFeedItems(db, models.Feed{ID: 1, Name: checkinFeedName}, feedItemListRequest{Limit: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if len(signinItems) != 1 || signinItems[0].UserDisplayName != user.Name || signinItems[0].UserEmail != user.Email {
		t.Fatalf("signin identity was not hydrated: %+v", signinItems)
	}
	otherItems, err := queryFeedItems(db, models.Feed{ID: 2, Name: "machine-access"}, feedItemListRequest{Limit: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if len(otherItems) != 1 || otherItems[0].UserDisplayName != "" || otherItems[0].UserEmail != "" {
		t.Fatalf("non-signin feed leaked user identity: %+v", otherItems)
	}
}

func TestSameFeedItemContentDetectsIdempotencyConflict(t *testing.T) {
	first := models.FeedMessage{FeedID: 1, Title: "Tap", Message: "Allowed", UserID: 10}
	identical := first
	changed := first
	changed.Message = "Denied"
	otherProducer := first
	otherProducer.AddedBy = 11
	otherTime := first
	otherTime.CreatedAt = time.Now()
	if !sameFeedItemContent(first, identical) {
		t.Fatal("identical retry was treated as a conflict")
	}
	if sameFeedItemContent(first, changed) {
		t.Fatal("changed retry content was accepted")
	}
	if sameFeedItemContent(first, otherProducer) {
		t.Fatal("different producer was treated as an identical retry")
	}
	if sameFeedItemContent(first, otherTime) {
		t.Fatal("different occurrence time was treated as an identical retry")
	}
}

func TestFeedAuthorizationRefreshesAcrossEnforcers(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "policy.db")
	firstDB, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	firstSQLDB, err := firstDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = firstSQLDB.Close() })
	secondDB, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	secondSQLDB, err := secondDB.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = secondSQLDB.Close() })
	firstEnforcer, err := leash_auth.InitializeCasbin(firstDB)
	if err != nil {
		t.Fatal(err)
	}
	secondEnforcer, err := leash_auth.InitializeCasbin(secondDB)
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{ID: 42, Role: "service"}
	permission := "leash.feeds.signin:read"
	first := leash_auth.EnforcerWrapper{Enforcer: firstEnforcer}
	if err := first.SetPermissionsForUser(user, []string{permission}); err != nil {
		t.Fatal(err)
	}
	if err := secondEnforcer.LoadPolicy(); err != nil {
		t.Fatal(err)
	}
	authentication := leash_auth.Authentication{
		Authenticator: leash_auth.AUTHENTICATOR_USER,
		User:          user,
		Enforcer:      leash_auth.EnforcerWrapper{Enforcer: secondEnforcer},
	}
	feed := models.Feed{Name: "signin"}
	if err := authorizeFeed(authentication, feed, "read"); err != nil {
		t.Fatalf("second pod did not observe grant: %v", err)
	}
	if err := first.SetPermissionsForUser(user, nil); err != nil {
		t.Fatal(err)
	}
	if err := secondEnforcer.LoadPolicy(); err != nil {
		t.Fatal(err)
	}
	if err := authorizeFeed(authentication, feed, "read"); err == nil {
		t.Fatal("second pod retained revoked feed permission after policy refresh")
	}
}

func TestFeedCursorRejectsConflictingDirections(t *testing.T) {
	after, before := uint(1), uint(2)
	_, err := queryFeedItems(nil, models.Feed{ID: 1}, feedItemListRequest{AfterID: &after, BeforeID: &before})
	var fiberError *fiber.Error
	if !errors.As(err, &fiberError) || fiberError.Code != fiber.StatusBadRequest {
		t.Fatalf("conflicting cursors returned %v, want Fiber 400", err)
	}
}

func TestFeedAfterCursorPagesOldestFirstWithoutGaps(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.FeedMessage{}); err != nil {
		t.Fatal(err)
	}

	for id := uint(1); id <= 205; id++ {
		item := models.FeedMessage{ID: id, FeedID: 7, Title: "Tap", Message: "Allowed"}
		if err := db.Create(&item).Error; err != nil {
			t.Fatal(err)
		}
	}

	cursor := uint(5)
	seen := make([]uint, 0, 200)
	for {
		limit := 100
		page, err := queryFeedItems(db, models.Feed{ID: 7}, feedItemListRequest{AfterID: &cursor, Limit: &limit})
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range page {
			seen = append(seen, item.ID)
			cursor = item.ID
		}
		if len(page) < limit {
			break
		}
	}

	if len(seen) != 200 {
		t.Fatalf("cursor recovery got %d items, want 200", len(seen))
	}
	for index, id := range seen {
		want := uint(index + 6)
		if id != want {
			t.Fatalf("cursor recovery item %d = %d, want %d", index, id, want)
		}
	}
}
