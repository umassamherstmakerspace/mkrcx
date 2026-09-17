package leash_backend_api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const printerHistoryPageSize = 20

type printerPageCursor struct {
	Printer string    `json:"p"`
	Filter  string    `json:"f"`
	At      time.Time `json:"at"`
	Kind    string    `json:"k"`
	ID      string    `json:"id"`
	Through time.Time `json:"through"`
}

type printerPageRef struct {
	At                 time.Time
	Kind, ID, PublicID string
}

// Each source reads at most one page plus a sentinel. The merge cursor orders equal
// timestamps by source type and stable ID; paging never uses an unbounded offset.
func printerPageQuery(db *gorm.DB, printer, kind, atColumn, idColumn string, cursor *printerPageCursor, through time.Time) *gorm.DB {
	q := db.Where("printer_id = ?", printer)
	upper := interface{}(through)
	if kind == "summary" {
		upper = through.Format("2006-01-02")
	}
	q = q.Where(atColumn+" <= ?", upper)
	if cursor != nil {
		at := interface{}(cursor.At)
		if kind == "summary" {
			at = cursor.At.Format("2006-01-02")
			midnight, _ := time.Parse("2006-01-02", at.(string))
			if cursor.At.After(midnight) {
				return q.Where(atColumn+" <= ?", at).Order(atColumn + " DESC, " + idColumn + " ASC").Limit(printerHistoryPageSize + 1)
			}
		}
		if kind > cursor.Kind {
			q = q.Where(atColumn+" <= ?", at)
		} else if kind < cursor.Kind {
			q = q.Where(atColumn+" < ?", at)
		} else {
			id := interface{}(cursor.ID)
			if kind == "edit" {
				id, _ = strconv.ParseUint(cursor.ID, 10, 64)
			}
			q = q.Where("("+atColumn+" < ? OR ("+atColumn+" = ? AND "+idColumn+" > ?))", at, at, id)
		}
	}
	return q.Order(atColumn + " DESC, " + idColumn + " ASC").Limit(printerHistoryPageSize + 1)
}

func printerStaffHistoryPage(c *fiber.Ctx) error {
	printer := c.Params("id")
	filter := c.Query("filter", "updates")
	if !printerIDPattern.MatchString(printer) || !printerChoice(filter, "all", "updates", "prints") {
		return fiber.ErrBadRequest
	}
	db := leash_auth.GetDB(c)
	var record models.PrinterRecord
	if err := db.First(&record, "id = ?", printer).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fiber.ErrNotFound
		}
		return fiber.ErrInternalServerError
	}
	through := time.Now().UTC()
	var cursor *printerPageCursor
	if raw := c.Query("cursor"); raw != "" {
		if len(raw) > 1200 {
			return fiber.ErrBadRequest
		}
		body, err := base64.RawURLEncoding.DecodeString(raw)
		cursor = &printerPageCursor{}
		if err != nil || json.Unmarshal(body, cursor) != nil || cursor.Printer != printer || cursor.Filter != filter || cursor.At.IsZero() || cursor.Through.IsZero() || cursor.Through.After(through.Add(time.Minute)) || cursor.At.After(cursor.Through) || !printerChoice(cursor.Kind, "event", "historical", "summary", "edit") || cursor.ID == "" || len(cursor.ID) > 160 {
			return fiber.ErrBadRequest
		}
		through = cursor.Through
	}
	var events []models.PrinterHistoryEvent
	q := printerPageQuery(db, printer, "event", "recorded_at", "source_id", cursor, through).Where("event_type <> ? AND COALESCE(hidden_reason, '') = ''", "started")
	if filter == "prints" {
		q = q.Where("event_type IN ?", []string{"printer_completed", "printer_cancelled", "printer_failed"})
	}
	if filter == "updates" {
		q = q.Where("(event_type IN ? OR (event_type IN ? AND (note_changed = ? OR (note_changed IS NULL AND detail LIKE ?))))", []string{"printer_error", "printer_failed"}, []string{"staff_runtime_changed", "printer_runtime_changed", "system_runtime_changed"}, true, "%Note:%")
	}
	// Authentication/control-only actions are retained in the database, not visible notes.
	q = q.Where("NOT (COALESCE(note_changed, 1) = 0 AND COALESCE(condition_changed, 1) = 0)")
	if q.Find(&events).Error != nil {
		return fiber.ErrInternalServerError
	}
	var history []models.PrinterHistoricalEntry
	hq := printerPageQuery(db, printer, "historical", "recorded_at", "source_id", cursor, through)
	if filter == "prints" {
		hq = hq.Where("kind = ?", "submission")
	}
	if filter == "updates" {
		hq = hq.Where("kind IN ?", []string{"service", "report", "origin", "retirement"})
	}
	if hq.Find(&history).Error != nil {
		return fiber.ErrInternalServerError
	}
	var summaries []models.PrinterSummary
	var edits []models.PrinterRecordEvent
	if filter != "prints" {
		if printerPageQuery(db, printer, "summary", "report_date", "source_id", cursor, through).Find(&summaries).Error != nil {
			return fiber.ErrInternalServerError
		}
		if printerPageQuery(db, printer, "edit", "created_at", "id", cursor, through).Find(&edits).Error != nil {
			return fiber.ErrInternalServerError
		}
	}
	refs := []printerPageRef{}
	for _, e := range events {
		refs = append(refs, printerPageRef{e.RecordedAt, "event", e.SourceID, "event:" + e.SourceID})
	}
	for _, h := range history {
		refs = append(refs, printerPageRef{h.RecordedAt, "historical", h.SourceID, "historical:" + h.SourceID})
	}
	for _, s := range summaries {
		at, _ := time.Parse("2006-01-02", s.ReportDate)
		refs = append(refs, printerPageRef{at, "summary", s.SourceID, "summary:" + s.SourceID})
	}
	for _, e := range edits {
		refs = append(refs, printerPageRef{e.CreatedAt, "edit", fmt.Sprintf("%020d", e.ID), fmt.Sprintf("edit:%d", e.Version)})
	}
	sort.Slice(refs, func(i, j int) bool {
		a, b := refs[i], refs[j]
		if !a.At.Equal(b.At) {
			return a.At.After(b.At)
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ID < b.ID
	})
	next := ""
	if len(refs) > printerHistoryPageSize {
		refs = refs[:printerHistoryPageSize]
		last := refs[len(refs)-1]
		body, _ := json.Marshal(printerPageCursor{printer, filter, last.At, last.Kind, last.ID, through})
		next = base64.RawURLEncoding.EncodeToString(body)
	}
	ids := []string{}
	selected := map[string]bool{}
	for _, r := range refs {
		ids = append(ids, r.PublicID)
		selected[r.PublicID] = true
	}
	visibleEvents := []models.PrinterHistoryEvent{}
	visibleHistory := []models.PrinterHistoricalEntry{}
	visibleSummaries := []models.PrinterSummary{}
	for _, e := range events {
		if selected["event:"+e.SourceID] {
			visibleEvents = append(visibleEvents, e)
		}
	}
	for _, h := range history {
		if selected["historical:"+h.SourceID] {
			visibleHistory = append(visibleHistory, h)
		}
	}
	for _, s := range summaries {
		if selected["summary:"+s.SourceID] {
			visibleSummaries = append(visibleSummaries, s)
		}
	}
	// Adjacent snapshots are context only, so changes at a page boundary stay accurate.
	contextEdits := map[uint64]models.PrinterRecordEvent{}
	for _, e := range edits {
		if selected[fmt.Sprintf("edit:%d", e.Version)] {
			contextEdits[e.Version] = e
			var prev models.PrinterRecordEvent
			if e.Version > 1 {
				err := db.Where("printer_id = ? AND version = ?", printer, e.Version-1).First(&prev).Error
				if err == nil {
					contextEdits[prev.Version] = prev
				} else if err != gorm.ErrRecordNotFound {
					return fiber.ErrInternalServerError
				}
			}
		}
	}
	changes := []fiber.Map{}
	for _, e := range contextEdits {
		var saved models.PrinterRecord
		if json.Unmarshal([]byte(e.Record), &saved) != nil {
			continue
		}
		name := e.ActorName
		if name == "" {
			var user struct{ Name string }
			var uid uint64
			if _, err := fmt.Sscanf(e.Actor, "user:%d", &uid); err == nil {
				db.Model(&models.User{}).Select("name").Where("id = ?", uid).Scan(&user)
				name = user.Name
			}
		}
		lifecycle, maintenance := models.PrinterStates(saved.Lifecycle, saved.Maintenance)
		changes = append(changes, fiber.Map{"recordedAt": e.CreatedAt, "actor": e.Actor, "actorName": name, "version": e.Version, "condition": saved.Condition, "note": saved.Note, "nextAction": saved.NextAction, "manual": saved.Manual, "lifecycle": lifecycle, "maintenance": maintenance, "location": saved.Location, "name": saved.Name})
	}
	usage, err := readPrinterUsage(db, printer)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	var legacy struct {
		Jobs  int64      `json:"jobs"`
		First *time.Time `json:"first"`
	}
	if db.Model(&models.PrinterHistoricalEntry{}).Where("printer_id = ? AND kind = ?", printer, "submission").Count(&legacy.Jobs).Error != nil {
		return fiber.ErrInternalServerError
	}
	var firstSubmission []models.PrinterHistoricalEntry
	if db.Where("printer_id = ? AND kind = ?", printer, "submission").Order("recorded_at ASC").Limit(1).Find(&firstSubmission).Error != nil {
		return fiber.ErrInternalServerError
	}
	if len(firstSubmission) > 0 {
		legacy.First = &firstSubmission[0].RecordedAt
	}
	var meters, origins []models.PrinterHistoricalEntry
	if db.Where("printer_id = ? AND meter_hours IS NOT NULL", printer).Order("recorded_at DESC, source_id ASC").Find(&meters).Error != nil {
		return fiber.ErrInternalServerError
	}
	if db.Where("printer_id = ? AND kind = ?", printer, "origin").Order("recorded_at DESC").Limit(5).Find(&origins).Error != nil {
		return fiber.ErrInternalServerError
	}
	estimate, err := readPrinterHistoryEstimate(db, printer, usage, legacy.Jobs, legacy.First, meters, origins)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	if err := applyPrinterDisplayNames(db, visibleEvents, visibleHistory, changes); err != nil {
		return fiber.ErrInternalServerError
	}
	if len(meters) > 1 {
		meters = meters[:1]
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(fiber.Map{"events": visibleEvents, "historical": visibleHistory, "summaries": visibleSummaries, "edits": changes, "pageIds": ids, "nextCursor": next, "usage": usage, "estimate": estimate, "legacy": legacy, "meters": meters, "origins": origins, "lastSync": record.HistorySyncedAt})
}
