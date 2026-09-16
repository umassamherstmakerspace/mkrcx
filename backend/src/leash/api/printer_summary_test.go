package leash_backend_api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mkrcx/mkrcx/src/shared/models"
)

func TestPrinterSummariesAreSourceLinkedIdempotentAndPrivate(t *testing.T) {
	app, db := printerTestApp(t)
	app.Post("/summary/:id", importPrinterSummary)
	printerRequest(t, app, "PUT", "/records/summary-fixture", editFor("Summary fixture"), "")
	input := printerSummaryInput{SourceID: "standup:summary-fixture:report-1", ReportDate: "2026-01-15", Body: "Sam replaced a connector; print verification remains open.", Sources: []models.PrinterSource{{URL: "https://example.org/message/1", Label: "Sam’s report"}}}
	for i := 0; i < 2; i++ {
		if status, _ := printerRequest(t, app, "POST", "/summary/summary-fixture", input, ""); status != 200 {
			t.Fatal(status)
		}
	}
	var count int64
	db.Model(&models.PrinterSummary{}).Count(&count)
	if count != 1 {
		t.Fatal("repeated import duplicated summary", count)
	}
	var record models.PrinterRecord
	db.First(&record, "id = ?", "summary-fixture")
	if record.Version != 1 || record.Condition != "out" || record.Note != "Awaiting thermistor" {
		t.Fatal("report changed current assessment", record)
	}
	_, history := printerRequest(t, app, "GET", "/history/summary-fixture", nil, "")
	entry := history["summaries"].([]interface{})[0].(map[string]interface{})
	if entry["preparedBy"] != "Staff fixture" || entry["reportDate"] != input.ReportDate || entry["importedAt"] == nil {
		t.Fatal(entry)
	}
	_, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	body, _ := json.Marshal(public)
	if strings.Contains(string(body), input.Body) || strings.Contains(string(body), "example.org") {
		t.Fatal("private summary exposed")
	}
	input.Body = "Changed report"
	if status, _ := printerRequest(t, app, "POST", "/summary/summary-fixture", input, ""); status != 409 {
		t.Fatal("source silently overwritten", status)
	}
	input.SourceID += ":bad"
	input.Sources[0].URL = "javascript:alert(1)"
	if status, _ := printerRequest(t, app, "POST", "/summary/summary-fixture", input, ""); status != 400 {
		t.Fatal("unsafe source accepted", status)
	}
}

func TestPrinterSummaryLinksAreOptionalAndEquivalentWhenEmpty(t *testing.T) {
	app, db := printerTestApp(t)
	app.Post("/summary/:id", importPrinterSummary)
	printerRequest(t, app, "PUT", "/records/composite", editFor("Composite"), "")
	input := map[string]interface{}{"sourceId": "standup:composite:review-1", "reportDate": "2026-01-15", "body": "Several reports describe intermittent heating; verification remains open."}
	for _, links := range []interface{}{nil, []models.PrinterSource{}} {
		if links != nil {
			input["sources"] = links
		}
		if status, _ := printerRequest(t, app, "POST", "/summary/composite", input, ""); status != 200 {
			t.Fatal("link-free summary or equivalent retry rejected", status)
		}
	}
	var count int64
	db.Model(&models.PrinterSummary{}).Count(&count)
	if count != 1 {
		t.Fatal("link-free retry duplicated summary", count)
	}
	_, history := printerRequest(t, app, "GET", "/history/composite", nil, "")
	entry := history["summaries"].([]interface{})[0].(map[string]interface{})
	if entry["body"] != input["body"] || entry["preparedBy"] != "Staff fixture" {
		t.Fatal("link-free summary lost text or attribution", entry)
	}
}

func TestRecordedUsageCountsOutcomesAndMissingDurations(t *testing.T) {
	_, db := printerTestApp(t)
	now := time.Now().UTC()
	duration := 3600.0
	for i, kind := range []string{"printer_completed", "printer_cancelled", "printer_failed", "started"} {
		event := models.PrinterHistoryEvent{SourceID: kind, PrinterID: "usage-fixture", RecordedAt: now.Add(time.Duration(i) * time.Minute), EventType: kind, DurationSeconds: &duration}
		if kind == "printer_failed" {
			event.DurationSeconds = nil
		}
		if err := db.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
	}
	usage, err := readPrinterUsage(db, "usage-fixture")
	if err != nil || usage.Seconds != 7200 || usage.Jobs != 3 || usage.MissingDurations != 1 || usage.FirstOutcome == nil {
		t.Fatal(usage, err)
	}
	empty, err := readPrinterUsage(db, "empty")
	if err != nil || empty.Jobs != 0 || empty.FirstOutcome != nil {
		t.Fatal(empty, err)
	}
}

func TestNewPrinterNoteVisibleWithoutReplacingAssessment(t *testing.T) {
	app, db := printerTestApp(t)
	printerRequest(t, app, "PUT", "/records/note-fixture", editFor("Note fixture"), "")
	db.Model(&models.PrinterRecord{}).Where("id = ?", "note-fixture").UpdateColumn("updated_at", time.Now().UTC().Add(-time.Hour))
	snapshot := printerSnapshot{FetchedAt: time.Now().UTC(), Printers: []printerReading{{ID: "note-fixture", Condition: "working", Activity: "printing", Note: "Connector replaced; testing now."}}}
	if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 204 {
		t.Fatal(status)
	}
	_, staff := printerRequest(t, app, "GET", "/staff", nil, "")
	p := printerItem(t, staff, "note-fixture")
	if p["condition"] != "out" || p["note"] != "Awaiting thermistor" || p["printerNote"] != "Connector replaced; testing now." || p["printerNoteAt"] == nil {
		t.Fatal(p)
	}
	_, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	if printerItem(t, public, "note-fixture")["printerNote"] != nil {
		t.Fatal("staff update exposed publicly")
	}
}

func TestPrinterErrorsSurviveDisconnectWithoutDuplicateOrFalseRecovery(t *testing.T) {
	app, db := printerTestApp(t)
	printerRequest(t, app, "PUT", "/records/error-fixture", editFor("Error fixture"), "")
	now := time.Now().UTC()
	readings := []printerReading{
		{ID: "error-fixture", Condition: "out", Activity: "unknown", Fault: "Heater not heating"},
		{ID: "error-fixture", Condition: "unknown", Activity: "unknown"},
		{ID: "error-fixture", Condition: "out", Activity: "unknown", Fault: "Heater not heating"},
		{ID: "error-fixture", Condition: "out", Activity: "idle"},
	}
	for i, reading := range readings {
		if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", printerSnapshot{FetchedAt: now.Add(time.Duration(i) * time.Millisecond), Printers: []printerReading{reading}}, "test-collector"); status != 204 {
			t.Fatal(status)
		}
		if i == 0 {
			_, staff := printerRequest(t, app, "GET", "/staff", nil, "")
			if printerItem(t, staff, "error-fixture")["fault"] != reading.Fault {
				t.Fatal("current error missing")
			}
			_, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
			if printerItem(t, public, "error-fixture")["fault"] != nil {
				t.Fatal("error leaked publicly")
			}
		}
	}
	var events []models.PrinterHistoryEvent
	db.Where("printer_id = ?", "error-fixture").Order("recorded_at ASC").Find(&events)
	if len(events) != 2 || events[0].EventType != "printer_error" || events[1].EventType != "printer_responding" {
		t.Fatal(events)
	}
	var record models.PrinterRecord
	db.First(&record, "id = ?", "error-fixture")
	if record.Condition != "out" {
		t.Fatal("reconnection cleared broken assessment")
	}
}
