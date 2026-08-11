package leash_backend_api

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const (
	checkinExportPermission = "leash.checkins:export"
	checkinEventSource      = "card-server"
	maxCheckinExportRange   = 370 * 24 * time.Hour
	maxCheckinExportRows    = 250000
)

type checkinExportRequest struct {
	Start string `query:"start" validate:"required"`
	End   string `query:"end" validate:"required"`
}

func getOrCreateCheckinIdentity(db *gorm.DB, userID uint) (string, error) {
	if userID == 0 {
		return "", nil
	}

	var identity models.CheckinIdentity
	result := db.Where("user_id = ?", userID).First(&identity)
	if result.Error == nil {
		return identity.MemberUUID, nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return "", result.Error
	}

	identity = models.CheckinIdentity{UserID: userID, MemberUUID: uuid.NewString()}
	if err := db.Create(&identity).Error; err == nil {
		return identity.MemberUUID, nil
	}

	// Another request may have created the identity concurrently. The existing
	// row is authoritative; a UUID is never regenerated for an existing user.
	if err := db.Where("user_id = ?", userID).First(&identity).Error; err != nil {
		return "", err
	}
	return identity.MemberUUID, nil
}

func checkinEventFromFeedItem(db *gorm.DB, item models.FeedMessage) (models.CheckinEvent, error) {
	if item.IdempotencyKey == nil || strings.TrimSpace(*item.IdempotencyKey) == "" {
		return models.CheckinEvent{}, errors.New("check-in event requires an idempotency key")
	}
	if item.CreatedAt.IsZero() {
		return models.CheckinEvent{}, errors.New("check-in event requires an occurrence time")
	}

	linkedAtTap := item.UserID != 0 && item.Title != cardNotLinkedTitle
	resolution := "unresolved"
	if linkedAtTap {
		resolution = "tap_time"
	} else if item.UserID != 0 {
		resolution = "later_link"
	}

	memberUUID, err := getOrCreateCheckinIdentity(db, item.UserID)
	if err != nil {
		return models.CheckinEvent{}, err
	}

	return models.CheckinEvent{
		OccurredAt:         item.CreatedAt.UTC(),
		UserID:             item.UserID,
		MemberUUID:         memberUUID,
		LinkedAtTap:        linkedAtTap,
		IdentityResolution: resolution,
		Decision:           decisionFromFeedItem(item),
		Source:             checkinEventSource,
		IdempotencyScope:   item.IdempotencyScope,
		IdempotencyKey:     strings.TrimSpace(*item.IdempotencyKey),
	}, nil
}

func sameCheckinEvent(first, second models.CheckinEvent) bool {
	return first.OccurredAt.Equal(second.OccurredAt) &&
		first.UserID == second.UserID &&
		first.MemberUUID == second.MemberUUID &&
		first.LinkedAtTap == second.LinkedAtTap &&
		first.IdentityResolution == second.IdentityResolution &&
		first.Decision == second.Decision &&
		first.Source == second.Source &&
		first.IdempotencyScope == second.IdempotencyScope &&
		first.IdempotencyKey == second.IdempotencyKey
}

func persistCheckinEvent(db *gorm.DB, item *models.FeedMessage) error {
	candidate, err := checkinEventFromFeedItem(db, *item)
	if err != nil {
		return err
	}

	var existing models.CheckinEvent
	result := db.Where(
		"idempotency_scope = ? AND idempotency_key = ?",
		candidate.IdempotencyScope, candidate.IdempotencyKey,
	).First(&existing)
	if result.Error == nil {
		if !sameCheckinEvent(existing, candidate) {
			return errors.New("idempotent check-in event content changed")
		}
		return nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}

	if err := db.Create(&candidate).Error; err == nil {
		return nil
	}
	if err := db.Where(
		"idempotency_scope = ? AND idempotency_key = ?",
		candidate.IdempotencyScope, candidate.IdempotencyKey,
	).First(&existing).Error; err != nil {
		return err
	}
	if !sameCheckinEvent(existing, candidate) {
		return errors.New("idempotent check-in event content changed")
	}
	return nil
}

func resolveCheckinEventIdentity(db *gorm.DB, item models.FeedMessage, userID uint) error {
	if item.IdempotencyKey == nil || strings.TrimSpace(*item.IdempotencyKey) == "" {
		return nil
	}
	if err := persistCheckinEvent(db, &item); err != nil {
		return err
	}
	memberUUID, err := getOrCreateCheckinIdentity(db, userID)
	if err != nil {
		return err
	}
	return db.Model(&models.CheckinEvent{}).
		Where("idempotency_scope = ? AND idempotency_key = ?", item.IdempotencyScope, strings.TrimSpace(*item.IdempotencyKey)).
		Updates(map[string]interface{}{
			"user_id":             userID,
			"member_uuid":         memberUUID,
			"identity_resolution": "later_link",
		}).Error
}

func authorizeCheckinExport(c *fiber.Ctx) error {
	authentication := leash_auth.GetAuthentication(c)
	if !authentication.IsUser() {
		return fiber.ErrForbidden
	}
	if err := authentication.Enforcer.Enforcer.LoadPolicy(); err != nil {
		return fiber.ErrInternalServerError
	}
	if !authentication.Enforcer.HasPermissionForUser(authentication.User, checkinExportPermission) {
		return fiber.ErrForbidden
	}
	return c.Next()
}

func parseCheckinExportRange(req checkinExportRequest) (time.Time, time.Time, error) {
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(req.Start))
	if err != nil {
		return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "start must be an RFC3339 timestamp")
	}
	end, err := time.Parse(time.RFC3339, strings.TrimSpace(req.End))
	if err != nil {
		return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "end must be an RFC3339 timestamp")
	}
	start, end = start.UTC(), end.UTC()
	if !end.After(start) {
		return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "end must be after start")
	}
	if end.Sub(start) > maxCheckinExportRange {
		return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "export range cannot exceed 370 days")
	}
	return start, end, nil
}

func exportCheckinsCSV(c *fiber.Ctx) error {
	req := c.Locals("query").(checkinExportRequest)
	start, end, err := parseCheckinExportRange(req)
	if err != nil {
		return err
	}

	db := leash_auth.GetDB(c)
	var count int64
	query := db.Model(&models.CheckinEvent{}).
		Where("occurred_at >= ? AND occurred_at < ?", start, end)
	if err := query.Count(&count).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	if count > maxCheckinExportRows {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "export contains too many rows; choose a smaller date range")
	}

	var events []models.CheckinEvent
	if err := query.Order("occurred_at ASC, id ASC").Find(&events).Error; err != nil {
		return fiber.ErrInternalServerError
	}

	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write([]string{
		"event_id",
		"occurred_at_utc",
		"member_uuid",
		"linked_at_tap",
		"identity_resolution",
		"outcome",
		"source",
	}); err != nil {
		return fiber.ErrInternalServerError
	}
	for _, event := range events {
		if err := writer.Write([]string{
			strconv.FormatUint(uint64(event.ID), 10),
			event.OccurredAt.UTC().Format(time.RFC3339Nano),
			event.MemberUUID,
			strconv.FormatBool(event.LinkedAtTap),
			event.IdentityResolution,
			event.Decision,
			event.Source,
		}); err != nil {
			return fiber.ErrInternalServerError
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fiber.ErrInternalServerError
	}

	authentication := leash_auth.GetAuthentication(c)
	audit := models.CheckinExportAudit{
		RequestedBy: authentication.User.ID,
		StartAt:     start,
		EndAt:       end,
		RowCount:    count,
	}
	if err := db.Create(&audit).Error; err != nil {
		return fiber.ErrInternalServerError
	}

	filename := fmt.Sprintf("checkins_%s_%s.csv", start.Format("20060102"), end.Format("20060102"))
	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Send(output.Bytes())
}
