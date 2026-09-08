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
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("db", db)
		c.Locals("auth", leash_auth.Authentication{Authenticator: leash_auth.AUTHENTICATOR_USER, User: models.User{ID: 42}})
		return c.Next()
	})
	registerPrinterPublicEndpoints(app)
	app.Put("/records/:id", savePrinterRecord)
	app.Get("/staff", func(c *fiber.Ctx) error { return respondPrinterFleet(c, true) })
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
	app, _ := printerTestApp(t)
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
