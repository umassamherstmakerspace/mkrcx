package leash_backend_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

var noteTestDatabaseSequence uint64

func noteTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:notes-%d-%d?mode=memory&cache=shared", time.Now().UnixNano(), atomic.AddUint64(&noteTestDatabaseSequence, 1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.NoteSubmission{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func noteTestApp(t *testing.T, authenticator leash_auth.Authenticator) (*fiber.App, *gorm.DB, models.User) {
	t.Helper()
	db := noteTestDatabase(t)
	user := models.User{Email: "student@example.edu", Name: "Student Name", Role: "member", Type: "student"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Post("/notes", func(c *fiber.Ctx) error {
		c.Locals("db", db)
		c.Locals("auth", leash_auth.Authentication{Authenticator: authenticator, User: user})
		return c.Next()
	}, noteBodyLimit, models.GetBodyMiddleware[noteSubmissionRequest], submitNote)
	return app, db, user
}

func postNote(t *testing.T, app *fiber.App, note, key string) *http.Response {
	t.Helper()
	body, err := json.Marshal(map[string]string{"note": note})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/notes", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestSubmitNoteRecordsIdentitySnapshotAndIsIdempotent(t *testing.T) {
	app, db, user := noteTestApp(t, leash_auth.AUTHENTICATOR_USER)
	response := postNote(t, app, "  First line\r\nSecond line  ", "note-attempt-1")
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("first submission returned %d", response.StatusCode)
	}

	var stored models.NoteSubmission
	if err := db.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.SubmittedBy != user.ID || stored.SubmitterName != user.Name || stored.SubmitterEmail != user.Email {
		t.Fatalf("identity snapshot mismatch: %+v", stored)
	}
	if stored.Content != "First line\nSecond line" || stored.DiscordNextAttemptAt.IsZero() {
		t.Fatalf("stored note mismatch: %+v", stored)
	}

	duplicate := postNote(t, app, "First line\nSecond line", "note-attempt-1")
	if duplicate.StatusCode != fiber.StatusOK {
		t.Fatalf("idempotent retry returned %d", duplicate.StatusCode)
	}
	var count int64
	if err := db.Model(&models.NoteSubmission{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("idempotent retry created %d rows", count)
	}

	conflict := postNote(t, app, "Different text", "note-attempt-1")
	if conflict.StatusCode != fiber.StatusConflict {
		t.Fatalf("changed idempotent retry returned %d", conflict.StatusCode)
	}
}

func TestSubmitNoteRejectsAPIKeysOversizeAndControls(t *testing.T) {
	apiKeyApp, _, _ := noteTestApp(t, leash_auth.AUTHENTICATOR_APIKEY)
	if response := postNote(t, apiKeyApp, "Valid note", "api-key-note"); response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("API key submission returned %d", response.StatusCode)
	}

	app, _, _ := noteTestApp(t, leash_auth.AUTHENTICATOR_USER)
	for name, note := range map[string]string{
		"too long": strings.Repeat("a", noteMaximumCharacters+1),
		"control":  "unsafe\x00text",
		"blank":    "  \n\t  ",
	} {
		t.Run(name, func(t *testing.T) {
			response := postNote(t, app, note, "invalid-"+strings.ReplaceAll(name, " ", "-"))
			if response.StatusCode != fiber.StatusBadRequest {
				t.Fatalf("invalid note returned %d", response.StatusCode)
			}
		})
	}
}

func TestSubmitNoteEnforcesPerAccountRateLimits(t *testing.T) {
	app, db, user := noteTestApp(t, leash_auth.AUTHENTICATOR_USER)
	now := time.Now().UTC()
	for index := 0; index < int(noteShortWindowLimit); index++ {
		note := models.NoteSubmission{
			CreatedAt: now.Add(-time.Duration(index) * time.Second), UpdatedAt: now,
			SubmittedBy: user.ID, SubmitterName: user.Name, SubmitterEmail: user.Email,
			Content: fmt.Sprintf("Existing %d", index), IdempotencyKey: fmt.Sprintf("existing-%d", index),
			DiscordNextAttemptAt: now,
		}
		if err := db.Create(&note).Error; err != nil {
			t.Fatal(err)
		}
	}
	response := postNote(t, app, "One too many", "limited-attempt")
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("rate-limited submission returned %d", response.StatusCode)
	}
	retry, err := strconv.Atoi(response.Header.Get("Retry-After"))
	if err != nil || retry < 1 || retry > int(noteShortWindow.Seconds()) {
		t.Fatalf("invalid Retry-After %q", response.Header.Get("Retry-After"))
	}
	var count int64
	if err := db.Model(&models.NoteSubmission{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != noteShortWindowLimit {
		t.Fatalf("rate-limited request changed row count to %d", count)
	}
}

func TestNoteBodyLimitAndContentType(t *testing.T) {
	app, _, _ := noteTestApp(t, leash_auth.AUTHENTICATOR_USER)
	large := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader(strings.Repeat("x", noteMaximumBodyBytes+1)))
	large.Header.Set("Content-Type", "application/json")
	response, err := app.Test(large)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusRequestEntityTooLarge {
		t.Fatalf("oversize body returned %d", response.StatusCode)
	}

	wrongType := httptest.NewRequest(http.MethodPost, "/notes", strings.NewReader("note"))
	wrongType.Header.Set("Content-Type", "text/plain")
	response, err = app.Test(wrongType)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusUnsupportedMediaType {
		t.Fatalf("wrong content type returned %d", response.StatusCode)
	}
}
