package leash_backend_api

import (
	"fmt"
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

func TestBuildActivityResponseSeparatesLinkedAndUnlinkedMembers(t *testing.T) {
	db := newCheckinExportTestDB(t)
	if err := db.AutoMigrate(&models.UserUpdate{}); err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation(activityTimezone)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 3, 14, 0, 0, 0, location)

	events := []models.CheckinEvent{
		{OccurredAt: time.Date(2026, time.August, 3, 10, 0, 0, 0, location).UTC(), MemberUUID: "member-c", LinkedAtTap: true, IdempotencyScope: "test", IdempotencyKey: "1"},
		{OccurredAt: time.Date(2026, time.September, 2, 10, 0, 0, 0, location).UTC(), MemberUUID: "member-a", LinkedAtTap: true, IdempotencyScope: "test", IdempotencyKey: "2"},
		{OccurredAt: time.Date(2026, time.September, 2, 10, 15, 0, 0, location).UTC(), MemberUUID: "member-a", LinkedAtTap: true, IdempotencyScope: "test", IdempotencyKey: "3"},
		{OccurredAt: time.Date(2026, time.September, 2, 10, 30, 0, 0, location).UTC(), MemberUUID: "member-b", LinkedAtTap: true, IdempotencyScope: "test", IdempotencyKey: "4"},
		{OccurredAt: time.Date(2026, time.September, 2, 10, 45, 0, 0, location).UTC(), IdempotencyScope: "test", IdempotencyKey: "5"},
		{OccurredAt: time.Date(2026, time.September, 3, 9, 0, 0, 0, location).UTC(), MemberUUID: "member-a", LinkedAtTap: true, IdempotencyScope: "test", IdempotencyKey: "6"},
		{OccurredAt: time.Date(2026, time.September, 2, 11, 0, 0, 0, location).UTC(), MemberUUID: "member-d", LinkedAtTap: false, IdempotencyScope: "test", IdempotencyKey: "7"},
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
	link := models.UserUpdate{
		Model:  models.Model{CreatedAt: time.Date(2026, time.September, 3, 12, 30, 0, 0, location).UTC()},
		UserID: accounts[1].ID, Field: "card_id", NewValue: "example-card",
	}
	if err := db.Create(&link).Error; err != nil {
		t.Fatal(err)
	}

	response, err := BuildActivityResponse(db, "semester", now, location)
	if err != nil {
		t.Fatal(err)
	}
	if response.Today.Visitors != 1 || response.Today.Checkins != 1 || response.Today.NewAccounts != 1 || response.Today.NewlyLinkedCards != 1 {
		t.Fatalf("unexpected today summary: %+v", response.Today)
	}
	if response.Week.Visitors != 2 || response.Week.UnlinkedCardHolders != 1 || response.Week.Checkins != 6 || response.Week.NewAccounts != 1 {
		t.Fatalf("unexpected week summary: %+v", response.Week)
	}
	if response.Selected.Visitors != 3 || response.Selected.UnlinkedCardHolders != 1 || response.Selected.Checkins != 7 || response.Selected.NewAccounts != 2 {
		t.Fatalf("unexpected selected summary: %+v", response.Selected)
	}
	if response.Coverage.IdentifiedCheckins != 6 || response.Coverage.TotalCheckins != 7 || response.Coverage.FirstCardLink == "" {
		t.Fatalf("unexpected coverage: %+v", response.Coverage)
	}
	var wednesdayTen int
	for _, cell := range response.Heatmap {
		if cell.Weekday == int(time.Wednesday) && cell.Hour == 10 {
			wednesdayTen = cell.Members
		}
	}
	if wednesdayTen != 2 {
		t.Fatalf("Wednesday 10am heat cell = %d, want 2", wednesdayTen)
	}
	if len(response.AcademicYears) != 3 || response.AcademicYears[2].Label != "2026–27" || response.AcademicYears[2].NewAccounts != 2 {
		t.Fatalf("unexpected academic-year comparison: %+v", response.AcademicYears)
	}
}

func TestUnknownCardDailyCountsFeedThePulse(t *testing.T) {
	db := newCheckinExportTestDB(t)
	if err := db.AutoMigrate(&models.UserUpdate{}, &models.Feed{}, &models.FeedMessage{}, &models.CheckinUnknownDaily{}); err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation(activityTimezone)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 3, 14, 0, 0, 0, location)
	yesterday := time.Date(2026, time.September, 2, 10, 0, 0, 0, location).UTC()
	expiredDay := time.Date(2026, time.August, 27, 10, 0, 0, 0, location).UTC()

	feed := models.Feed{Name: checkinFeedName}
	if err := db.Create(&feed).Error; err != nil {
		t.Fatal(err)
	}
	cardOne, cardTwo := "fingerprint-one", "fingerprint-two"
	items := []models.FeedMessage{
		{Model: models.Model{CreatedAt: yesterday}, FeedID: feed.ID, PendingCardFingerprint: &cardOne},
		{Model: models.Model{CreatedAt: yesterday.Add(time.Hour)}, FeedID: feed.ID, PendingCardFingerprint: &cardOne},
		{Model: models.Model{CreatedAt: yesterday.Add(2 * time.Hour)}, FeedID: feed.ID, PendingCardFingerprint: &cardTwo},
		// August 27 began before the seven-day cutoff, so its saved count must stay frozen.
		{Model: models.Model{CreatedAt: expiredDay.Add(6 * time.Hour)}, FeedID: feed.ID, PendingCardFingerprint: &cardTwo},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.CheckinUnknownDaily{Day: "2026-08-27", DistinctCards: 9}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := recordUnknownCardDailyCounts(db, now, location); err != nil {
		t.Fatal(err)
	}
	var saved []models.CheckinUnknownDaily
	if err := db.Order("day ASC").Find(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if len(saved) != 2 || saved[0].DistinctCards != 9 || saved[1].Day != "2026-09-02" || saved[1].DistinctCards != 2 {
		t.Fatalf("unexpected saved unknown-card counts: %+v", saved)
	}

	events := []models.CheckinEvent{
		{OccurredAt: yesterday, IdempotencyScope: "test", IdempotencyKey: "u1"},
		{OccurredAt: yesterday.Add(time.Hour), IdempotencyScope: "test", IdempotencyKey: "u2"},
		{OccurredAt: yesterday.Add(2 * time.Hour), IdempotencyScope: "test", IdempotencyKey: "u3"},
		{OccurredAt: yesterday.Add(3 * time.Hour), MemberUUID: "member-a", LinkedAtTap: true, IdempotencyScope: "test", IdempotencyKey: "m1"},
		{OccurredAt: yesterday.Add(4 * time.Hour), MemberUUID: "member-b", LinkedAtTap: false, IdempotencyScope: "test", IdempotencyKey: "m2"},
	}
	if err := db.Create(&events).Error; err != nil {
		t.Fatal(err)
	}
	response, err := BuildActivityResponse(db, "semester", now, location)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Pulse) != 3 {
		t.Fatalf("pulse windows = %d, want 3", len(response.Pulse))
	}
	week := response.Pulse[1]
	// Four people is below the open-day minimum: counted, but not averaged.
	if week.Key != "7_days" || week.OpenDays != 0 || week.People != 4 || week.AvgDailyPeople != 0 || week.NotLinkedPeople != 3 || week.NotLinkedPercent != 75 || week.Checkins != 5 {
		t.Fatalf("unexpected 7-day pulse: %+v", week)
	}
	if today := response.Pulse[0]; today.OpenDays != 0 || today.People != 0 {
		t.Fatalf("unexpected today pulse: %+v", today)
	}
	// Two distinct unknown cards and two members in the past seven days.
	if response.StillUnlinked.Cards != 2 || response.StillUnlinked.Visitors != 4 || response.StillUnlinked.Percent != 50 {
		t.Fatalf("unexpected still-unlinked summary: %+v", response.StillUnlinked)
	}
	if response.HeatmapOpenDays[int(time.Wednesday)] != 1 {
		t.Fatalf("unexpected heatmap open days: %+v", response.HeatmapOpenDays)
	}
}

func TestPulseAveragesOnlyOpenDays(t *testing.T) {
	location, err := time.LoadLocation(activityTimezone)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, time.September, 1, 0, 0, 0, 0, location)
	var events []activityEvent
	for index := 0; index < 6; index++ {
		events = append(events, activityEvent{OccurredAt: start.Add(10 * time.Hour), MemberUUID: fmt.Sprintf("member-%d", index), LinkedAtTap: true})
	}
	// A lone staff tap on a closed day.
	events = append(events, activityEvent{OccurredAt: start.AddDate(0, 0, 1).Add(10 * time.Hour), MemberUUID: "member-0", LinkedAtTap: true})

	pulse := pulseFor("7_days", "Past 7 days", events, nil, nil, map[string]int{}, start, start.AddDate(0, 0, 7), location)
	if pulse.OpenDays != 1 || pulse.People != 7 || pulse.AvgDailyPeople != 6 || pulse.Checkins != 7 {
		t.Fatalf("unexpected pulse: %+v", pulse)
	}
}
