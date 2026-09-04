package leash_backend_api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const docusignSignatureHeader = "X-DocuSign-Signature-1"

type docusignConnectConfig struct {
	hmacSecret    []byte
	accountID     string
	subjectPrefix string
	recipientRole string
	enabled       bool
}

type docusignConnectPayload struct {
	Event string `json:"event"`
	Data  struct {
		AccountID       string `json:"accountId"`
		EnvelopeID      string `json:"envelopeId"`
		EnvelopeSummary struct {
			Status            string `json:"status"`
			EmailSubject      string `json:"emailSubject"`
			CompletedDateTime string `json:"completedDateTime"`
			Recipients        struct {
				Signers []struct {
					Email    string `json:"email"`
					RoleName string `json:"roleName"`
					Status   string `json:"status"`
				} `json:"signers"`
			} `json:"recipients"`
		} `json:"envelopeSummary"`
	} `json:"data"`
}

func docusignConnectConfigFromEnv() docusignConnectConfig {
	secret := strings.TrimSpace(os.Getenv("DOCUSIGN_CONNECT_HMAC_SECRET"))
	accountID := strings.TrimSpace(os.Getenv("DOCUSIGN_ACCOUNT_ID"))
	subjectPrefix := strings.TrimSpace(os.Getenv("DOCUSIGN_WAIVER_SUBJECT_PREFIX"))
	recipientRole := strings.TrimSpace(os.Getenv("DOCUSIGN_WAIVER_RECIPIENT_ROLE"))

	values := []string{secret, accountID, subjectPrefix, recipientRole}
	configured := 0
	for _, value := range values {
		if value != "" {
			configured++
		}
	}

	return docusignConnectConfig{
		hmacSecret:    []byte(secret),
		accountID:     accountID,
		subjectPrefix: subjectPrefix,
		recipientRole: recipientRole,
		enabled:       configured == len(values),
	}
}

func validDocusignSignature(body, secret []byte, signature string) bool {
	provided, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

func normalizeDocusignEmail(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	address, err := mail.ParseAddress(raw)
	if err != nil || !strings.EqualFold(address.Address, raw) {
		return "", errors.New("invalid signer email")
	}
	return strings.ToLower(address.Address), nil
}

func completedDocusignSigner(payload docusignConnectPayload, role string) (string, error) {
	var email string
	matches := 0
	for _, signer := range payload.Data.EnvelopeSummary.Recipients.Signers {
		if !strings.EqualFold(strings.TrimSpace(signer.RoleName), role) ||
			!strings.EqualFold(strings.TrimSpace(signer.Status), "completed") {
			continue
		}
		normalized, err := normalizeDocusignEmail(signer.Email)
		if err != nil {
			return "", err
		}
		matches++
		if matches > 1 {
			return "", errors.New("multiple completed waiver signers")
		}
		email = normalized
	}
	if email == "" {
		return "", errors.New("completed waiver signer not found")
	}
	return email, nil
}

func clearDocusignHold(db *gorm.DB, email string, completedAt time.Time) (string, error) {
	var user struct {
		ID uint
	}
	if err := db.Table("users").Select("id").
		Where("deleted_at IS NULL").
		Where("LOWER(email) = ?", email).
		Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "unmatched", nil
		}
		return "", err
	}

	var hold struct {
		ID      uint
		AddedBy uint
	}
	if err := db.Unscoped().Table("holds").Select("id", "added_by").
		Where("user_id = ? AND name = ?", user.ID, docusignHoldName).
		Where("deleted_at IS NULL").
		Where("created_at <= ?", completedAt).
		Order("created_at DESC").
		Take(&hold).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "already_resolved", nil
		}
		return "", err
	}

	result := db.Unscoped().Model(&models.Hold{}).
		Where("id = ? AND deleted_at IS NULL", hold.ID).
		Updates(map[string]interface{}{
			"deleted_at": completedAt,
			"removed_by": hold.AddedBy,
		})
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		return "already_resolved", nil
	}
	return "resolved", nil
}

func handleDocusignConnect(db *gorm.DB, cfg docusignConnectConfig, c *fiber.Ctx) error {
	if !cfg.enabled {
		return fiber.NewError(fiber.StatusServiceUnavailable, "DocuSign completion handling is not configured")
	}
	body := c.Body()
	if !validDocusignSignature(body, cfg.hmacSecret, c.Get(docusignSignatureHeader)) {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid DocuSign signature")
	}

	var payload docusignConnectPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid DocuSign payload")
	}
	if !strings.EqualFold(strings.TrimSpace(payload.Event), "envelope-completed") {
		return c.SendStatus(fiber.StatusNoContent)
	}
	if payload.Data.AccountID != cfg.accountID ||
		!strings.HasPrefix(payload.Data.EnvelopeSummary.EmailSubject, cfg.subjectPrefix) {
		return c.SendStatus(fiber.StatusNoContent)
	}
	if !strings.EqualFold(strings.TrimSpace(payload.Data.EnvelopeSummary.Status), "completed") ||
		strings.TrimSpace(payload.Data.EnvelopeID) == "" {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "Incomplete DocuSign completion payload")
	}

	completedAt, err := time.Parse(time.RFC3339Nano, payload.Data.EnvelopeSummary.CompletedDateTime)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "Invalid DocuSign completion time")
	}
	email, err := completedDocusignSigner(payload, cfg.recipientRole)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	status, err := clearDocusignHold(db, email, completedAt.UTC())
	if err != nil {
		return fiber.ErrInternalServerError
	}
	responseStatus := fiber.StatusOK
	if status == "unmatched" {
		responseStatus = fiber.StatusAccepted
	}
	return c.Status(responseStatus).JSON(fiber.Map{"status": status})
}

func registerDocusignConnectEndpoint(api fiber.Router) {
	cfg := docusignConnectConfigFromEnv()
	api.Post("/webhooks/docusign", func(c *fiber.Ctx) error {
		return handleDocusignConnect(leash_auth.GetDB(c), cfg, c)
	})
}
