package leash_backend_api

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const printerFreshness = 90 * time.Second

var printerIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)

type printerEdit struct {
	Version   uint64 `json:"version"`
	Name      string `json:"name"`
	Model     string `json:"model"`
	MachineID string `json:"machineId"`
	Location  string `json:"location"`
	Lifecycle string `json:"lifecycle"`
	Host      string `json:"host"`
	MAC       string `json:"mac"`
	Condition string `json:"condition"`
	Note      string `json:"note"`
	Manual    bool   `json:"manual"`
}
type printerJob struct {
	Person   string `json:"person"`
	File     string `json:"file"`
	Material string `json:"material"`
	Started  string `json:"started"`
}
type printerReading struct {
	ID        string      `json:"id"`
	Condition string      `json:"condition"`
	Activity  string      `json:"activity"`
	Note      string      `json:"note,omitempty"`
	Minutes   *float64    `json:"minutes,omitempty"`
	Progress  *float64    `json:"progress,omitempty"`
	Job       *printerJob `json:"job,omitempty"`
}
type printerSnapshot struct {
	FetchedAt time.Time        `json:"fetchedAt"`
	Printers  []printerReading `json:"printers"`
}

func printerChoice(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
func printerText(value string, limit int) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > limit {
		return false
	}
	for _, r := range value {
		if r < 32 && r != '\n' && r != '\t' {
			return false
		}
	}
	return true
}
func decodePrinterJSON(c *fiber.Ctx, target interface{}, maximum int) error {
	if len(c.Body()) > maximum {
		return fiber.ErrRequestEntityTooLarge
	}
	if !strings.HasPrefix(strings.ToLower(c.Get("Content-Type")), "application/json") {
		return fiber.ErrUnsupportedMediaType
	}
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fiber.NewError(400, "Invalid printer data")
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return fiber.NewError(400, "Invalid printer data")
	}
	return nil
}

func printerCollectorAuth(c *fiber.Ctx) error {
	expected := os.Getenv("PRINTER_FLEET_INGEST_SECRET")
	if expected == "" || subtle.ConstantTimeCompare([]byte(c.Get("Authorization")), []byte("Bearer "+expected)) != 1 {
		return fiber.ErrUnauthorized
	}
	return c.Next()
}
func printerPermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := leash_auth.GetAuthentication(c).Authorize(permission); err != nil {
			return fiber.ErrForbidden
		}
		return c.Next()
	}
}

// Public/collector routes precede the normal API authentication middleware.
func registerPrinterPublicEndpoints(api fiber.Router) {
	api.Get("/printer-fleet", func(c *fiber.Ctx) error { return respondPrinterFleet(c, false) })
	api.Post("/printer-fleet/ingest", printerCollectorAuth, ingestPrinterFleet)
	api.Get("/printer-fleet/roster", printerCollectorAuth, printerRoster)
}
func registerPrinterEndpoints(api fiber.Router) {
	api.Get("/printer-fleet/staff", printerPermission("leash.printers:read"), func(c *fiber.Ctx) error { return respondPrinterFleet(c, true) })
	api.Get("/printer-fleet/records", printerPermission("leash.printers:manage"), listPrinterRecords)
	api.Put("/printer-fleet/records/:id", printerPermission("leash.printers:manage"), savePrinterRecord)
	api.Get("/printer-fleet/records/:id/history", printerPermission("leash.printers:manage"), printerHistory)
}

func listPrinterRecords(c *fiber.Ctx) error {
	var records []models.PrinterRecord
	if err := leash_auth.GetDB(c).Order("name ASC").Find(&records).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	for i := range records {
		if !records[i].Manual {
			if records[i].ObservedCondition != "" {
				records[i].Condition = records[i].ObservedCondition
			}
			records[i].Note = records[i].ObservedNote
		}
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(records)
}
func printerHistory(c *fiber.Ctx) error {
	var events []models.PrinterRecordEvent
	if err := leash_auth.GetDB(c).Where("printer_id = ?", c.Params("id")).Order("id DESC").Limit(100).Find(&events).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(events)
}
func savePrinterRecord(c *fiber.Ctx) error {
	var edit printerEdit
	if err := decodePrinterJSON(c, &edit, 16384); err != nil {
		return err
	}
	id := c.Params("id")
	edit.Name = strings.TrimSpace(edit.Name)
	edit.Model = strings.TrimSpace(edit.Model)
	edit.Host = strings.TrimSpace(edit.Host)
	edit.MAC = strings.ToLower(strings.TrimSpace(edit.MAC))
	if !printerIDPattern.MatchString(id) || edit.Name == "" || edit.Model == "" ||
		!printerText(edit.Name, 120) || !printerText(edit.Model, 80) || !printerText(edit.Location, 120) || !printerText(edit.MachineID, 120) || !printerText(edit.Note, 2000) ||
		!printerChoice(edit.Lifecycle, "active", "testing", "repair", "retired") || !printerChoice(edit.Condition, "working", "limited", "out", "unknown") {
		return fiber.NewError(400, "Invalid printer record")
	}
	// Only a verified LAN IPv4 target and a unicast hardware identity can be polled.
	if edit.Host != "" {
		ip := net.ParseIP(edit.Host).To4()
		if ip == nil || ip[0] != 192 || ip[1] != 168 || ip[2] != 1 || ip[3] == 0 || ip[3] == 255 || edit.MAC == "" {
			return fiber.NewError(400, "Use a printer-subnet address (192.168.1.1–254) with its hardware MAC")
		}
	}
	if edit.MAC != "" {
		mac, err := net.ParseMAC(edit.MAC)
		if err != nil || len(mac) != 6 || mac[0]&1 != 0 {
			return fiber.NewError(400, "A unicast hardware MAC is required")
		}
		edit.MAC = mac.String()
	}
	auth := leash_auth.GetAuthentication(c)
	actor := fmt.Sprintf("user:%d", auth.User.ID)
	if auth.IsAPIKey() {
		actor = fmt.Sprintf("service-user:%d", auth.User.ID)
	}
	var saved models.PrinterRecord
	err := leash_auth.GetDB(c).Transaction(func(tx *gorm.DB) error {
		var current models.PrinterRecord
		found := tx.First(&current, "id = ?", id).Error
		if found != nil && !errors.Is(found, gorm.ErrRecordNotFound) {
			return found
		}
		if (found == nil && current.Version != edit.Version) || (found != nil && edit.Version != 0) {
			return fiber.NewError(409, "Printer was changed; reload before saving")
		}
		if current.MAC != "" && edit.MAC != current.MAC {
			return fiber.NewError(409, "A replacement machine needs its own printer record; hardware identity cannot be reassigned")
		}
		if edit.MAC != "" {
			var count int64
			if err := tx.Model(&models.PrinterRecord{}).Where("id <> ? AND (mac = ? OR (host <> '' AND host = ?))", id, edit.MAC, edit.Host).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return fiber.NewError(409, "Hardware identity or address already belongs to another printer")
			}
		}
		saved = current
		saved.ID = id
		saved.Name = edit.Name
		saved.Model = edit.Model
		saved.MachineID = edit.MachineID
		saved.Location = edit.Location
		saved.Lifecycle = edit.Lifecycle
		saved.Host = edit.Host
		saved.MAC = edit.MAC
		saved.Condition = edit.Condition
		saved.Note = edit.Note
		saved.Manual = edit.Manual
		saved.Version = edit.Version + 1
		saved.UpdatedAt = time.Now().UTC()
		saved.UpdatedBy = actor
		saved.HostKey = nil
		saved.MACKey = nil
		if saved.Host != "" {
			saved.HostKey = &saved.Host
		}
		if saved.MAC != "" {
			saved.MACKey = &saved.MAC
		}
		if found != nil {
			if err := tx.Create(&saved).Error; err != nil {
				return fiber.NewError(409, "Printer already exists or conflicts with another record")
			}
		} else {
			// Explicit fields prevent telemetry ingest and record edits from overwriting one another.
			result := tx.Model(&models.PrinterRecord{}).Where("id = ? AND version = ?", id, edit.Version).Updates(map[string]interface{}{
				"name": saved.Name, "model": saved.Model, "machine_id": saved.MachineID, "location": saved.Location, "lifecycle": saved.Lifecycle,
				"host": saved.Host, "mac": saved.MAC, "host_key": saved.HostKey, "mac_key": saved.MACKey, "condition": saved.Condition, "note": saved.Note, "manual": saved.Manual, "version": saved.Version, "updated_at": saved.UpdatedAt, "updated_by": actor,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return fiber.NewError(409, "Printer was changed; reload before saving")
			}
		}
		body, _ := json.Marshal(saved)
		return tx.Create(&models.PrinterRecordEvent{PrinterID: id, Version: saved.Version, Actor: actor, CreatedAt: saved.UpdatedAt, Record: string(body)}).Error
	})
	if err != nil {
		if e, ok := err.(*fiber.Error); ok {
			return e
		}
		return fiber.ErrInternalServerError
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(saved)
}

func printerRoster(c *fiber.Ctx) error {
	var records []models.PrinterRecord
	if err := leash_auth.GetDB(c).Where("lifecycle <> ? AND host <> ''", "retired").Order("id").Find(&records).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	roster := make([]fiber.Map, 0, len(records))
	for _, p := range records {
		roster = append(roster, fiber.Map{"id": p.ID, "host": p.Host, "mac": p.MAC})
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(fiber.Map{"printers": roster})
}

func ingestPrinterFleet(c *fiber.Ctx) error {
	var snapshot printerSnapshot
	if err := decodePrinterJSON(c, &snapshot, 1048576); err != nil {
		return err
	}
	now := time.Now().UTC()
	if snapshot.FetchedAt.IsZero() || now.Sub(snapshot.FetchedAt) > printerFreshness || snapshot.FetchedAt.Sub(now) > 5*time.Second || len(snapshot.Printers) > 200 {
		return fiber.NewError(400, "Invalid snapshot timestamp or size")
	}
	seen := map[string]bool{}
	for _, p := range snapshot.Printers {
		if seen[p.ID] || !printerIDPattern.MatchString(p.ID) || !printerChoice(p.Condition, "working", "limited", "out", "unknown") || !printerChoice(p.Activity, "idle", "printing", "paused", "unknown") || !printerText(p.Note, 2000) {
			return fiber.NewError(400, "Invalid printer reading")
		}
		seen[p.ID] = true
		if (p.Minutes != nil && (*p.Minutes < 0 || *p.Minutes > 525600)) || (p.Progress != nil && (*p.Progress < 0 || *p.Progress > 100)) {
			return fiber.NewError(400, "Invalid progress")
		}
		if p.Job != nil && (!printerText(p.Job.Person, 200) || !printerText(p.Job.File, 1000) || !printerText(p.Job.Material, 120) || !printerText(p.Job.Started, 120)) {
			return fiber.NewError(400, "Invalid job")
		}
	}
	err := leash_auth.GetDB(c).Transaction(func(tx *gorm.DB) error {
		for _, p := range snapshot.Printers {
			var record models.PrinterRecord
			if err := tx.First(&record, "id = ?", p.ID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fiber.NewError(400, "Unknown printer identity")
				}
				return err
			}
			if record.FetchedAt != nil && !snapshot.FetchedAt.After(*record.FetchedAt) {
				continue
			}
			body, _ := json.Marshal(p)
			updates := map[string]interface{}{"fetched_at": snapshot.FetchedAt, "telemetry": string(body)}
			if p.Activity != "unknown" {
				updates["last_seen"] = snapshot.FetchedAt
			}
			// Connectivity failures are observations, never instructions to clear a repair record.
			if p.Condition != "unknown" {
				updates["observed_condition"] = p.Condition
				updates["observed_note"] = p.Note
				updates["condition_observed_at"] = snapshot.FetchedAt
			}
			if err := tx.Model(&models.PrinterRecord{}).Where("id = ? AND (fetched_at IS NULL OR fetched_at < ?)", p.ID, snapshot.FetchedAt).UpdateColumns(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if e, ok := err.(*fiber.Error); ok {
			return e
		}
		return fiber.ErrInternalServerError
	}
	return c.SendStatus(204)
}

func respondPrinterFleet(c *fiber.Ctx, staff bool) error {
	var records []models.PrinterRecord
	if err := leash_auth.GetDB(c).Where("lifecycle <> ?", "retired").Order("name").Find(&records).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	now := time.Now().UTC()
	anyFresh := false
	var latest *time.Time
	printers := make([]fiber.Map, 0, len(records))
	for _, p := range records {
		fresh := p.FetchedAt != nil && now.Sub(*p.FetchedAt) <= printerFreshness
		if fresh {
			anyFresh = true
		}
		if p.FetchedAt != nil && (latest == nil || p.FetchedAt.After(*latest)) {
			latest = p.FetchedAt
		}
		condition, note, source, updated := p.ObservedCondition, p.ObservedNote, "station", p.ConditionObservedAt
		if p.Manual {
			condition = p.Condition
			note = p.Note
			source = "record"
			updated = &p.UpdatedAt
		}
		if condition == "" {
			condition = "unknown"
		}
		var reading printerReading
		_ = json.Unmarshal([]byte(p.Telemetry), &reading)
		activity := "unknown"
		if fresh && reading.Activity != "" {
			activity = reading.Activity
		}
		item := fiber.Map{"id": p.ID, "name": p.Name, "model": p.Model, "machineId": p.MachineID, "location": p.Location, "lifecycle": p.Lifecycle, "condition": condition, "note": note, "conditionSource": source, "conditionUpdatedAt": updated, "lastSeen": p.LastSeen, "activity": activity, "stale": !fresh, "connected": fresh && activity != "unknown"}
		if fresh && printerChoice(activity, "printing", "paused") {
			if reading.Minutes != nil {
				item["minutes"] = reading.Minutes
			}
			if staff && reading.Job != nil {
				item["job"] = reading.Job
				item["progress"] = reading.Progress
			}
		}
		printers = append(printers, item)
	}
	audience := "public"
	if staff {
		audience = "staff"
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(fiber.Map{"audience": audience, "stale": !anyFresh, "fetchedAt": latest, "printers": printers})
}
