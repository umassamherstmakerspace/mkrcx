package leash_backend_api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mkrcx/mkrcx/src/shared/models"
)

func TestLoadActivitySnapshotSelectsRequestedRange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activity.json")
	contents := []byte(`{
		"semester": {"timezone":"America/New_York","snapshot_at":"2026-09-03T20:15:00Z","range":{"key":"semester","label":"This semester","start":"2026-08-01","end":"2026-09-03"}},
		"30_days": {"timezone":"America/New_York","range":{"key":"30_days","label":"Past 30 days","start":"2026-08-05","end":"2026-09-03"}}
	}`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}

	response, err := loadActivitySnapshot(path, "30_days")
	if err != nil {
		t.Fatal(err)
	}
	if response.Range.Key != "30_days" {
		t.Fatalf("range key = %q, want 30_days", response.Range.Key)
	}

	response, err = loadActivitySnapshot(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if response.Range.Key != "semester" || response.SnapshotAt == "" {
		t.Fatalf("unexpected default snapshot: %+v", response)
	}
}

func TestActivityPresetUsesEasternAcademicWindows(t *testing.T) {
	location, err := time.LoadLocation(activityTimezone)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		requested string
		now       time.Time
		wantStart string
		wantEnd   string
	}{
		{"fall semester", "semester", time.Date(2026, time.September, 3, 15, 0, 0, 0, location), "2026-08-01", "2026-09-04"},
		{"spring semester", "semester", time.Date(2027, time.February, 4, 15, 0, 0, 0, location), "2027-01-01", "2027-02-05"},
		{"summer", "semester", time.Date(2027, time.July, 4, 15, 0, 0, 0, location), "2027-06-01", "2027-07-05"},
		{"academic year", "academic_year", time.Date(2027, time.February, 4, 15, 0, 0, 0, location), "2026-08-01", "2027-02-05"},
		{"past 30 days", "30_days", time.Date(2026, time.September, 3, 15, 0, 0, 0, location), "2026-08-05", "2026-09-04"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, start, end := activityPreset(test.requested, test.now, location)
			if start.Format("2006-01-02") != test.wantStart || end.Format("2006-01-02") != test.wantEnd {
				t.Fatalf("range = %s through %s, want %s through %s", start.Format("2006-01-02"), end.Format("2006-01-02"), test.wantStart, test.wantEnd)
			}
		})
	}
}

func TestBuildActivityResponseDeduplicatesVisitorsAndKeepsAllCheckinsInHeatmap(t *testing.T) {
	db := newCheckinExportTestDB(t)
	location, err := time.LoadLocation(activityTimezone)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 3, 14, 0, 0, 0, location)

	events := []models.CheckinEvent{
		{OccurredAt: time.Date(2026, time.August, 3, 10, 0, 0, 0, location).UTC(), MemberUUID: "member-c", IdempotencyScope: "test", IdempotencyKey: "1"},
		{OccurredAt: time.Date(2026, time.September, 2, 10, 0, 0, 0, location).UTC(), MemberUUID: "member-a", IdempotencyScope: "test", IdempotencyKey: "2"},
		{OccurredAt: time.Date(2026, time.September, 2, 10, 15, 0, 0, location).UTC(), MemberUUID: "member-a", IdempotencyScope: "test", IdempotencyKey: "3"},
		{OccurredAt: time.Date(2026, time.September, 2, 10, 30, 0, 0, location).UTC(), MemberUUID: "member-b", IdempotencyScope: "test", IdempotencyKey: "4"},
		{OccurredAt: time.Date(2026, time.September, 2, 10, 45, 0, 0, location).UTC(), IdempotencyScope: "test", IdempotencyKey: "5"},
		{OccurredAt: time.Date(2026, time.September, 3, 9, 0, 0, 0, location).UTC(), MemberUUID: "member-a", IdempotencyScope: "test", IdempotencyKey: "6"},
	}
	if err := db.Create(&events).Error; err != nil {
		t.Fatal(err)
	}
	accounts := []models.User{
		{Model: models.Model{CreatedAt: time.Date(2026, time.August, 15, 12, 0, 0, 0, location).UTC()}, Email: "member@example.com", Role: "member"},
		{Model: models.Model{CreatedAt: time.Date(2026, time.September, 3, 12, 0, 0, 0, location).UTC()}, Email: "today@example.com", Role: "member"},
		{Model: models.Model{CreatedAt: time.Date(2026, time.September, 3, 12, 0, 0, 0, location).UTC()}, Email: "service@example.com", Role: "service"},
	}
	if err := db.Create(&accounts).Error; err != nil {
		t.Fatal(err)
	}

	response, err := BuildActivityResponse(db, "semester", now, location)
	if err != nil {
		t.Fatal(err)
	}
	if response.Today.Visitors != 1 || response.Today.Checkins != 1 || response.Today.NewAccounts != 1 {
		t.Fatalf("unexpected today summary: %+v", response.Today)
	}
	if response.Week.Visitors != 2 || response.Week.Checkins != 5 || response.Week.NewAccounts != 1 {
		t.Fatalf("unexpected week summary: %+v", response.Week)
	}
	if response.Selected.Visitors != 3 || response.Selected.Checkins != 6 || response.Selected.NewAccounts != 2 {
		t.Fatalf("unexpected selected summary: %+v", response.Selected)
	}
	if response.Coverage.IdentifiedCheckins != 5 || response.Coverage.TotalCheckins != 6 {
		t.Fatalf("unexpected coverage: %+v", response.Coverage)
	}
	var wednesdayTen int
	for _, cell := range response.Heatmap {
		if cell.Weekday == int(time.Wednesday) && cell.Hour == 10 {
			wednesdayTen = cell.Checkins
		}
	}
	if wednesdayTen != 4 {
		t.Fatalf("Wednesday 10am heat cell = %d, want 4", wednesdayTen)
	}
}
