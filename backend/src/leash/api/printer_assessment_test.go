package leash_backend_api

import (
	"github.com/mkrcx/mkrcx/src/shared/models"
	"testing"
	"time"
)

func TestPrinterAssessmentSurvivesTestPrintAndNextActionIsPrivate(t *testing.T) {
	app, _ := printerTestApp(t)
	action := "Verify cable repair"
	edit := editFor("Bench fixture")
	edit.Manual = false // Older clients cannot turn an assessment back into printer permission.
	edit.NextAction = &action
	status, saved := printerRequest(t, app, "PUT", "/records/assessment", edit, "")
	if status != 200 || saved["manual"] != true {
		t.Fatal(status, saved)
	}
	reading := printerReading{ID: "assessment", Condition: "working", Activity: "printing", Note: "Test enabled"}
	if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", printerSnapshot{FetchedAt: time.Now().UTC(), Printers: []printerReading{reading}}, "test-collector"); status != 204 {
		t.Fatal(status)
	}
	_, fleet := printerRequest(t, app, "GET", "/staff", nil, "")
	p := printerItem(t, fleet, "assessment")
	if p["condition"] != "out" || p["activity"] != "printing" || p["nextAction"] != action || p["note"] != edit.Note {
		t.Fatal(p)
	}
	_, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	if _, ok := printerItem(t, public, "assessment")["nextAction"]; ok {
		t.Fatal("staff action leaked publicly")
	}
	edit.Version = 1
	edit.NextAction = nil
	_, saved = printerRequest(t, app, "PUT", "/records/assessment", edit, "")
	if saved["nextAction"] != action {
		t.Fatal("older client cleared next action")
	}
	empty := ""
	edit.Version = 2
	edit.NextAction = &empty
	_, saved = printerRequest(t, app, "PUT", "/records/assessment", edit, "")
	if saved["nextAction"] != "" {
		t.Fatal("could not clear next action")
	}
}

func TestPrinterAssessmentBaselinePreservesVisibleValuesAndDoesNotReset(t *testing.T) {
	_, db := printerTestApp(t)
	at := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	legacy := models.PrinterRecord{ID: "assessment-legacy", Name: "Legacy", Model: "K1", Lifecycle: "shelved", Condition: "unknown", Manual: false, Version: 8, ObservedCondition: "out", ObservedNote: "Cable repair pending", ConditionObservedAt: &at}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := models.MigratePrinterRegistry(db); err != nil {
		t.Fatal(err)
	}
	var record models.PrinterRecord
	db.First(&record, "id = ?", legacy.ID)
	if !record.Manual || record.Condition != "out" || record.Note != legacy.ObservedNote || record.Version != 8 || !record.UpdatedAt.Equal(at) {
		t.Fatal(record)
	}
	db.Model(&record).UpdateColumns(map[string]interface{}{"observed_condition": "working", "observed_note": "Testing"})
	if err := models.MigratePrinterRegistry(db); err != nil {
		t.Fatal(err)
	}
	db.First(&record, "id = ?", legacy.ID)
	if record.Condition != "out" || record.Note != legacy.ObservedNote {
		t.Fatal("baseline reran", record)
	}
}

func TestPrinterHistoryChangeEnrichmentPreservesFalseAndRejectsCollisions(t *testing.T) {
	app, db := printerTestApp(t)
	printerRequest(t, app, "PUT", "/records/deltas", editFor("Deltas"), "")
	now := time.Now().UTC()
	yes, no := true, false
	event := models.PrinterHistoryEvent{SourceID: "station:condition:1", PrinterID: "deltas", RecordedAt: now.Add(-time.Hour), EventType: "staff_runtime_changed", Detail: "Condition: out_of_service\nNote: Fan failed"}
	snapshot := printerSnapshot{FetchedAt: now, Printers: []printerReading{{ID: "deltas", Condition: "working", Activity: "idle"}}, History: []models.PrinterHistoryEvent{event}}
	send := func() {
		t.Helper()
		if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 204 {
			t.Fatal(status)
		}
	}
	send()
	snapshot.History[0].NoteChanged = &yes
	snapshot.History[0].ConditionChanged = &no
	snapshot.History[0].PreviousCondition = "out_of_service"
	send()
	send()
	var stored models.PrinterHistoryEvent
	db.First(&stored, "source_id = ?", event.SourceID)
	if stored.NoteChanged == nil || !*stored.NoteChanged || stored.ConditionChanged == nil || *stored.ConditionChanged {
		t.Fatal("lost comparison", stored)
	}
	snapshot.History[0].ConditionChanged = &yes
	snapshot.History[0].PreviousCondition = "available"
	send()
	db.First(&stored, "source_id = ?", event.SourceID)
	if *stored.ConditionChanged || stored.PreviousCondition != "out_of_service" {
		t.Fatal("replaced comparison", stored)
	}
	snapshot.History[0] = event
	snapshot.History[0].SourceID = "station:condition:2"
	send()
	snapshot.History[0].RecordedAt = now.Add(-2 * time.Hour)
	snapshot.History[0].NoteChanged = &yes
	send()
	var collision models.PrinterHistoryEvent
	db.First(&collision, "source_id = ?", "station:condition:2")
	if collision.NoteChanged != nil {
		t.Fatal("enriched mismatched source event")
	}
	snapshot.History[0].EventType = "printer_completed"
	if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 400 {
		t.Fatal("accepted changes on job", status)
	}
}
