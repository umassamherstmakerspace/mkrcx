package leash_backend_api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

func newCheckinTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:checkins-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Hold{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAuthorizeCardCheckinRequiresServiceAPIKeyPermission(t *testing.T) {
	db := newCheckinTestDB(t)
	enforcer, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		t.Fatal(err)
	}
	wrapper := leash_auth.EnforcerWrapper{Enforcer: enforcer}
	allowedKey := models.APIKey{Key: "allowed"}
	if err := wrapper.SetPermissionsForAPIKey(allowedKey, []string{checkinPermission}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		authentication leash_auth.Authentication
		wantStatus     int
	}{
		{
			name: "dedicated service key",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_APIKEY,
				User:          models.User{Role: "service"},
				Data:          allowedKey,
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusNoContent,
		},
		{
			name: "service key without permission",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_APIKEY,
				User:          models.User{Role: "service"},
				Data:          models.APIKey{Key: "not-allowed"},
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusForbidden,
		},
		{
			name: "full access key is rejected",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_APIKEY,
				User:          models.User{Role: "service"},
				Data:          models.APIKey{Key: "full-access", FullAccess: true},
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusForbidden,
		},
		{
			name: "signed-in service user",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_USER,
				User:          models.User{Role: "service"},
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusForbidden,
		},
		{
			name: "non-service API key",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_APIKEY,
				User:          models.User{Role: "staff"},
				Data:          allowedKey,
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/", func(c *fiber.Ctx) error {
				c.Locals("auth", test.authentication)
				return c.Next()
			}, authorizeCardCheckin, func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusNoContent)
			})
			response, err := app.Test(httptest.NewRequest(http.MethodPost, "/", nil))
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
		})
	}
}

func TestRecordCardCheckinPersistsOnceWithoutRawCard(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:record-checkin-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Hold{}, &models.Feed{}, &models.FeedMessage{}); err != nil {
		t.Fatal(err)
	}
	feed := models.Feed{Name: checkinFeedName}
	if err := db.Create(&feed).Error; err != nil {
		t.Fatal(err)
	}
	card := "aabbccddeeff0011"
	user := models.User{CardID: &card, Name: "Casey Jordan", Email: "casey@example.test", Role: "member", Type: "other"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	when := time.Now().UTC().Add(-time.Minute).Truncate(time.Millisecond)
	authentication := leash_auth.Authentication{
		Authenticator: leash_auth.AUTHENTICATOR_APIKEY,
		User:          models.User{ID: 99, Role: "service"},
		Data:          models.APIKey{Key: "reader-key"},
	}

	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		c.Locals("db", db)
		c.Locals("auth", authentication)
		c.Locals("body", cardCheckinRequest{Card: card, OccurredAt: &when})
		return c.Next()
	}, recordCardCheckin(NewLocalFeedRuntime()))

	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/", nil)
		request.Header.Set("Idempotency-Key", "card-server-swipe-123")
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		wantStatus := fiber.StatusCreated
		if attempt > 0 {
			wantStatus = fiber.StatusOK
		}
		if response.StatusCode != wantStatus {
			t.Fatalf("attempt %d status = %d, want %d", attempt+1, response.StatusCode, wantStatus)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		var decision struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(body, &decision); err != nil || decision.Status != checkinDecisionGreen {
			t.Fatalf("attempt %d returned an invalid decision: %q (%v)", attempt+1, body, err)
		}
		for _, private := range []string{card, user.Name, user.Email} {
			if strings.Contains(string(body), private) {
				t.Fatalf("attempt %d leaked private data: %q", attempt+1, body)
			}
		}
		if attempt == 0 {
			if err := db.Model(&user).Update("card_id", nil).Error; err != nil {
				t.Fatal(err)
			}
			newOwner := models.User{CardID: &card, Name: "New Owner", Email: "new-owner@example.test", Role: "member", Type: "other"}
			if err := db.Create(&newOwner).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	var count int64
	if err := db.Model(&models.FeedMessage{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("idempotent retry created %d rows, want 1", count)
	}
	var persisted models.FeedMessage
	if err := db.First(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.UserID != user.ID || persisted.LogLevel != 0 || !persisted.CreatedAt.Equal(when) {
		t.Fatalf("unexpected persisted item: %+v", persisted)
	}
}

func TestRecordUnknownCardStoresOnlyKeyedFingerprint(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:unknown-checkin-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Hold{}, &models.Feed{}, &models.FeedMessage{}); err != nil {
		t.Fatal(err)
	}
	feed := models.Feed{Name: checkinFeedName}
	if err := db.Create(&feed).Error; err != nil {
		t.Fatal(err)
	}
	card := "0badcafe12345678"
	when := time.Now().UTC().Add(-time.Minute).Truncate(time.Millisecond)
	authentication := leash_auth.Authentication{
		Authenticator: leash_auth.AUTHENTICATOR_APIKEY,
		User: models.User{ID: 99, Role: "service"},
		Data: models.APIKey{Key: "reader-key"},
	}
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		c.Locals("db", db)
		c.Locals("auth", authentication)
		c.Locals("hmac_secret", []byte("fingerprint-secret"))
		c.Locals("body", cardCheckinRequest{Card: card, OccurredAt: &when})
		return c.Next()
	}, recordCardCheckin(NewLocalFeedRuntime()))
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("Idempotency-Key", "unknown-event")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusCreated)
	}
	var decision struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decision); err != nil || decision.Status != checkinDecisionRed {
		t.Fatalf("unexpected decision: %+v (%v)", decision, err)
	}
	var stored models.FeedMessage
	if err := db.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.PendingCardFingerprint == nil || len(*stored.PendingCardFingerprint) != 64 || strings.Contains(*stored.PendingCardFingerprint, card) {
		t.Fatalf("unsafe or missing card fingerprint: %+v", stored.PendingCardFingerprint)
	}
	encoded, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), card) || strings.Contains(string(encoded), *stored.PendingCardFingerprint) {
		t.Fatalf("private card correlation leaked in JSON: %s", encoded)
	}
}

func TestValidateCardCheckinTimeBoundsAndNormalizes(t *testing.T) {
	now := time.Date(2026, time.August, 4, 16, 30, 0, 0, time.UTC)
	accepted, err := validateCardCheckinTime(now.Add(-time.Hour+123456*time.Nanosecond), now)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Nanosecond()%int(time.Millisecond) != 0 || accepted.Location() != time.UTC {
		t.Fatalf("time was not UTC millisecond precision: %s", accepted)
	}
	for _, outside := range []time.Time{
		now.Add(-maxCheckinAge - time.Millisecond),
		now.Add(maxCheckinFutureSkew + time.Millisecond),
	} {
		if _, err := validateCardCheckinTime(outside, now); err == nil {
			t.Fatalf("out-of-window time was accepted: %s", outside)
		}
	}
}

func TestResolveCardCheckinIgnoresDeletedUserAndMatchesUppercaseStoredCard(t *testing.T) {
	db := newCheckinTestDB(t)
	when := time.Now().UTC()
	uppercaseCard := "ABCDEF0123456789"
	active := models.User{CardID: &uppercaseCard, Name: "Active Person", Email: "active@example.test", Role: "member", Type: "other"}
	if err := db.Create(&active).Error; err != nil {
		t.Fatal(err)
	}
	item, err := resolveCardCheckin(db, strings.ToLower(uppercaseCard), when)
	if err != nil {
		t.Fatal(err)
	}
	if item.UserID == nil || *item.UserID != active.ID {
		t.Fatalf("uppercase stored card was not resolved: %+v", item)
	}

	deletedCard := "0011AABB2233CCDD"
	deleted := models.User{CardID: &deletedCard, Name: "Deleted Person", Email: "deleted@example.test", Role: "member", Type: "other"}
	if err := db.Create(&deleted).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatal(err)
	}
	item, err = resolveCardCheckin(db, strings.ToLower(deletedCard), when)
	if err != nil {
		t.Fatal(err)
	}
	if item.UserID != nil || item.LogLevel != 4 {
		t.Fatalf("deleted user did not resolve as anonymous unknown card: %+v", item)
	}
}

func TestResolveCardCheckinStates(t *testing.T) {
	db := newCheckinTestDB(t)
	when := time.Date(2026, time.August, 4, 15, 21, 9, 0, time.UTC)

	unknown, err := resolveCardCheckin(db, "0011223344556677", when)
	if err != nil {
		t.Fatal(err)
	}
	if unknown.LogLevel != 4 || unknown.Title != cardNotLinkedTitle || unknown.Message != cardNotLinkedText || unknown.UserID != nil {
		t.Fatalf("unexpected unknown-card result: %+v", unknown)
	}

	card := "8899aabbccddeeff"
	user := models.User{CardID: &card, Name: "Alex Rivera", Email: "alex@example.test", Role: "member", Type: "other"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	orientation := models.Hold{Model: models.Model{CreatedAt: when.Add(-time.Hour)}, UserID: user.ID, Name: "orientation", Reason: "retired", Priority: 2}
	if err := db.Create(&orientation).Error; err != nil {
		t.Fatal(err)
	}

	complete, err := resolveCardCheckin(db, strings.ToUpper(card), when)
	if err != nil {
		t.Fatal(err)
	}
	if complete.LogLevel != 0 || complete.Title != cardLinkedTitle || complete.Message != docusignCompleteText || complete.UserID == nil || *complete.UserID != user.ID || complete.CheckinDecision != checkinDecisionYellow {
		t.Fatalf("unexpected completed result: %+v", complete)
	}

	active := models.Hold{Model: models.Model{CreatedAt: when.Add(-time.Hour)}, UserID: user.ID, Name: docusignHoldName, Reason: "agreement required", Priority: 1}
	if err := db.Create(&active).Error; err != nil {
		t.Fatal(err)
	}
	required, err := resolveCardCheckin(db, card, when)
	if err != nil {
		t.Fatal(err)
	}
	if required.LogLevel != 2 || required.Title != cardLinkedTitle || required.Message != docusignRequiredText || required.UserID == nil || *required.UserID != user.ID || required.CheckinDecision != checkinDecisionYellow {
		t.Fatalf("unexpected required result: %+v", required)
	}
}

func TestResolveCardCheckinReturnsGreenWithNoActiveHolds(t *testing.T) {
	db := newCheckinTestDB(t)
	when := time.Now().UTC()
	card := "13579bdf2468ace0"
	user := models.User{CardID: &card, Name: "Green Member", Email: "green@example.test", Role: "member", Type: "other"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	item, err := resolveCardCheckin(db, card, when)
	if err != nil {
		t.Fatal(err)
	}
	if item.CheckinDecision != checkinDecisionGreen {
		t.Fatalf("decision = %q, want green", item.CheckinDecision)
	}
}

func TestResolveCardCheckinUsesOnlyActiveDocusignHold(t *testing.T) {
	db := newCheckinTestDB(t)
	when := time.Date(2026, time.August, 4, 15, 21, 9, 0, time.UTC)
	card := "0123456789abcdef"
	user := models.User{CardID: &card, Name: "Morgan Lee", Email: "morgan@example.test", Role: "member", Type: "other"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	past := when.Add(-time.Hour)
	future := when.Add(time.Hour)
	holds := []models.Hold{
		{Model: models.Model{CreatedAt: when.Add(-2 * time.Hour)}, UserID: user.ID, Name: docusignHoldName, End: &past, Priority: 1},
		{Model: models.Model{CreatedAt: when.Add(-2 * time.Hour)}, UserID: user.ID, Name: docusignHoldName, Start: &future, Priority: 1},
		{Model: models.Model{CreatedAt: when.Add(-2 * time.Hour)}, UserID: user.ID, Name: "other", Priority: 1},
	}
	for index := range holds {
		if err := db.Create(&holds[index]).Error; err != nil {
			t.Fatal(err)
		}
	}
	deletedBeforeTap := models.Hold{Model: models.Model{CreatedAt: when.Add(-2 * time.Hour)}, UserID: user.ID, Name: docusignHoldName, Priority: 1}
	if err := db.Create(&deletedBeforeTap).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&deletedBeforeTap).Update("deleted_at", when.Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}

	item, err := resolveCardCheckin(db, card, when)
	if err != nil {
		t.Fatal(err)
	}
	if item.LogLevel != 0 || item.Message != docusignCompleteText {
		t.Fatalf("inactive or unrelated hold changed the result: %+v", item)
	}
	if item.CheckinDecision != checkinDecisionYellow {
		t.Fatalf("active non-DocuSign hold returned decision %q, want yellow", item.CheckinDecision)
	}
}

func TestResolveCardCheckinCountsHoldRemovedAfterTap(t *testing.T) {
	db := newCheckinTestDB(t)
	when := time.Date(2026, time.August, 4, 15, 21, 9, 0, time.UTC)
	card := "1122334455667788"
	user := models.User{CardID: &card, Name: "Taylor Kim", Email: "taylor@example.test", Role: "member", Type: "other"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	hold := models.Hold{Model: models.Model{CreatedAt: when.Add(-time.Hour)}, UserID: user.ID, Name: docusignHoldName, Priority: 1}
	if err := db.Create(&hold).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&hold).Update("deleted_at", when.Add(time.Minute)).Error; err != nil {
		t.Fatal(err)
	}

	item, err := resolveCardCheckin(db, card, when)
	if err != nil {
		t.Fatal(err)
	}
	if item.LogLevel != 2 || item.Message != docusignRequiredText {
		t.Fatalf("hold active at tap was not counted: %+v", item)
	}
}

func TestResolveCardCheckinRejectsMalformedCardWithoutLeakingIt(t *testing.T) {
	db := newCheckinTestDB(t)
	card := "not-a-card-secret"
	_, err := resolveCardCheckin(db, card, time.Now())
	if err == nil {
		t.Fatal("malformed card was accepted")
	}
	if strings.Contains(err.Error(), card) {
		t.Fatalf("raw card leaked in error: %v", err)
	}
}

func TestResolvedCheckinFeedItemDoesNotSerializeIdentityText(t *testing.T) {
	db := newCheckinTestDB(t)
	card := "fedcba9876543210"
	user := models.User{CardID: &card, Name: "Private Person", Email: "private@example.test", Role: "member", Type: "other"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	item, err := resolveCardCheckin(db, card, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{card, user.Name, user.Email} {
		if strings.Contains(string(payload), private) {
			t.Fatalf("private check-in data leaked in item JSON: %s", payload)
		}
	}
}

func TestUnknownCheckinResolvesAfterCardLinkWithoutChangingRedHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:resolve-unknown-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Hold{}, &models.Feed{}, &models.FeedMessage{}); err != nil {
		t.Fatal(err)
	}
	feed := models.Feed{Name: checkinFeedName}
	if err := db.Create(&feed).Error; err != nil {
		t.Fatal(err)
	}

	card := "1234567890abcdef"
	fingerprint, err := fingerprintCheckinCardWithSecret([]byte("test-secret"), card)
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().UTC().Add(-time.Hour)
	item := models.FeedMessage{
		FeedID: feed.ID, LogLevel: 4, Title: cardNotLinkedTitle, Message: cardNotLinkedText,
		Model: models.Model{CreatedAt: when}, PendingCardFingerprint: &fingerprint,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Name: "Later Linked", Email: "later@example.test", Role: "member", Type: "other", CardID: &card}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	hold := models.Hold{Model: models.Model{CreatedAt: when.Add(-time.Minute)}, UserID: user.ID, Name: docusignHoldName}
	if err := db.Create(&hold).Error; err != nil {
		t.Fatal(err)
	}

	count, err := resolvePendingCardCheckins(db, NewLocalFeedRuntime(), user.ID, fingerprint, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("resolved %d rows, want 1", count)
	}
	var resolved models.FeedMessage
	if err := db.First(&resolved, item.ID).Error; err != nil {
		t.Fatal(err)
	}
	if resolved.UserID != user.ID || resolved.LogLevel != 4 || resolved.Title != cardNotLinkedTitle || resolved.Message != cardResolvedRequiredText || resolved.PendingCardFingerprint != nil {
		t.Fatalf("unexpected resolved history: %+v", resolved)
	}
}

func TestCheckinFeedRetentionHardDeletesOnlyExpiredSigninItems(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:retention-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Feed{}, &models.FeedMessage{}); err != nil {
		t.Fatal(err)
	}
	feeds := []models.Feed{{Name: checkinFeedName}, {Name: "other-feed"}}
	if err := db.Create(&feeds).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	items := []models.FeedMessage{
		{Model: models.Model{CreatedAt: now.Add(-checkinFeedRetention - time.Minute)}, FeedID: feeds[0].ID, Title: "old", Message: "old"},
		{Model: models.Model{CreatedAt: now.Add(-time.Hour)}, FeedID: feeds[0].ID, Title: "recent", Message: "recent"},
		{Model: models.Model{CreatedAt: now.Add(-checkinFeedRetention - time.Minute)}, FeedID: feeds[1].ID, Title: "other", Message: "other"},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	visible, err := queryFeedItems(db, feeds[0], feedItemListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0].ID != items[1].ID {
		t.Fatalf("read-time retention returned %+v, want only recent signin item %d", visible, items[1].ID)
	}
	deleted, err := purgeExpiredCheckinFeedItems(db, now)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted %d rows, want 1", deleted)
	}
	var count int64
	if err := db.Unscoped().Model(&models.FeedMessage{}).Where("id = ?", items[0].ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("expired signin item was not hard-deleted")
	}
	if err := db.Unscoped().Model(&models.FeedMessage{}).Where("id IN ?", []uint{items[1].ID, items[2].ID}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("retention removed non-expired or non-signin rows; remaining=%d", count)
	}
}
