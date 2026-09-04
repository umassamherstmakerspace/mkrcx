package leash_backend_api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

func docusignTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Hold{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func docusignTestConfig() docusignConnectConfig {
	return docusignConnectConfig{
		hmacSecret:    []byte("test-connect-secret"),
		accountID:     "account-123",
		subjectPrefix: "UMass Makerspace Use Agreement |",
		recipientRole: "Member",
		enabled:       true,
	}
}

func docusignPayload(t *testing.T, email string, completedAt time.Time) []byte {
	t.Helper()
	payload := docusignConnectPayload{Event: "envelope-completed"}
	payload.Data.AccountID = "account-123"
	payload.Data.EnvelopeID = "f4efc042-4348-45c9-a747-a8be673780e3"
	payload.Data.EnvelopeSummary.Status = "completed"
	payload.Data.EnvelopeSummary.EmailSubject = "UMass Makerspace Use Agreement | member@example.edu | Member Name"
	payload.Data.EnvelopeSummary.CompletedDateTime = completedAt.Format(time.RFC3339Nano)
	payload.Data.EnvelopeSummary.Recipients.Signers = append(payload.Data.EnvelopeSummary.Recipients.Signers, struct {
		Email    string `json:"email"`
		RoleName string `json:"roleName"`
		Status   string `json:"status"`
	}{Email: email, RoleName: "Member", Status: "completed"})
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func docusignSignature(body, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func postDocusign(t *testing.T, db *gorm.DB, cfg docusignConnectConfig, body []byte, signature string) (int, string) {
	t.Helper()
	app := fiber.New()
	app.Post("/api/webhooks/docusign", func(c *fiber.Ctx) error {
		return handleDocusignConnect(db, cfg, c)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/docusign", bytes.NewReader(body))
	req.Header.Set("Content-Type", fiber.MIMEApplicationJSON)
	req.Header.Set(docusignSignatureHeader, signature)
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(responseBody)
}

func TestDocusignCompletionClearsOnlyTheEligibleHold(t *testing.T) {
	db := docusignTestDB(t)
	cfg := docusignTestConfig()
	completedAt := time.Now().UTC().Truncate(time.Second)
	user := models.User{Email: "member@example.edu", Name: "Member Name", Pronouns: "they/them", Role: "member", Type: "other"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	hold := models.Hold{Model: models.Model{CreatedAt: completedAt.Add(-time.Hour)}, UserID: user.ID, Name: docusignHoldName, AddedBy: 42, Priority: 1}
	if err := db.Create(&hold).Error; err != nil {
		t.Fatal(err)
	}
	body := docusignPayload(t, "MEMBER@example.edu", completedAt)
	status, response := postDocusign(t, db, cfg, body, docusignSignature(body, cfg.hmacSecret))
	if status != fiber.StatusOK || response != `{"status":"resolved"}` {
		t.Fatalf("completion returned %d %s", status, response)
	}

	var cleared models.Hold
	if err := db.Unscoped().First(&cleared, hold.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !cleared.DeletedAt.Valid || !cleared.DeletedAt.Time.Equal(completedAt) || cleared.RemovedBy != hold.AddedBy {
		t.Fatalf("hold was not cleared at completion: %+v", cleared)
	}

	newHold := models.Hold{Model: models.Model{CreatedAt: completedAt.Add(time.Minute)}, UserID: user.ID, Name: docusignHoldName, AddedBy: 42, Priority: 1}
	if err := db.Create(&newHold).Error; err != nil {
		t.Fatal(err)
	}
	status, response = postDocusign(t, db, cfg, body, docusignSignature(body, cfg.hmacSecret))
	if status != fiber.StatusOK || response != `{"status":"already_resolved"}` {
		t.Fatalf("replay returned %d %s", status, response)
	}
	var active int64
	if err := db.Model(&models.Hold{}).Where("id = ?", newHold.ID).Count(&active).Error; err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatal("an old completion replay cleared a newer hold")
	}
}

func TestDocusignCompletionRejectsInvalidSignature(t *testing.T) {
	db := docusignTestDB(t)
	cfg := docusignTestConfig()
	body := docusignPayload(t, "member@example.edu", time.Now().UTC())
	status, _ := postDocusign(t, db, cfg, body, "invalid")
	if status != fiber.StatusUnauthorized {
		t.Fatalf("invalid signature returned %d", status)
	}
}

func TestDocusignCompletionIgnoresOtherAccountsAndSubjects(t *testing.T) {
	db := docusignTestDB(t)
	cfg := docusignTestConfig()
	body := docusignPayload(t, "member@example.edu", time.Now().UTC())
	var payload docusignConnectPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*docusignConnectPayload){
		"account": func(value *docusignConnectPayload) { value.Data.AccountID = "other-account" },
		"subject": func(value *docusignConnectPayload) { value.Data.EnvelopeSummary.EmailSubject = "Unrelated envelope" },
	} {
		t.Run(name, func(t *testing.T) {
			copyPayload := payload
			mutate(&copyPayload)
			candidate, err := json.Marshal(copyPayload)
			if err != nil {
				t.Fatal(err)
			}
			status, response := postDocusign(t, db, cfg, candidate, docusignSignature(candidate, cfg.hmacSecret))
			if status != fiber.StatusNoContent || response != "" {
				t.Fatalf("ignored completion returned %d %q", status, response)
			}
		})
	}
}

func TestDocusignCompletionReportsUnmatchedMember(t *testing.T) {
	db := docusignTestDB(t)
	cfg := docusignTestConfig()
	body := docusignPayload(t, "missing@example.edu", time.Now().UTC())
	status, response := postDocusign(t, db, cfg, body, docusignSignature(body, cfg.hmacSecret))
	if status != fiber.StatusAccepted || response != `{"status":"unmatched"}` {
		t.Fatalf("unmatched completion returned %d %s", status, response)
	}
}

func TestDocusignCompletionRequiresCompleteSignerData(t *testing.T) {
	db := docusignTestDB(t)
	cfg := docusignTestConfig()
	body := docusignPayload(t, "member@example.edu", time.Now().UTC())
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	data := payload["data"].(map[string]interface{})
	summary := data["envelopeSummary"].(map[string]interface{})
	summary["recipients"] = map[string]interface{}{"signers": []interface{}{}}
	candidate, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	status, response := postDocusign(t, db, cfg, candidate, docusignSignature(candidate, cfg.hmacSecret))
	if status != fiber.StatusUnprocessableEntity || !strings.Contains(response, "completed waiver signer not found") {
		t.Fatalf("missing signer returned %d %s", status, response)
	}
}

func TestDocusignCompletionRejectsMultipleMatchingSigners(t *testing.T) {
	db := docusignTestDB(t)
	cfg := docusignTestConfig()
	body := docusignPayload(t, "member@example.edu", time.Now().UTC())
	var payload docusignConnectPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatal(err)
	}
	payload.Data.EnvelopeSummary.Recipients.Signers = append(
		payload.Data.EnvelopeSummary.Recipients.Signers,
		payload.Data.EnvelopeSummary.Recipients.Signers[0],
	)
	candidate, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	status, response := postDocusign(t, db, cfg, candidate, docusignSignature(candidate, cfg.hmacSecret))
	if status != fiber.StatusUnprocessableEntity || !strings.Contains(response, "multiple completed waiver signers") {
		t.Fatalf("multiple signers returned %d %s", status, response)
	}
}

func Example_validDocusignSignature() {
	body := []byte(`{"event":"envelope-completed"}`)
	secret := []byte("secret")
	fmt.Println(validDocusignSignature(body, secret, docusignSignature(body, secret)))
	// Output: true
}
