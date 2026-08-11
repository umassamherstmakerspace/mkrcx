package leash_backend_api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const (
	checkinFeedName            = "signin"
	docusignHoldName           = "docusign"
	checkinPermission          = "leash.checkins:record"
	cardLinkedTitle            = "Card linked"
	docusignCompleteText       = "DocuSign complete"
	docusignRequiredText       = "DocuSign required: ask them to complete the participation agreement."
	cardNotLinkedTitle         = "Card not linked"
	cardNotLinkedText          = "Check whether the user is registered; if so, direct them to link their UCard."
	cardResolvedCompleteText   = "Card was not linked at tap time. The participation agreement was complete then."
	cardResolvedRequiredText   = "Card was not linked at tap time. The participation agreement was also required then."
	checkinDecisionGreen       = "green"
	checkinDecisionYellow      = "yellow"
	checkinDecisionRed         = "red"
	maxCheckinAge              = time.Hour
	maxCheckinFutureSkew       = 5 * time.Minute
	checkinFeedRetention       = 7 * 24 * time.Hour
	checkinMaintenanceInterval = time.Hour
)

var cardIDPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

type cardCheckinRequest struct {
	Card       string     `json:"card" validate:"required,notblank,max=64"`
	OccurredAt *time.Time `json:"occurred_at" validate:"required"`
}

type checkinUser struct {
	ID uint
}

func normalizeCheckinCard(rawCard string) (string, error) {
	card := strings.ToLower(strings.TrimSpace(rawCard))
	if !cardIDPattern.MatchString(card) {
		return "", fiber.NewError(fiber.StatusBadRequest, "card must be 16 hexadecimal characters")
	}
	return card, nil
}

func fingerprintCheckinCard(hasher hash.Hash, card string) (string, error) {
	hasher.Reset()
	if _, err := hasher.Write([]byte(card)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func fingerprintCheckinCardWithSecret(secret []byte, card string) (string, error) {
	return fingerprintCheckinCard(hmac.New(sha256.New, secret), card)
}

func activeDocusignHoldCount(db *gorm.DB, userID uint, occurredAt time.Time) (int64, error) {
	var count int64
	err := db.Unscoped().Model(&models.Hold{}).
		Where("user_id = ? AND name = ?", userID, docusignHoldName).
		Where("created_at <= ?", occurredAt).
		Where("deleted_at IS NULL OR deleted_at > ?", occurredAt).
		Where("(start IS NULL OR start <= ?) AND (end IS NULL OR end > ?)", occurredAt, occurredAt).
		Count(&count).Error
	return count, err
}

func activeHoldCount(db *gorm.DB, userID uint, occurredAt time.Time) (int64, error) {
	var count int64
	err := db.Unscoped().Model(&models.Hold{}).
		Where("user_id = ?", userID).
		Where("created_at <= ?", occurredAt).
		Where("deleted_at IS NULL OR deleted_at > ?", occurredAt).
		Where("(start IS NULL OR start <= ?) AND (end IS NULL OR end > ?)", occurredAt, occurredAt).
		Count(&count).Error
	return count, err
}

func decisionFromFeedItem(item models.FeedMessage) string {
	if item.CheckinDecision != "" {
		return item.CheckinDecision
	}
	if item.UserID == 0 || item.LogLevel >= 4 {
		return checkinDecisionRed
	}
	if item.LogLevel > 0 {
		return checkinDecisionYellow
	}
	return checkinDecisionGreen
}

func sendCheckinDecision(c *fiber.Ctx, item models.FeedMessage, status int) error {
	decision := decisionFromFeedItem(item)
	if decision != checkinDecisionGreen && decision != checkinDecisionYellow && decision != checkinDecisionRed {
		return fiber.ErrInternalServerError
	}
	return c.Status(status).JSON(struct {
		Status string `json:"status"`
	}{Status: decision})
}

func resolveCardCheckin(db *gorm.DB, rawCard string, occurredAt time.Time) (feedItemRequest, error) {
	card, err := normalizeCheckinCard(rawCard)
	if err != nil {
		return feedItemRequest{}, err
	}

	var user checkinUser
	result := db.Table("users").Select("id").Where("deleted_at IS NULL").Where("card_id IN ?", []string{card, strings.ToUpper(card)}).Take(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return feedItemRequest{
			LogLevel: 4, Title: cardNotLinkedTitle, Message: cardNotLinkedText, OccurredAt: &occurredAt, CheckinDecision: checkinDecisionRed,
		}, nil
	}
	if result.Error != nil {
		return feedItemRequest{}, fiber.ErrInternalServerError
	}

	docusignHolds, err := activeDocusignHoldCount(db, user.ID, occurredAt)
	if err != nil {
		return feedItemRequest{}, fiber.ErrInternalServerError
	}
	activeHolds, err := activeHoldCount(db, user.ID, occurredAt)
	if err != nil {
		return feedItemRequest{}, fiber.ErrInternalServerError
	}

	level, message := uint(0), docusignCompleteText
	if docusignHolds > 0 {
		level, message = 2, docusignRequiredText
	}
	decision := checkinDecisionGreen
	if activeHolds > 0 {
		decision = checkinDecisionYellow
	}
	return feedItemRequest{
		LogLevel: level, Title: cardLinkedTitle, Message: message, UserID: &user.ID, OccurredAt: &occurredAt, CheckinDecision: decision,
	}, nil
}

func validateCardCheckinTime(occurredAt, now time.Time) (time.Time, error) {
	occurredAt = occurredAt.UTC().Truncate(time.Millisecond)
	now = now.UTC()
	if occurredAt.After(now.Add(maxCheckinFutureSkew)) || occurredAt.Before(now.Add(-maxCheckinAge)) {
		return time.Time{}, fiber.NewError(fiber.StatusBadRequest, "occurred_at is outside the accepted delivery window")
	}
	return occurredAt, nil
}

func recordCardCheckin(runtime *FeedRuntime) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Idempotency-Key is required")
		}

		var feed models.Feed
		db := leash_auth.GetDB(c)
		if result := db.Where("name = ?", checkinFeedName).First(&feed); errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusServiceUnavailable, "check-in feed is not configured")
		} else if result.Error != nil {
			return fiber.ErrInternalServerError
		}
		authentication := leash_auth.GetAuthentication(c)
		idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
		var existing models.FeedMessage
		result := db.Where(
			"feed_id = ? AND idempotency_scope = ? AND idempotency_key = ?",
			feed.ID, feedIdempotencyScope(authentication), idempotencyKey,
		).First(&existing)
		if result.Error == nil {
			return sendCheckinDecision(c, existing, fiber.StatusOK)
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fiber.ErrInternalServerError
		}

		req := c.Locals("body").(cardCheckinRequest)
		occurredAt, err := validateCardCheckinTime(req.OccurredAt.UTC(), time.Now())
		if err != nil {
			return err
		}
		item, err := resolveCardCheckin(db, req.Card, occurredAt)
		if err != nil {
			return err
		}
		if item.UserID == nil {
			card, err := normalizeCheckinCard(req.Card)
			if err != nil {
				return err
			}
			fingerprint, err := fingerprintCheckinCard(leash_auth.GetHMACSHA256(c), card)
			if err != nil {
				return fiber.ErrInternalServerError
			}
			item.PendingCardFingerprint = &fingerprint
		}
		return persistFeedItemWithHook(c, runtime, feed, item, feedItemResponseCheckinDecision, persistCheckinEvent)
	}
}

func resolvePendingCardCheckins(db *gorm.DB, runtime *FeedRuntime, userID uint, fingerprint string, now time.Time) (int64, error) {
	var feed models.Feed
	if result := db.Where("name = ?", checkinFeedName).First(&feed); errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return 0, nil
	} else if result.Error != nil {
		return 0, result.Error
	}
	var pending []models.FeedMessage
	if err := db.Where(
		"feed_id = ? AND user_id = 0 AND pending_card_fingerprint = ? AND created_at >= ?",
		feed.ID, fingerprint, now.UTC().Add(-checkinFeedRetention),
	).Order("id ASC").Find(&pending).Error; err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}

	updated := make([]models.FeedMessage, 0, len(pending))
	err := db.Transaction(func(tx *gorm.DB) error {
		for index := range pending {
			item := &pending[index]
			if err := resolveCheckinEventIdentity(tx, *item, userID); err != nil {
				return err
			}
			holds, err := activeDocusignHoldCount(tx, userID, item.CreatedAt)
			if err != nil {
				return err
			}
			item.UserID = userID
			item.PendingCardFingerprint = nil
			item.Message = cardResolvedCompleteText
			if holds > 0 {
				item.Message = cardResolvedRequiredText
			}
			result := tx.Model(&models.FeedMessage{}).
				Where("id = ? AND user_id = 0 AND pending_card_fingerprint = ?", item.ID, fingerprint).
				Updates(map[string]interface{}{
					"user_id":                  userID,
					"pending_card_fingerprint": nil,
					"message":                  item.Message,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				continue
			}
			updated = append(updated, *item)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	for index := range updated {
		if err := hydrateFeedUserDisplayName(db, &updated[index]); err != nil {
			return int64(index), err
		}
		if err := runtime.publish(updated[index]); err != nil {
			log.Printf("resolved feed item %d committed but live delivery failed: %v", updated[index].ID, err)
		}
	}
	return int64(len(updated)), nil
}

func resolvePendingCardCheckinsForUser(c *fiber.Ctx, runtime *FeedRuntime, userID uint, rawCard string) error {
	card, err := normalizeCheckinCard(rawCard)
	if err != nil {
		return err
	}
	fingerprint, err := fingerprintCheckinCard(leash_auth.GetHMACSHA256(c), card)
	if err != nil {
		return err
	}
	_, err = resolvePendingCardCheckins(leash_auth.GetDB(c), runtime, userID, fingerprint, time.Now().UTC())
	return err
}

func purgeExpiredCheckinFeedItems(db *gorm.DB, now time.Time) (int64, error) {
	var feed models.Feed
	if result := db.Where("name = ?", checkinFeedName).First(&feed); errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return 0, nil
	} else if result.Error != nil {
		return 0, result.Error
	}
	result := db.Unscoped().Where("feed_id = ? AND created_at < ?", feed.ID, now.UTC().Add(-checkinFeedRetention)).Delete(&models.FeedMessage{})
	return result.RowsAffected, result.Error
}

func reconcilePendingCardCheckins(db *gorm.DB, secret []byte, runtime *FeedRuntime, now time.Time) (int64, error) {
	var feed models.Feed
	if result := db.Where("name = ?", checkinFeedName).First(&feed); errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return 0, nil
	} else if result.Error != nil {
		return 0, result.Error
	}
	var pending []struct {
		Fingerprint string
	}
	if err := db.Model(&models.FeedMessage{}).
		Distinct("pending_card_fingerprint AS fingerprint").
		Where("feed_id = ? AND user_id = 0 AND pending_card_fingerprint IS NOT NULL AND created_at >= ?", feed.ID, now.UTC().Add(-checkinFeedRetention)).
		Scan(&pending).Error; err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}
	wanted := make(map[string]struct{}, len(pending))
	for _, item := range pending {
		wanted[item.Fingerprint] = struct{}{}
	}
	var users []struct {
		ID     uint
		CardID *string
	}
	if err := db.Model(&models.User{}).Select("id", "card_id").Where("deleted_at IS NULL AND card_id IS NOT NULL").Scan(&users).Error; err != nil {
		return 0, err
	}
	var resolved int64
	for _, user := range users {
		if user.CardID == nil {
			continue
		}
		card, err := normalizeCheckinCard(*user.CardID)
		if err != nil {
			continue
		}
		fingerprint, err := fingerprintCheckinCardWithSecret(secret, card)
		if err != nil {
			return resolved, err
		}
		if _, exists := wanted[fingerprint]; !exists {
			continue
		}
		count, err := resolvePendingCardCheckins(db, runtime, user.ID, fingerprint, now)
		if err != nil {
			return resolved, err
		}
		resolved += count
	}
	return resolved, nil
}

func StartCheckinFeedMaintenance(db *gorm.DB, secret []byte, runtime *FeedRuntime) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	var stopOnce sync.Once
	run := func() {
		now := time.Now().UTC()
		if count, err := purgeExpiredCheckinFeedItems(db, now); err != nil {
			log.Printf("check-in feed retention sweep failed: %v", err)
		} else if count > 0 {
			log.Printf("check-in feed retention removed %d expired item(s)", count)
		}
		if count, err := reconcilePendingCardCheckins(db, secret, runtime, now); err != nil {
			log.Printf("check-in feed card-resolution sweep failed: %v", err)
		} else if count > 0 {
			log.Printf("check-in feed resolved %d earlier unknown item(s)", count)
		}
	}
	run()
	go func() {
		defer close(done)
		ticker := time.NewTicker(checkinMaintenanceInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	return func() {
		stopOnce.Do(func() { close(stop) })
		<-done
	}
}

func authorizeCardCheckin(c *fiber.Ctx) error {
	authentication := leash_auth.GetAuthentication(c)
	if !authentication.IsAPIKey() || authentication.User.Role != "service" {
		return fiber.ErrForbidden
	}
	apiKey := authentication.Data.(models.APIKey)
	if apiKey.FullAccess {
		return fiber.ErrForbidden
	}
	if err := authentication.Enforcer.Enforcer.LoadPolicy(); err != nil {
		return fiber.ErrInternalServerError
	}
	if !authentication.Enforcer.HasPermissionForAPIKey(apiKey, checkinPermission) {
		return fiber.ErrForbidden
	}
	return c.Next()
}

func registerCheckinEndpoints(api fiber.Router, runtime *FeedRuntime) {
	checkins := api.Group("/checkins")
	checkins.Get(
		"/export.csv",
		authorizeCheckinExport,
		models.GetQueryMiddleware[checkinExportRequest],
		exportCheckinsCSV,
	)
	checkins.Post(
		"/card",
		authorizeCardCheckin,
		models.GetBodyMiddleware[cardCheckinRequest],
		recordCardCheckin(runtime),
	)
}
