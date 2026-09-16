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

const printerFreshness = 90 * time.Second

var printerIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)

type printerEdit struct {
	Version     uint64  `json:"version"`
	Name        string  `json:"name"`
	Model       string  `json:"model"`
	MachineID   string  `json:"machineId"`
	Location    string  `json:"location"`
	Lifecycle   string  `json:"lifecycle"`
	Maintenance *string `json:"maintenance,omitempty"`
	Host        string  `json:"host"`
	MAC         string  `json:"mac"`
	Condition   string  `json:"condition"`
	Note        string  `json:"note"`
	NextAction  *string `json:"nextAction,omitempty"`
	Manual      bool    `json:"manual"`
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
	Fault     string      `json:"fault,omitempty"`
}
type printerSnapshot struct {
	FetchedAt time.Time                    `json:"fetchedAt"`
	Printers  []printerReading             `json:"printers"`
	History   []models.PrinterHistoryEvent `json:"history,omitempty"`
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
	api.Get("/printer-fleet/history/:id", printerPermission("leash.printers:read"), printerStaffHistory)
	api.Get("/printer-fleet/staff", printerPermission("leash.printers:read"), func(c *fiber.Ctx) error { return respondPrinterFleet(c, true) })
	api.Get("/printer-fleet/records", printerPermission("leash.printers:manage"), listPrinterRecords)
	api.Put("/printer-fleet/records/:id", printerPermission("leash.printers:manage"), savePrinterRecord)
	api.Get("/printer-fleet/records/:id/history", printerPermission("leash.printers:manage"), printerHistory)
}

func printerStaffHistory(c *fiber.Ctx) error {
	if !printerIDPattern.MatchString(c.Params("id")) {
		return fiber.ErrBadRequest
	}
	db := leash_auth.GetDB(c)
	var record models.PrinterRecord
	if err := db.First(&record, "id = ?", c.Params("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.ErrNotFound
		}
		return fiber.ErrInternalServerError
	}
	var events []models.PrinterHistoryEvent
	var edits []models.PrinterRecordEvent
	if err := db.Where("printer_id = ? AND event_type <> ?", record.ID, "started").Order("recorded_at DESC, source_id DESC").Limit(100).Find(&events).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	if err := db.Where("printer_id = ?", record.ID).Order("id DESC").Limit(50).Find(&edits).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	// Resolve old account IDs; new edits retain the name at the time of the edit.
	userIDs := []uint64{}
	actorIDs := map[string]uint64{}
	for _, edit := range edits {
		if edit.ActorName != "" {
			continue
		}
		kind, value, ok := strings.Cut(edit.Actor, ":")
		if !ok || (kind != "user" && kind != "service-user") {
			continue
		}
		id, err := strconv.ParseUint(value, 10, 64)
		if err == nil {
			actorIDs[edit.Actor] = id
			userIDs = append(userIDs, id)
		}
	}
	names := map[uint64]string{}
	if len(userIDs) > 0 {
		var users []struct {
			ID   uint64
			Name string
		}
		// Scan only display fields, without User's permission-loading AfterFind hook.
		if err := db.Model(&models.User{}).Unscoped().Select("id", "name").Where("id IN ?", userIDs).Scan(&users).Error; err != nil {
			return fiber.ErrInternalServerError
		}
		for _, user := range users {
			names[user.ID] = user.Name
		}
	}
	changes := make([]fiber.Map, 0, len(edits))
	for _, edit := range edits {
		var saved models.PrinterRecord
		if json.Unmarshal([]byte(edit.Record), &saved) != nil {
			continue
		}
		lifecycle, maintenance := models.PrinterStates(saved.Lifecycle, saved.Maintenance)
		actorName := edit.ActorName
		if actorName == "" {
			actorName = names[actorIDs[edit.Actor]]
		}
		changes = append(changes, fiber.Map{"recordedAt": edit.CreatedAt, "actor": edit.Actor, "actorName": actorName, "version": edit.Version, "condition": saved.Condition, "note": saved.Note, "nextAction": saved.NextAction, "manual": saved.Manual, "lifecycle": lifecycle, "maintenance": maintenance, "location": saved.Location, "name": saved.Name})
	}
	c.Set("Cache-Control", "private, no-store")
	return c.JSON(fiber.Map{"events": events, "edits": changes, "lastSync": record.HistorySyncedAt, "stationCondition": record.ObservedCondition, "stationNote": record.ObservedNote, "stationReportedAt": record.ConditionObservedAt})
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
	// Older clients can still submit a combined state, but cannot silently clear maintenance.
	if edit.Lifecycle == "testing" || edit.Lifecycle == "repair" {
		legacy := edit.Lifecycle
		if edit.Maintenance != nil && *edit.Maintenance != legacy {
			return fiber.NewError(400, "Conflicting printer states")
		}
		edit.Maintenance = &legacy
		edit.Lifecycle = "active"
	}
	if !printerIDPattern.MatchString(id) || edit.Name == "" || edit.Model == "" ||
		!printerText(edit.Name, 120) || !printerText(edit.Model, 80) || !printerText(edit.Location, 120) || !printerText(edit.MachineID, 120) || !printerText(edit.Note, 2000) ||
		(edit.NextAction != nil && !printerText(*edit.NextAction, 2000)) ||
		!printerChoice(edit.Lifecycle, "active", "shelved", "retired") || !printerChoice(edit.Condition, "working", "limited", "out", "unknown") ||
		(edit.Maintenance != nil && !printerChoice(*edit.Maintenance, "none", "diagnosis", "repair", "testing")) {
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
		_, saved.Maintenance = models.PrinterStates(current.Lifecycle, current.Maintenance)
		if edit.Maintenance != nil {
			saved.Maintenance = *edit.Maintenance
		}
		saved.Host = edit.Host
		saved.MAC = edit.MAC
		saved.Condition = edit.Condition
		saved.Note = edit.Note
		if edit.NextAction != nil {
			saved.NextAction = *edit.NextAction
		}
		saved.Manual = true
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
				"name": saved.Name, "model": saved.Model, "machine_id": saved.MachineID, "location": saved.Location, "lifecycle": saved.Lifecycle, "maintenance": saved.Maintenance,
				"host": saved.Host, "mac": saved.MAC, "host_key": saved.HostKey, "mac_key": saved.MACKey, "condition": saved.Condition, "note": saved.Note, "next_action": saved.NextAction, "manual": saved.Manual, "version": saved.Version, "updated_at": saved.UpdatedAt, "updated_by": actor,
			})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return fiber.NewError(409, "Printer was changed; reload before saving")
			}
		}
		body, _ := json.Marshal(saved)
		return tx.Create(&models.PrinterRecordEvent{PrinterID: id, Version: saved.Version, Actor: actor, ActorName: auth.User.Name, CreatedAt: saved.UpdatedAt, Record: string(body)}).Error
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
	// A shelved printer may be powered on at the repair bench. Keep reading known hosts.
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
	if len(snapshot.History) > 1000 {
		return fiber.ErrRequestEntityTooLarge
	}
	for _, event := range snapshot.History {
		if !printerText(event.PreviousCondition, 80) ||
			((event.NoteChanged != nil || event.ConditionChanged != nil || event.PreviousCondition != "") && !printerChoice(event.EventType, "staff_runtime_changed", "printer_runtime_changed", "system_runtime_changed")) {
			return fiber.NewError(400, "Invalid printer change details")
		}
		if !printerText(event.ActorName, 200) || !printerChoice(event.ActorMethod, "", "ucard", "local_pin", "printer_api") || (event.ActorName != "" && event.ActorMethod != "ucard") {
			return fiber.NewError(400, "Invalid printer event attribution")
		}
		if !printerText(event.Person, 200) || (event.DurationSeconds != nil && (*event.DurationSeconds < 0 || *event.DurationSeconds > 31536000)) {
			return fiber.NewError(400, "Invalid printer job details")
		}
		if !printerIDPattern.MatchString(event.PrinterID) || !printerText(event.SourceID, 160) || !strings.HasPrefix(event.SourceID, "station:") || !printerText(event.EventType, 80) || event.EventType == "" || !printerText(event.Detail, 4000) || !printerText(event.File, 1000) || !printerText(event.Material, 120) || event.RecordedAt.IsZero() || event.RecordedAt.After(now.Add(5*time.Second)) {
			return fiber.NewError(400, "Invalid printer history event")
		}
	}
	for _, p := range snapshot.Printers {
		if seen[p.ID] || !printerIDPattern.MatchString(p.ID) || !printerChoice(p.Condition, "working", "limited", "out", "unknown") || !printerChoice(p.Activity, "idle", "printing", "paused", "unknown") || !printerText(p.Note, 2000) || !printerText(p.Fault, 4000) {
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
			var previous printerReading
			_ = json.Unmarshal([]byte(record.Telemetry), &previous)
			if p.Fault != "" && previous.Fault != p.Fault {
				event := models.PrinterHistoryEvent{SourceID: fmt.Sprintf("observer:%s:%d", p.ID, snapshot.FetchedAt.UnixNano()), PrinterID: p.ID, RecordedAt: snapshot.FetchedAt, EventType: "printer_error", Detail: p.Fault}
				if err := tx.Create(&event).Error; err != nil {
					return err
				}
			}
			updates := map[string]interface{}{"fetched_at": snapshot.FetchedAt, "telemetry": string(body)}
			if p.Activity != "unknown" {
				updates["last_seen"] = snapshot.FetchedAt
			}
			// Connectivity failures are observations, never instructions to clear a repair record.
			if p.Condition != "unknown" && (p.Condition != record.ObservedCondition || p.Note != record.ObservedNote || record.ConditionObservedAt == nil) {
				updates["observed_condition"] = p.Condition
				updates["observed_note"] = p.Note
				updates["condition_observed_at"] = snapshot.FetchedAt
			}
			if err := tx.Model(&models.PrinterRecord{}).Where("id = ? AND (fetched_at IS NULL OR fetched_at < ?)", p.ID, snapshot.FetchedAt).UpdateColumns(updates).Error; err != nil {
				return err
			}
		}
		for _, event := range snapshot.History {
			if !seen[event.PrinterID] {
				return fiber.NewError(400, "History must belong to an observed printer")
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&event).Error; err != nil {
				return err
			}
			// Older projections omitted these fields. Fill missing metadata only, and
			// require the same printer, type and timestamp before enriching an event.
			// MariaDB stores datetime(3); allow sub-millisecond truncation/rounding.
			match := func() *gorm.DB {
				return tx.Model(&models.PrinterHistoryEvent{}).Where("source_id = ? AND printer_id = ? AND event_type = ? AND recorded_at > ? AND recorded_at < ?", event.SourceID, event.PrinterID, event.EventType, event.RecordedAt.Add(-time.Millisecond), event.RecordedAt.Add(time.Millisecond))
			}
			if event.Person != "" {
				if err := match().Where("person IS NULL OR person = ''").UpdateColumn("person", event.Person).Error; err != nil {
					return err
				}
			}
			if event.DurationSeconds != nil {
				if err := match().Where("duration_seconds IS NULL").UpdateColumn("duration_seconds", *event.DurationSeconds).Error; err != nil {
					return err
				}
			}
			// Keep attribution together so a later import cannot relabel a PIN edit.
			if event.ActorMethod != "" {
				if err := match().Where("(actor_method IS NULL OR actor_method = '') AND (actor_name IS NULL OR actor_name = '')").UpdateColumns(map[string]interface{}{"actor_method": event.ActorMethod, "actor_name": event.ActorName}).Error; err != nil {
					return err
				}
			}
			if event.NoteChanged != nil {
				if err := match().Where("note_changed IS NULL").UpdateColumn("note_changed", *event.NoteChanged).Error; err != nil {
					return err
				}
			}
			if event.ConditionChanged != nil {
				if err := match().Where("condition_changed IS NULL").UpdateColumns(map[string]interface{}{"condition_changed": *event.ConditionChanged, "previous_condition": event.PreviousCondition}).Error; err != nil {
					return err
				}
			}
		}
		if snapshot.History != nil {
			for id := range seen {
				if err := tx.Model(&models.PrinterRecord{}).Where("id = ? AND (history_synced_at IS NULL OR history_synced_at < ?)", id, snapshot.FetchedAt).UpdateColumn("history_synced_at", snapshot.FetchedAt).Error; err != nil {
					return err
				}
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
	query := leash_auth.GetDB(c).Order("name")
	if !staff {
		query = query.Where("lifecycle NOT IN ?", []string{"shelved", "retired"})
	}
	if err := query.Find(&records).Error; err != nil {
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
		if reading.Fault != "" {
			activity = "unknown"
		}
		lifecycle, maintenance := models.PrinterStates(p.Lifecycle, p.Maintenance)
		item := fiber.Map{"id": p.ID, "name": p.Name, "model": p.Model, "machineId": p.MachineID, "location": p.Location, "lifecycle": lifecycle, "maintenance": maintenance, "condition": condition, "note": note, "conditionSource": source, "conditionUpdatedAt": updated, "lastSeen": p.LastSeen, "activity": activity, "stale": !fresh, "connected": fresh && activity != "unknown"}
		if staff {
			item["nextAction"] = p.NextAction
		}
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
