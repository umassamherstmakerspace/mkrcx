package leash_backend_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type printerSummaryInput struct {
	SourceID   string                 `json:"sourceId"`
	ReportDate string                 `json:"reportDate"`
	Body       string                 `json:"body"`
	Sources    []models.PrinterSource `json:"sources"`
}

// An operator API for reviewed standup summaries; there is no website entry form.
func importPrinterSummary(c *fiber.Ctx) error {
	id := c.Params("id")
	var input printerSummaryInput
	if err := decodePrinterJSON(c, &input, 24000); err != nil {
		return err
	}
	input.Body = strings.TrimSpace(input.Body)
	report, err := time.Parse("2006-01-02", input.ReportDate)
	if !printerIDPattern.MatchString(id) || !strings.HasPrefix(input.SourceID, "standup:") || len(input.SourceID) <= 8 || !printerText(input.SourceID, 160) || input.Body == "" || !printerText(input.Body, 4000) || err != nil || report.After(time.Now().UTC()) || len(input.Sources) == 0 || len(input.Sources) > 10 {
		return fiber.NewError(400, "A dated summary and its sources are required")
	}
	for _, source := range input.Sources {
		u, err := url.Parse(source.URL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || !printerText(source.URL, 2000) || !printerText(source.Label, 200) || source.Label == "" {
			return fiber.NewError(400, "Invalid summary source")
		}
	}
	db := leash_auth.GetDB(c)
	var record models.PrinterRecord
	if err := db.First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.ErrNotFound
		}
		return fiber.ErrInternalServerError
	}
	auth := leash_auth.GetAuthentication(c)
	entry := models.PrinterSummary{SourceID: input.SourceID, PrinterID: id, ReportDate: input.ReportDate, Body: input.Body, Sources: input.Sources, PreparedBy: auth.User.Name, Actor: fmt.Sprintf("user:%d", auth.User.ID), ImportedAt: time.Now().UTC()}
	if auth.IsAPIKey() {
		entry.Actor = fmt.Sprintf("service-user:%d", auth.User.ID)
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&entry).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	var stored models.PrinterSummary
	if err := db.First(&stored, "source_id = ?", input.SourceID).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	a, _ := json.Marshal(stored.Sources)
	b, _ := json.Marshal(input.Sources)
	if stored.PrinterID != id || stored.ReportDate != input.ReportDate || stored.Body != input.Body || string(a) != string(b) {
		return fiber.NewError(409, "This source ID already has a different summary; use a new ID for a correction")
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(stored)
}

type printerUsage struct {
	Seconds          float64    `json:"seconds"`
	Jobs             int64      `json:"jobs"`
	MissingDurations int64      `json:"missingDurations"`
	FirstOutcome     *time.Time `json:"firstOutcome"`
}

func readPrinterUsage(db *gorm.DB, id string) (printerUsage, error) {
	var usage printerUsage
	err := db.Model(&models.PrinterHistoryEvent{}).
		Select("COALESCE(SUM(duration_seconds),0) AS seconds, COUNT(*) AS jobs, COALESCE(SUM(CASE WHEN duration_seconds IS NULL THEN 1 ELSE 0 END),0) AS missing_durations").
		Where("printer_id = ? AND event_type IN ?", id, []string{"printer_completed", "printer_cancelled", "printer_failed"}).Scan(&usage).Error
	if err != nil {
		return usage, err
	}
	var first models.PrinterHistoryEvent
	err = db.Where("printer_id = ? AND event_type IN ?", id, []string{"printer_completed", "printer_cancelled", "printer_failed"}).Order("recorded_at ASC").Limit(1).Find(&first).Error
	if !first.RecordedAt.IsZero() {
		usage.FirstOutcome = &first.RecordedAt
	}
	return usage, err
}
