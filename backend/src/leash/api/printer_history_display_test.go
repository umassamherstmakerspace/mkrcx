package leash_backend_api

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mkrcx/mkrcx/src/shared/models"
)

func TestPrinterMeterEstimateResetsCorrectionsAndDuplicates(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []float64
		want   float64
	}{
		{"steady", []float64{100, 200, 300}, 300},
		{"small correction", []float64{100, 260, 250}, 250},
		{"counter reset", []float64{1000, 1360, 96, 157, 242}, 1602},
		{"older counter era", []float64{1806, 770, 1144, 2794}, 4600},
		{"isolated low reading", []float64{1000, 20, 1100}, 1100},
		{"isolated high reading", []float64{100, 1000, 200}, 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			var rows []models.PrinterHistoricalEntry
			for i, value := range test.values {
				value := value
				entry := models.PrinterHistoricalEntry{RecordedAt: time.Date(2024, 1, i+1, 12, 0, 0, 0, time.UTC), DateOnly: true, MeterHours: &value}
				rows = append(rows, entry, entry)
			}
			hours, through, ok := printerMeterEstimate(rows)
			if !ok || hours != test.want || through.Hour() != 0 || through.Day() != len(test.values)+1 {
				t.Fatalf("%v %v %v", hours, through, ok)
			}
		})
	}
}

func TestPrinterHistoryCompactEstimateAndNamesPreserveEvidence(t *testing.T) {
	app, db := printerTestApp(t)
	db.Create(&models.PrinterRecord{ID: "display", Name: "Display fixture", Lifecycle: "active", Manual: true})
	db.Exec("INSERT INTO users(id,name,email) VALUES (9, 'Account Person', 'account@umass.edu')")
	db.Create(&models.PrinterIdentityAlias{Alias: "reviewed", Name: "Reviewed Person", Source: "Confirmed by staff"})
	at := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	meter, seconds := 100.0, 3600.0
	db.Create(&models.PrinterHistoricalEntry{SourceID: "meter", PrinterID: "display", RecordedAt: at, DateOnly: true, Kind: "meter", MeterHours: &meter})
	db.Create(&models.PrinterHistoricalEntry{SourceID: "origin", PrinterID: "display", RecordedAt: at.AddDate(-1, 0, 0), Kind: "origin", Body: "Recorded start date: Jan 01, 2023 (service log)."})
	for i, who := range []string{"account", "reviewed", "unknown", "Account Person"} {
		db.Create(&models.PrinterHistoricalEntry{SourceID: fmt.Sprint("old", i), PrinterID: "display", RecordedAt: at, Kind: "submission", Reporter: who})
	}
	for i, when := range []time.Time{at.Add(-time.Hour), at.Add(48 * time.Hour)} {
		db.Create(&models.PrinterHistoryEvent{SourceID: fmt.Sprint("job", i), PrinterID: "display", RecordedAt: when, EventType: "printer_completed", DurationSeconds: &seconds, Person: "reviewed", ActorName: "account", ActorMethod: "ucard"})
	}
	status, body := printerRequest(t, app, "GET", "/history/display?page=1&filter=all", nil, "")
	if status != 200 {
		t.Fatal(status, body)
	}
	estimate := body["estimate"].(map[string]interface{})
	if estimate["hours"] != 101.0 || estimate["jobs"] != 6.0 || estimate["since"] != "2023-01-01" {
		t.Fatal(estimate)
	}
	for _, raw := range body["events"].([]interface{}) {
		event := raw.(map[string]interface{})
		if event["actorName"] != "Account Person" || event["person"] != "Reviewed Person" {
			t.Fatal(event)
		}
	}
	want := map[string]string{"old0": "Account Person", "old1": "Reviewed Person", "old2": "unknown", "old3": "Account Person"}
	for _, raw := range body["historical"].([]interface{}) {
		entry := raw.(map[string]interface{})
		if name, ok := want[entry["sourceId"].(string)]; ok && entry["reporter"] != name {
			t.Fatal(entry)
		}
	}
	var saved models.PrinterHistoryEvent
	db.First(&saved, "source_id = ?", "job0")
	if saved.ActorName != "account" || saved.Person != "reviewed" {
		t.Fatal("source evidence rewritten")
	}
	var original models.PrinterHistoricalEntry
	db.First(&original, "source_id = ?", "old1")
	if original.Reporter != "reviewed" {
		t.Fatal("source reporter rewritten")
	}
	_, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	if strings.Contains(fmt.Sprint(public), "Reviewed Person") || strings.Contains(fmt.Sprint(public), "Account Person") {
		t.Fatal("staff names exposed publicly")
	}
}

func TestPrinterHistoryEstimateKeepsApproximateOriginAndUnknownHours(t *testing.T) {
	_, db := printerTestApp(t)
	origin := models.PrinterHistoricalEntry{RecordedAt: time.Now(), Body: "In service since fall 2023 (staff-confirmed; exact date unknown)."}
	estimate, err := readPrinterHistoryEstimate(db, "empty", printerUsage{}, 4, nil, nil, []models.PrinterHistoricalEntry{origin})
	if err != nil || estimate.Hours != nil || estimate.Since != "fall 2023" || estimate.Jobs != 4 {
		t.Fatal(estimate, err)
	}
}
