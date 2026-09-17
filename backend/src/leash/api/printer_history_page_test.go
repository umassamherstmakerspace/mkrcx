package leash_backend_api

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/mkrcx/mkrcx/src/shared/models"
)

func TestPrinterHistoryCursorTraversesTiesAndFiltersBeforePaging(t *testing.T) {
	app, db := printerTestApp(t)
	record := models.PrinterRecord{ID: "pages", Name: "Pagination fixture", Lifecycle: "active", Manual: true}
	if err := db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	at := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 125; i++ {
		kind := "submission"
		if i%10 == 0 {
			kind = "service"
		}
		if err := db.Create(&models.PrinterHistoricalEntry{SourceID: fmt.Sprintf("legacy:%03d", i), PrinterID: record.ID, RecordedAt: at, Kind: kind, Body: "Evidence", Sources: []models.PrinterSource{{URL: "https://private.example/source", Label: "Original"}}, ImportID: "private-import"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	seconds := 90.0
	for i := 0; i < 22; i++ {
		db.Create(&models.PrinterHistoryEvent{SourceID: fmt.Sprintf("station:job:%03d", i), PrinterID: record.ID, RecordedAt: at, EventType: "printer_completed", DurationSeconds: &seconds})
	}
	db.Create(&models.PrinterSummary{SourceID: "standup:old", PrinterID: record.ID, ReportDate: "2024-01-01", Body: "Report"})
	for _, filter := range []string{"all", "prints", "updates"} {
		seen := map[string]bool{}
		cursor := ""
		pages := 0
		for {
			status, body := printerRequest(t, app, "GET", "/history/pages?page=1&filter="+filter+"&cursor="+url.QueryEscape(cursor), nil, "")
			if status != 200 {
				t.Fatalf("%s: %d %v", filter, status, body)
			}
			ids := body["pageIds"].([]interface{})
			if len(ids) > 20 {
				t.Fatal("unbounded page")
			}
			for _, id := range ids {
				key := id.(string)
				if seen[key] {
					t.Fatalf("duplicate %s", key)
				}
				seen[key] = true
			}
			for _, h := range body["historical"].([]interface{}) {
				entry := h.(map[string]interface{})
				if entry["sources"] != nil || entry["importId"] != nil {
					t.Fatal("private provenance exposed")
				}
				if filter == "updates" && entry["kind"] == "submission" {
					t.Fatal("filter applied after paging")
				}
			}
			if body["usage"].(map[string]interface{})["jobs"] != float64(22) || body["usage"].(map[string]interface{})["seconds"] != 1980.0 {
				t.Fatal("legacy evidence contaminated measured hours")
			}
			if body["legacy"].(map[string]interface{})["jobs"] != float64(112) {
				t.Fatal("legacy total depends on page")
			}
			cursor = body["nextCursor"].(string)
			pages++
			if cursor == "" {
				break
			}
			if pages > 12 {
				t.Fatal("cursor loop")
			}
		}
		want := 148
		if filter == "prints" {
			want = 134
		}
		if filter == "updates" {
			want = 14
		}
		if len(seen) != want {
			t.Fatalf("%s got %d want %d", filter, len(seen), want)
		}
	}
}

func TestPrinterHistoryCursorValidationAndRetiredPrivacy(t *testing.T) {
	app, db := printerTestApp(t)
	db.Create(&models.PrinterRecord{ID: "retired-fixture", Name: "Retired", Lifecycle: "retired", Manual: true, Host: "192.168.1.10"})
	for _, query := range []string{"filter=invalid", "cursor=not-json", "cursor=e30"} {
		status, _ := printerRequest(t, app, "GET", "/history/retired-fixture?page=1&"+query, nil, "")
		if status != 400 {
			t.Fatal(status)
		}
	}
	status, public := printerRequest(t, app, "GET", "/printer-fleet", nil, "")
	if status != 200 {
		t.Fatal("retired record became public", status)
	}
	for _, item := range public["printers"].([]interface{}) {
		if item.(map[string]interface{})["id"] == "retired-fixture" {
			t.Fatal("retired record became public")
		}
	}
	_, roster := printerRequest(t, app, "GET", "/printer-fleet/roster", nil, "test-collector")
	for _, item := range roster["printers"].([]interface{}) {
		if item.(map[string]interface{})["id"] == "retired-fixture" {
			t.Fatal("retired printer added to polling")
		}
	}
}
