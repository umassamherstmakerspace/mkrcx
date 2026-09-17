package leash_backend_api

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/mkrcx/mkrcx/src/shared/models"
)

func TestPrinterEditNamesPreserveSnapshotAndResolveLegacyAccounts(t *testing.T) {
	app, db := printerTestApp(t)
	if status, _ := printerRequest(t, app, "PUT", "/records/attribution", editFor("Attribution"), ""); status != 200 {
		t.Fatal(status)
	}
	var saved models.PrinterRecordEvent
	db.First(&saved, "printer_id = ?", "attribution")
	if saved.ActorName != "Staff fixture" {
		t.Fatal("missing name snapshot", saved)
	}
	db.Exec("INSERT INTO users (id, name) VALUES (42, 'Renamed user'), (7, 'Legacy staff')")
	legacy := saved
	legacy.ID = 0
	legacy.Version = 2
	legacy.Actor = "user:7"
	legacy.ActorName = ""
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	missing := legacy
	missing.ID = 0
	missing.Version = 3
	missing.Actor = "user:99"
	if err := db.Create(&missing).Error; err != nil {
		t.Fatal(err)
	}
	status, history := printerRequest(t, app, "GET", "/history/attribution", nil, "")
	if status != 200 {
		t.Fatal(status)
	}
	edits := history["edits"].([]interface{})
	if edits[0].(map[string]interface{})["actorName"] != "" || edits[1].(map[string]interface{})["actorName"] != "Legacy staff" || edits[2].(map[string]interface{})["actorName"] != "Staff fixture" {
		t.Fatal(edits)
	}
	// The lookup must not rewrite old immutable events.
	db.First(&legacy, "id = ?", legacy.ID)
	if legacy.ActorName != "" {
		t.Fatal("rewrote legacy event")
	}
}

func TestPrinterStationAttributionBackfillPrivacyAndCollisions(t *testing.T) {
	app, _ := printerTestApp(t)
	printerRequest(t, app, "PUT", "/records/attribution", editFor("Attribution"), "")
	now := time.Now().UTC()
	event := models.PrinterHistoryEvent{SourceID: "station:condition:1", PrinterID: "attribution", RecordedAt: now.Add(-time.Hour), EventType: "staff_runtime_changed", Detail: "Condition: working"}
	snapshot := printerSnapshot{FetchedAt: now, Printers: []printerReading{{ID: "attribution", Condition: "working", Activity: "idle"}}, History: []models.PrinterHistoryEvent{event}}
	send := func(status int) {
		t.Helper()
		if got, _ := printerRequest(t, app, "POST", "/printer-fleet/ingest", snapshot, "test-collector"); got != status {
			t.Fatal(got, status)
		}
	}
	read := func() map[string]interface{} {
		t.Helper()
		status, history := printerRequest(t, app, "GET", "/history/attribution", nil, "")
		if status != 200 {
			t.Fatal(status)
		}
		return history["events"].([]interface{})[0].(map[string]interface{})
	}
	send(204)
	snapshot.History[0].ActorMethod = "local_pin"
	snapshot.History[0].RecordedAt = event.RecordedAt.Add(-time.Hour)
	send(204)
	if read()["actorMethod"] != nil {
		t.Fatal("enriched mismatched timestamp")
	}
	snapshot.History[0].RecordedAt = event.RecordedAt
	send(204)
	if read()["actorMethod"] != "local_pin" {
		t.Fatal("missing PIN attribution")
	}
	snapshot.History[0].ActorMethod = "ucard"
	snapshot.History[0].ActorName = "Private staff"
	send(204)
	if got := read(); got["actorMethod"] != "local_pin" || got["actorName"] != nil {
		t.Fatal("relabeled PIN edit", got)
	}
	snapshot.History[0].SourceID = "station:condition:2"
	snapshot.History[0].RecordedAt = now.Add(-time.Minute)
	send(204)
	if got := read(); got["actorName"] != "Private staff" || got["actorMethod"] != "ucard" {
		t.Fatal(got)
	}
	_, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	body, _ := json.Marshal(public)
	for _, private := range []string{"actorName", "actorMethod", "Private staff", "Staff fixture"} {
		if bytes.Contains(body, []byte(private)) {
			t.Fatal("public attribution leak", private)
		}
	}
	snapshot.History[0].ActorMethod = "local_pin"
	send(400) // A PIN does not identify a person.
	snapshot.History[0].ActorName = ""
	snapshot.History[0].ActorMethod = "secret-token"
	send(400)
}
