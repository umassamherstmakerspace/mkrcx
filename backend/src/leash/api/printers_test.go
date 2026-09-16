package leash_backend_api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

func printerTestApp(t *testing.T) (*fiber.App, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:printers-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = models.MigratePrinterRegistry(db); err != nil {
		t.Fatal(err)
	}
	if err = db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("db", db)
		c.Locals("auth", leash_auth.Authentication{Authenticator: leash_auth.AUTHENTICATOR_USER, User: models.User{ID: 42, Name: "Staff fixture"}})
		return c.Next()
	})
	registerPrinterPublicEndpoints(app)
	app.Put("/records/:id", savePrinterRecord)
	app.Get("/staff", func(c *fiber.Ctx) error { return respondPrinterFleet(c, true) })
	app.Get("/history/:id", printerStaffHistory)
	app.Get("/denied", func(c *fiber.Ctx) error {
		c.Locals("auth", leash_auth.Authentication{Authenticator: leash_auth.AUTHENTICATOR_LOGGED_OUT})
		return c.Next()
	}, printerPermission("leash.printers:manage"), listPrinterRecords)
	t.Setenv("PRINTER_FLEET_INGEST_SECRET", "test-collector")
	return app, db
}
func printerRequest(t *testing.T, app *fiber.App, method, path string, body interface{}, token string) (int, map[string]interface{}) {
	t.Helper()
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var result map[string]interface{}
	_ = json.NewDecoder(res.Body).Decode(&result)
	return res.StatusCode, result
}
func printerItem(t *testing.T, result map[string]interface{}, id string) map[string]interface{} {
	t.Helper()
	for _, item := range result["printers"].([]interface{}) {
		p := item.(map[string]interface{})
		if p["id"] == id {
			return p
		}
	}
	t.Fatalf("missing printer %s", id)
	return nil
}
func editFor(name string) printerEdit {
	return printerEdit{Name: name, Model: "New model", Lifecycle: "testing", Condition: "out", Note: "Awaiting thermistor", Manual: true}
}

func TestPrinterShelvingAndHistorySurviveOffline(t *testing.T) {
	app, db := printerTestApp(t)
	edit := editFor("History fixture")
	if status, _ := printerRequest(t, app, "PUT", "/records/history-fixture", edit, ""); status != 200 {
		t.Fatal(status)
	}
	now := time.Now().UTC()
	snapshot := printerSnapshot{FetchedAt: now, Printers: []printerReading{{ID: "history-fixture", Condition: "working", Activity: "idle", Fault: "Heater not heating"}}, History: []models.PrinterHistoryEvent{{SourceID: "station:job:10", PrinterID: "history-fixture", RecordedAt: now.Add(-time.Hour), EventType: "printer_failed", Detail: "Recorded failure", File: "staff-only.gcode"}}}
	for i := 0; i < 2; i++ {
		snapshot.FetchedAt = now.Add(time.Duration(i) * time.Millisecond)
		if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 204 {
			t.Fatal(status)
		}
	}
	var count int64
	db.Model(&models.PrinterHistoryEvent{}).Count(&count)
	if count != 2 {
		t.Fatalf("duplicate fault or imported event: %d", count)
	}
	_, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	p := printerItem(t, public, "history-fixture")
	if p["fault"] != nil || p["history"] != nil || p["activity"] != "unknown" {
		t.Fatalf("unsafe public fault: %#v", p)
	}
	edit.Version = 1
	edit.Lifecycle = "shelved"
	if status, _ := printerRequest(t, app, "PUT", "/records/history-fixture", edit, ""); status != 200 {
		t.Fatal(status)
	}
	_, public = printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	for _, item := range public["printers"].([]interface{}) {
		if item.(map[string]interface{})["id"] == "history-fixture" {
			t.Fatal("shelved printer public")
		}
	}
	_, staff := printerRequest(t, app, "GET", "/staff", nil, "")
	if printerItem(t, staff, "history-fixture")["note"] != edit.Note {
		t.Fatal("shelved note lost")
	}
	snapshot.FetchedAt = now.Add(2 * time.Millisecond)
	snapshot.History = nil
	snapshot.Printers[0] = printerReading{ID: "history-fixture", Condition: "unknown", Activity: "unknown"}
	printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector")
	if err := models.MigratePrinterRegistry(db); err != nil {
		t.Fatal(err)
	}
	status, history := printerRequest(t, app, "GET", "/history/history-fixture", nil, "")
	if status != 200 || len(history["events"].([]interface{})) != 2 || len(history["edits"].([]interface{})) != 2 || history["lastSync"] == nil {
		t.Fatalf("history missing: %#v", history)
	}
	serialized, _ := json.Marshal(history)
	if bytes.Contains(serialized, []byte("host")) || bytes.Contains(serialized, []byte("mac")) {
		t.Fatal("connection details leaked in history")
	}
	snapshot.History = []models.PrinterHistoryEvent{{SourceID: "station:job:11", PrinterID: "another-printer", RecordedAt: now, EventType: "printer_failed"}}
	if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 400 {
		t.Fatal("accepted mismatched event", status)
	}
}

func TestPrinterHistoryEnrichmentPreservesEventsAndPrivacy(t *testing.T) {
	app, db := printerTestApp(t)
	printerRequest(t, app, "PUT", "/records/enrichment", editFor("Enrichment"), "")
	now := time.Now().UTC()
	event := models.PrinterHistoryEvent{SourceID: "station:job:40", PrinterID: "enrichment", RecordedAt: now.Add(-time.Hour), EventType: "printer_completed", File: "private.gcode", Detail: "Print duration: 125 min"}
	snapshot := printerSnapshot{FetchedAt: now, Printers: []printerReading{{ID: "enrichment", Condition: "working", Activity: "idle"}}, History: []models.PrinterHistoryEvent{event}}
	send := func() {
		t.Helper()
		if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 204 {
			t.Fatal(status)
		}
	}
	send()
	duration := 7510.0
	// MariaDB datetime(3) truncates station timestamps to millisecond precision.
	db.Model(&models.PrinterHistoryEvent{}).Where("source_id = ?", event.SourceID).UpdateColumn("recorded_at", event.RecordedAt.Truncate(time.Millisecond))
	snapshot.History[0].Person = "Private student"
	snapshot.History[0].DurationSeconds = &duration
	send()
	send()
	var stored models.PrinterHistoryEvent
	db.First(&stored, "source_id = ?", event.SourceID)
	if stored.Person != "Private student" || stored.DurationSeconds == nil || *stored.DurationSeconds != duration || stored.Detail != event.Detail {
		t.Fatal("metadata not enriched safely", stored)
	}
	var count int64
	db.Model(&models.PrinterHistoryEvent{}).Count(&count)
	if count != 1 {
		t.Fatal("duplicate event", count)
	}
	snapshot.History[0].Person = "Replacement name"
	send()
	db.First(&stored, "source_id = ?", event.SourceID)
	if stored.Person != "Private student" {
		t.Fatal("overwrote populated metadata")
	}
	_, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	body, _ := json.Marshal(public)
	if bytes.Contains(body, []byte("Private student")) || bytes.Contains(body, []byte("private.gcode")) {
		t.Fatal("job data leaked publicly")
	}
	_, history := printerRequest(t, app, "GET", "/history/enrichment", nil, "")
	job := history["events"].([]interface{})[0].(map[string]interface{})
	if job["person"] != "Private student" || job["durationSeconds"] != duration {
		t.Fatal("staff history missing job metadata")
	}
	// A colliding source ID cannot enrich a different event.
	snapshot.History[0].SourceID = "station:job:41"
	snapshot.History[0].Person = ""
	snapshot.History[0].DurationSeconds = nil
	send()
	snapshot.History[0].Person = "Wrong job"
	snapshot.History[0].RecordedAt = now.Add(-2 * time.Hour)
	send()
	var collision models.PrinterHistoryEvent
	db.First(&collision, "source_id = ?", "station:job:41")
	if collision.Person != "" {
		t.Fatal("enriched mismatched event")
	}
}

func TestPrinterNotesSurviveOfflineRestartAndStaleTelemetry(t *testing.T) {
	app, db := printerTestApp(t)
	edit := editFor("Replacement")
	if status, _ := printerRequest(t, app, "PUT", "/records/replacement", edit, ""); status != 200 {
		t.Fatal(status)
	}
	reading := printerReading{ID: "replacement", Condition: "working", Activity: "printing", Note: "Older station note", Job: &printerJob{Person: "Private student", File: "private.gcode"}}
	if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", printerSnapshot{FetchedAt: time.Now().UTC(), Printers: []printerReading{reading}}, "test-collector"); status != 204 {
		t.Fatal(status)
	}
	_, result := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	p := printerItem(t, result, "replacement")
	if p["note"] != edit.Note || p["condition"] != "out" || p["job"] != nil || p["progress"] != nil || p["host"] != nil {
		t.Fatalf("public record %#v", p)
	}
	_, staff := printerRequest(t, app, "GET", "/staff", nil, "")
	if printerItem(t, staff, "replacement")["job"] == nil {
		t.Fatal("fresh staff job missing")
	}
	reading.Condition = "unknown"
	reading.Activity = "unknown"
	reading.Note = "Printer status unavailable."
	reading.Job = nil
	printerRequest(t, app, "POST", "/printer-fleet/ingest", printerSnapshot{FetchedAt: time.Now().UTC().Add(time.Millisecond), Printers: []printerReading{reading}}, "test-collector")
	db.Model(&models.PrinterRecord{}).Where("id = ?", "replacement").UpdateColumn("fetched_at", time.Now().Add(-2*time.Minute))
	if err := models.MigratePrinterRegistry(db); err != nil {
		t.Fatal(err)
	}
	_, result = printerRequest(t, app, "GET", "/staff", nil, "")
	p = printerItem(t, result, "replacement")
	if p["note"] != edit.Note || p["condition"] != "out" || p["activity"] != "unknown" || p["job"] != nil || p["lastSeen"] == nil {
		t.Fatalf("stale record %#v", p)
	}
}
func TestPrinterObservedNoteSurvivesOfflineWithoutManualOverride(t *testing.T) {
	app, db := printerTestApp(t)
	// A legacy row before its assessment baseline is initialized.
	db.Model(&models.PrinterRecord{}).Where("id = ?", "k1c-1f44").UpdateColumn("manual", false)
	now := time.Now().UTC()
	for i, p := range []printerReading{{ID: "k1c-1f44", Condition: "limited", Activity: "idle", Note: "No lights"}, {ID: "k1c-1f44", Condition: "unknown", Activity: "unknown", Note: "Printer status unavailable."}} {
		status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", printerSnapshot{FetchedAt: now.Add(time.Duration(i) * time.Millisecond), Printers: []printerReading{p}}, "test-collector")
		if status != 204 {
			t.Fatal(status)
		}
	}
	_, result := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	p := printerItem(t, result, "k1c-1f44")
	if p["note"] != "No lights" || p["condition"] != "limited" || p["connected"] != false {
		t.Fatalf("lost condition %#v", p)
	}
}
func TestPrinterEditsConflictRetainHistoryAndRetireWithoutDeletion(t *testing.T) {
	app, db := printerTestApp(t)
	edit := editFor("Bench printer")
	printerRequest(t, app, "PUT", "/records/bench", edit, "")
	if status, _ := printerRequest(t, app, "PUT", "/records/bench", edit, ""); status != 409 {
		t.Fatal("missing conflict", status)
	}
	edit.Version = 1
	edit.Lifecycle = "retired"
	if status, _ := printerRequest(t, app, "PUT", "/records/bench", edit, ""); status != 200 {
		t.Fatal(status)
	}
	_, result := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	for _, item := range result["printers"].([]interface{}) {
		if item.(map[string]interface{})["id"] == "bench" {
			t.Fatal("retired printer public")
		}
	}
	var events int64
	db.Model(&models.PrinterRecordEvent{}).Where("printer_id = ?", "bench").Count(&events)
	if events != 2 {
		t.Fatal(events)
	}
	var saved models.PrinterRecord
	db.First(&saved, "id = ?", "bench")
	if saved.Note != edit.Note {
		t.Fatal("retirement lost note")
	}
}
func TestPrinterIngestRejectsUnauthorizedUnknownDuplicateStaleAndBadValues(t *testing.T) {
	app, _ := printerTestApp(t)
	p := printerReading{ID: "k1c-1f44", Condition: "working", Activity: "idle"}
	snapshot := printerSnapshot{FetchedAt: time.Now().UTC(), Printers: []printerReading{p}}
	if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, ""); status != 401 {
		t.Fatal(status)
	}
	if status, _ := printerRequest(t, app, "GET", "/denied", nil, ""); status != 403 {
		t.Fatal(status)
	}
	for _, s := range []printerSnapshot{
		{FetchedAt: time.Now().UTC(), Printers: []printerReading{p, p}},
		{FetchedAt: time.Now().Add(-2 * time.Minute), Printers: []printerReading{p}},
		{FetchedAt: time.Now().UTC(), Printers: []printerReading{{ID: "unregistered", Condition: "working", Activity: "idle"}}},
		{FetchedAt: time.Now().UTC(), Printers: []printerReading{{ID: p.ID, Condition: "working", Activity: "invented"}}},
	} {
		if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", s, "test-collector"); status != 400 {
			t.Fatal(status)
		}
	}
}
func TestPrinterHardwareCannotBeReusedOrReassigned(t *testing.T) {
	app, _ := printerTestApp(t)
	edit := editFor("New")
	edit.Host = "127.0.0.1"
	edit.MAC = "fc:ee:28:00:30:eb"
	if status, _ := printerRequest(t, app, "PUT", "/records/new", edit, ""); status != 400 {
		t.Fatal(status)
	}
	edit.Host = "192.168.1.159"
	if status, _ := printerRequest(t, app, "PUT", "/records/new", edit, ""); status != 409 {
		t.Fatal(status)
	}
	edit.Version = 1
	edit.Host = "192.168.1.160"
	edit.MAC = "fc:ee:28:00:30:aa"
	if status, _ := printerRequest(t, app, "PUT", "/records/k1-30eb", edit, ""); status != 409 {
		t.Fatal(status)
	}
}

func TestPrinterFleetAndMaintenanceAreIndependent(t *testing.T) {
	app, db := printerTestApp(t)
	maintenance := "repair"
	edit := editFor("Bench repair")
	edit.Lifecycle, edit.Maintenance, edit.Location = "shelved", &maintenance, "Repair bench"
	edit.Host, edit.MAC = "192.168.1.250", "02:11:22:33:44:55"
	status, saved := printerRequest(t, app, "PUT", "/records/bench-repair", edit, "")
	if status != 200 || saved["lifecycle"] != "shelved" || saved["maintenance"] != "repair" || saved["condition"] != "out" {
		t.Fatalf("states lost: %d %#v", status, saved)
	}
	_, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	for _, p := range public["printers"].([]interface{}) {
		if p.(map[string]interface{})["id"] == "bench-repair" {
			t.Fatal("shelved printer public")
		}
	}
	_, roster := printerRequest(t, app, "GET", "/printer-fleet/roster", nil, "test-collector")
	if printerItem(t, roster, "bench-repair")["host"] != edit.Host {
		t.Fatal("repair-bench telemetry excluded")
	}
	_, staff := printerRequest(t, app, "GET", "/staff", nil, "")
	if printerItem(t, staff, "bench-repair")["maintenance"] != "repair" {
		t.Fatal("staff maintenance missing")
	}
	// A client that predates the maintenance field must not clear it on an unrelated edit.
	edit.Version, edit.Maintenance = 1, nil
	status, saved = printerRequest(t, app, "PUT", "/records/bench-repair", edit, "")
	if status != 200 || saved["maintenance"] != "repair" {
		t.Fatal("legacy client lost maintenance", saved)
	}
	reading := printerReading{ID: "bench-repair", Condition: "unknown", Activity: "unknown"}
	printerRequest(t, app, "POST", "/printer-fleet/ingest", printerSnapshot{FetchedAt: time.Now().UTC(), Printers: []printerReading{reading}}, "test-collector")
	if err := models.MigratePrinterRegistry(db); err != nil {
		t.Fatal(err)
	}
	_, staff = printerRequest(t, app, "GET", "/staff", nil, "")
	p := printerItem(t, staff, "bench-repair")
	if p["maintenance"] != "repair" || p["condition"] != "out" || p["note"] != edit.Note || p["location"] != "Repair bench" {
		t.Fatal("offline/restart lost fields", p)
	}
	maintenance = "none"
	edit.Version, edit.Maintenance = 2, &maintenance
	status, saved = printerRequest(t, app, "PUT", "/records/bench-repair", edit, "")
	if status != 200 || saved["maintenance"] != "none" || saved["lifecycle"] != "shelved" || saved["condition"] != "out" {
		t.Fatal("clear maintenance affected other states", saved)
	}
	edit.Version, edit.Lifecycle = 3, "retired"
	printerRequest(t, app, "PUT", "/records/bench-repair", edit, "")
	_, roster = printerRequest(t, app, "GET", "/printer-fleet/roster", nil, "test-collector")
	for _, p := range roster["printers"].([]interface{}) {
		if p.(map[string]interface{})["id"] == "bench-repair" {
			t.Fatal("retired printer polled")
		}
	}
	_, history := printerRequest(t, app, "GET", "/history/bench-repair", nil, "")
	first := history["edits"].([]interface{})[3].(map[string]interface{})
	if first["maintenance"] != "repair" || first["location"] != "Repair bench" {
		t.Fatal("history fields missing", first)
	}
}

func TestPrinterLegacyMaintenanceMigration(t *testing.T) {
	app, db := printerTestApp(t)
	at := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	record := models.PrinterRecord{ID: "legacy-repair", Name: "Legacy", Model: "K1", Lifecycle: "repair", Condition: "out", Note: "Keep this note", Manual: true, Version: 9, UpdatedAt: at}
	if err := db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	// Reproduce the existing schema before maintenance was added.
	if err := db.Migrator().DropColumn(&models.PrinterRecord{}, "maintenance"); err != nil {
		t.Fatal(err)
	}
	if err := models.MigratePrinterRegistry(db); err != nil {
		t.Fatal(err)
	}
	var migrated models.PrinterRecord
	db.First(&migrated, "id = ?", record.ID)
	if migrated.Lifecycle != "active" || migrated.Maintenance != "repair" || migrated.Note != record.Note || migrated.Version != 9 || !migrated.UpdatedAt.Equal(at) {
		t.Fatalf("migration changed unrelated state: %#v", migrated)
	}
	body := `{"lifecycle":"testing","condition":"out","note":"Keep this note","manual":true}`
	if err := db.Create(&models.PrinterRecordEvent{PrinterID: record.ID, Version: 9, Record: body, CreatedAt: at}).Error; err != nil {
		t.Fatal(err)
	}
	_, history := printerRequest(t, app, "GET", "/history/"+record.ID, nil, "")
	item := history["edits"].([]interface{})[0].(map[string]interface{})
	if item["lifecycle"] != "active" || item["maintenance"] != "testing" {
		t.Fatal("legacy history projection", item)
	}
	var event models.PrinterRecordEvent
	db.First(&event, "printer_id = ?", record.ID)
	if event.Record != body {
		t.Fatal("rewrote immutable history")
	}
	db.Model(&migrated).UpdateColumns(map[string]interface{}{"lifecycle": "shelved", "maintenance": "diagnosis"})
	if err := models.MigratePrinterRegistry(db); err != nil {
		t.Fatal(err)
	}
	db.First(&migrated, "id = ?", record.ID)
	if migrated.Lifecycle != "shelved" || migrated.Maintenance != "diagnosis" {
		t.Fatal("repeat migration reset states")
	}
}

func TestPrinterMaintenanceValidationAndLegacySave(t *testing.T) {
	app, _ := printerTestApp(t)
	edit := editFor("Legacy testing")
	status, saved := printerRequest(t, app, "PUT", "/records/legacy-testing", edit, "")
	if status != 200 || saved["lifecycle"] != "active" || saved["maintenance"] != "testing" {
		t.Fatal("legacy save", status, saved)
	}
	bad := "printing"
	edit.Maintenance, edit.Lifecycle, edit.Version = &bad, "shelved", 1
	if status, _ := printerRequest(t, app, "PUT", "/records/legacy-testing", edit, ""); status != 400 {
		t.Fatal("invalid maintenance accepted", status)
	}
}
