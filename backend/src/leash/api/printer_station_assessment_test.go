package leash_backend_api

import (
	"testing"
	"time"

	"github.com/mkrcx/mkrcx/src/shared/models"
)

func TestStaffPrinterChangesUpdateAssessmentWithoutReplayingOldFields(t *testing.T) {
	app, db := printerTestApp(t)
	edit := editFor("Sync fixture")
	edit.Lifecycle = "shelved"
	printerRequest(t, app, "PUT", "/records/sync", edit, "")
	now := time.Now().UTC()
	base := now.Add(-time.Hour)
	db.Model(&models.PrinterRecord{}).Where("id = ?", "sync").UpdateColumns(map[string]interface{}{"updated_at": base, "condition_set_at": base, "note_set_at": base})
	yes, no := true, false
	note := models.PrinterHistoryEvent{SourceID: "station:condition:sync-note", PrinterID: "sync", EventType: "staff_runtime_changed", RecordedAt: now.Add(-time.Minute), Detail: "Condition: out_of_service\nNote: Replaced connector\nTesting next", NoteChanged: &yes, ConditionChanged: &no}
	condition := models.PrinterHistoryEvent{SourceID: "station:condition:sync-condition", PrinterID: "sync", EventType: "staff_runtime_changed", RecordedAt: now.Add(-2 * time.Minute), Detail: "Condition: needs_attention\nNote: Old note", NoteChanged: &no, ConditionChanged: &yes}
	send := func(events ...models.PrinterHistoryEvent) {
		t.Helper()
		snapshot := printerSnapshot{FetchedAt: time.Now().UTC(), Printers: []printerReading{{ID: "sync", Condition: "working", Activity: "printing"}}, History: events}
		if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 204 {
			t.Fatal(status)
		}
	}
	send(note, condition) // Collector order is newest first; each field must still advance.
	var saved models.PrinterRecord
	db.First(&saved, "id = ?", "sync")
	if saved.Condition != "limited" || saved.Note != "Replaced connector\nTesting next" || saved.Version != 3 || saved.Lifecycle != "shelved" {
		t.Fatal(saved)
	}
	send(condition, note)
	db.First(&saved, "id = ?", "sync")
	if saved.Version != 3 {
		t.Fatal("duplicate collection changed assessment", saved)
	}
	edit.Version = saved.Version
	edit.Condition = "out"
	edit.Note = "Repair review supersedes printer report"
	if status, _ := printerRequest(t, app, "PUT", "/records/sync", edit, ""); status != 200 {
		t.Fatal(status)
	}
	send(condition, note)
	db.First(&saved, "id = ?", "sync")
	if saved.Version != 4 || saved.Condition != "out" || saved.Note != edit.Note {
		t.Fatal("old printer event replaced reviewed assessment", saved)
	}
	// Normal optimistic concurrency still protects an editor with the pre-sync version.
	edit.Version = 1
	if status, _ := printerRequest(t, app, "PUT", "/records/sync", edit, ""); status != 409 {
		t.Fatal("stale edit accepted", status)
	}
}

func TestPrinterAssessmentIgnoresControlAndMachineEventsAndAcceptsNoteClear(t *testing.T) {
	app, db := printerTestApp(t)
	printerRequest(t, app, "PUT", "/records/controls", editFor("Controls"), "")
	now := time.Now().UTC()
	base := now.Add(-time.Hour)
	db.Model(&models.PrinterRecord{}).Where("id = ?", "controls").UpdateColumns(map[string]interface{}{"updated_at": base, "condition_set_at": base, "note_set_at": base})
	yes, no := true, false
	events := []models.PrinterHistoryEvent{
		{SourceID: "station:condition:machine", PrinterID: "controls", EventType: "system_runtime_changed", RecordedAt: now, Detail: "Condition: available\nNote:", ConditionChanged: &yes, NoteChanged: &yes},
		{SourceID: "station:condition:override", PrinterID: "controls", EventType: "staff_runtime_changed", RecordedAt: now, Detail: "Condition: available\nNote:", ConditionChanged: &no, NoteChanged: &no},
		{SourceID: "station:condition:unknown", PrinterID: "controls", EventType: "staff_runtime_changed", RecordedAt: now, Detail: "Condition: available\nNote:"},
	}
	snapshot := printerSnapshot{FetchedAt: now, Printers: []printerReading{{ID: "controls", Condition: "working", Activity: "printing"}}, History: events}
	if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 204 {
		t.Fatal(status)
	}
	var saved models.PrinterRecord
	db.First(&saved, "id = ?", "controls")
	if saved.Version != 1 || saved.Condition != "out" {
		t.Fatal("test permission changed assessment", saved)
	}
	snapshot.History = []models.PrinterHistoryEvent{{SourceID: "station:condition:clear", PrinterID: "controls", EventType: "staff_runtime_changed", RecordedAt: now, Detail: "Condition: out_of_service\nNote:", ConditionChanged: &no, NoteChanged: &yes}}
	if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 204 {
		t.Fatal(status)
	}
	db.First(&saved, "id = ?", "controls")
	if saved.Note != "" || saved.Condition != "out" || saved.Version != 2 {
		t.Fatal("note clear lost", saved)
	}
}

func TestHiddenPrinterTestHistoryRemainsStoredAndExcludedOnRetry(t *testing.T) {
	app, db := printerTestApp(t)
	app.Get("/paged/:id", printerStaffHistoryPage)
	printerRequest(t, app, "PUT", "/records/hidden", editFor("Hidden"), "")
	yes := true
	event := models.PrinterHistoryEvent{SourceID: "station:condition:hidden", PrinterID: "hidden", EventType: "staff_runtime_changed", RecordedAt: time.Now().UTC(), Detail: "Condition: available\nNote: keyboard test", NoteChanged: &yes, ConditionChanged: &yes, HiddenReason: "Confirmed interface test"}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	event.HiddenReason = ""
	snapshot := printerSnapshot{FetchedAt: time.Now().UTC(), Printers: []printerReading{{ID: "hidden", Condition: "working", Activity: "idle"}}, History: []models.PrinterHistoryEvent{event}}
	if status, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); status != 204 {
		t.Fatal(status)
	}
	for _, path := range []string{"/history/hidden", "/paged/hidden?filter=all", "/paged/hidden?filter=updates"} {
		status, history := printerRequest(t, app, "GET", path, nil, "")
		if status != 200 {
			t.Fatal(status)
		}
		if events, ok := history["events"].([]interface{}); ok && len(events) != 0 {
			t.Fatal("hidden event returned", path)
		}
	}
	var stored models.PrinterHistoryEvent
	db.First(&stored, "source_id = ?", event.SourceID)
	if stored.HiddenReason == "" || stored.Detail != event.Detail {
		t.Fatal("original source or exclusion lost")
	}
	var record models.PrinterRecord
	db.First(&record, "id = ?", "hidden")
	if record.Version != 1 || record.Condition != "out" {
		t.Fatal("hidden test changed assessment", record)
	}
}
