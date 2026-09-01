package leash_backend_api

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	noteMaximumCharacters = 1500
	noteMaximumBodyBytes  = 4096
	noteShortWindowLimit  = int64(5)
	noteDailyLimit        = int64(25)
	noteShortWindow       = 10 * time.Minute
	noteDailyWindow       = 24 * time.Hour
)

var noteIdempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type noteSubmissionRequest struct {
	Note string `json:"note"`
}

type noteRateLimitError struct {
	retryAfter time.Duration
}

func (e *noteRateLimitError) Error() string {
	return "note submission rate limit reached"
}

func noteBodyLimit(c *fiber.Ctx) error {
	if len(c.Body()) > noteMaximumBodyBytes {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "note request is too large")
	}
	if contentType := strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType))); !strings.HasPrefix(contentType, fiber.MIMEApplicationJSON) {
		return fiber.NewError(fiber.StatusUnsupportedMediaType, "note requests must be JSON")
	}
	return c.Next()
}

func parseNoteBody(c *fiber.Ctx) error {
	var request noteSubmissionRequest
	if err := c.BodyParser(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "note request is invalid")
	}
	c.Locals("body", request)
	return c.Next()
}

func normalizeNote(value string) (string, error) {
	if !utf8.ValidString(value) {
		return "", fiber.NewError(fiber.StatusBadRequest, "note must be valid UTF-8 text")
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, "note cannot be blank")
	}
	if utf8.RuneCountInString(value) > noteMaximumCharacters {
		return "", fiber.NewError(fiber.StatusBadRequest, "note cannot exceed 1500 characters")
	}
	for _, character := range value {
		if character < 0x20 && character != '\n' && character != '\t' {
			return "", fiber.NewError(fiber.StatusBadRequest, "note contains an unsupported control character")
		}
	}
	return value, nil
}

func lockNoteSubmitter(tx *gorm.DB, userID uint) error {
	query := tx.Model(&models.User{}).Select("id").Where("id = ?", userID)
	if tx.Dialector.Name() != "sqlite" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var target struct{ ID uint }
	if err := query.Scan(&target).Error; err != nil {
		return err
	}
	if target.ID == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func noteWindowRetry(tx *gorm.DB, userID uint, now time.Time, window time.Duration, limit int64) (time.Duration, error) {
	threshold := now.Add(-window)
	var count int64
	query := tx.Model(&models.NoteSubmission{}).Where("submitted_by = ? AND created_at >= ?", userID, threshold)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	if count < limit {
		return 0, nil
	}

	var oldest models.NoteSubmission
	if err := tx.Where("submitted_by = ? AND created_at >= ?", userID, threshold).Order("created_at ASC").First(&oldest).Error; err != nil {
		return 0, err
	}
	retry := oldest.CreatedAt.Add(window).Sub(now)
	if retry < time.Second {
		retry = time.Second
	}
	return retry, nil
}

func enforceNoteRateLimit(tx *gorm.DB, userID uint, now time.Time) error {
	for _, rule := range []struct {
		window time.Duration
		limit  int64
	}{
		{window: noteShortWindow, limit: noteShortWindowLimit},
		{window: noteDailyWindow, limit: noteDailyLimit},
	} {
		retry, err := noteWindowRetry(tx, userID, now, rule.window, rule.limit)
		if err != nil {
			return err
		}
		if retry > 0 {
			return &noteRateLimitError{retryAfter: retry}
		}
	}
	return nil
}

func submitNote(c *fiber.Ctx) error {
	authentication := leash_auth.GetAuthentication(c)
	if !authentication.IsUser() {
		return fiber.NewError(fiber.StatusForbidden, "notes require an authenticated person")
	}

	req := c.Locals("body").(noteSubmissionRequest)
	note, err := normalizeNote(req.Note)
	if err != nil {
		return err
	}
	idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key"))
	if !noteIdempotencyKeyPattern.MatchString(idempotencyKey) {
		return fiber.NewError(fiber.StatusBadRequest, "a valid Idempotency-Key is required")
	}

	db := leash_auth.GetDB(c)
	now := time.Now().UTC()
	user := authentication.User
	submission := models.NoteSubmission{}
	created := false
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := lockNoteSubmitter(tx, user.ID); err != nil {
			return err
		}

		result := tx.Where("submitted_by = ? AND idempotency_key = ?", user.ID, idempotencyKey).First(&submission)
		if result.Error == nil {
			if submission.Content != note {
				return fiber.NewError(fiber.StatusConflict, "Idempotency-Key was already used for a different note")
			}
			return nil
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}

		if err := enforceNoteRateLimit(tx, user.ID, now); err != nil {
			return err
		}

		submission = models.NoteSubmission{
			CreatedAt:            now,
			UpdatedAt:            now,
			SubmittedBy:          user.ID,
			SubmitterName:        strings.TrimSpace(user.Name),
			SubmitterEmail:       strings.TrimSpace(user.Email),
			Content:              note,
			IdempotencyKey:       idempotencyKey,
			DiscordNextAttemptAt: now,
		}
		if err := tx.Create(&submission).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		var rateLimit *noteRateLimitError
		if errors.As(err, &rateLimit) {
			retrySeconds := int(math.Ceil(rateLimit.retryAfter.Seconds()))
			c.Set(fiber.HeaderRetryAfter, strconv.Itoa(retrySeconds))
			return fiber.NewError(fiber.StatusTooManyRequests, "Please wait before sending another note.")
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.ErrUnauthorized
		}
		if fiberError, ok := err.(*fiber.Error); ok {
			return fiberError
		}
		return fiber.ErrInternalServerError
	}

	c.Set(fiber.HeaderCacheControl, "private, no-store")
	status := fiber.StatusOK
	if created {
		status = fiber.StatusCreated
	}
	return c.Status(status).JSON(fiber.Map{
		"id":         submission.ID,
		"created_at": submission.CreatedAt,
	})
}

func registerNoteEndpoints(api fiber.Router) {
	notes := api.Group("/notes", leash_auth.ConcatPermissionPrefixMiddleware("notes"))
	notes.Post("/", leash_auth.PrefixAuthorizationMiddleware("submit"), noteBodyLimit, parseNoteBody, submitNote)
}
