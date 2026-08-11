package main

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

type backfillTestUser struct {
	ID     uint `gorm:"primaryKey"`
	CardID *string
}

func (backfillTestUser) TableName() string { return "users" }

func backfillTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func stringRef(value string) *string { return &value }

func TestBackfillIsDryRunByDefaultAndImportsStablePrivacySafeIdentity(t *testing.T) {
	source := backfillTestDB(t, "source")
	target := backfillTestDB(t, "target")
	if err := source.AutoMigrate(&sourceSwipe{}); err != nil {
		t.Fatal(err)
	}
	if err := target.AutoMigrate(&backfillTestUser{}, &models.CheckinIdentity{}, &models.CheckinEvent{}); err != nil {
		t.Fatal(err)
	}
	if err := target.Create(&backfillTestUser{ID: 7, CardID: stringRef("ABCDEF0123456789")}).Error; err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	swipes := []sourceSwipe{
		{ID: 1, CreatedAt: start, CardNumber: "abcdef0123456789"},
		{ID: 2, CreatedAt: start.Add(time.Hour), CardNumber: "ABCDEF0123456789"},
		{ID: 3, CreatedAt: start.Add(2 * time.Hour), CardNumber: "1111111111111111"},
		{ID: 4, CreatedAt: start.Add(3 * time.Hour), CardNumber: "not-a-card"},
	}
	if err := source.Create(&swipes).Error; err != nil {
		t.Fatal(err)
	}

	dryRun, err := runBackfill(source, target, backfillOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.Scanned != 4 || dryRun.Linked != 2 || dryRun.Unresolved != 2 || dryRun.InvalidCard != 1 || dryRun.WouldInsert != 4 || dryRun.Inserted != 0 {
		t.Fatalf("unexpected dry-run stats: %+v", dryRun)
	}
	var count int64
	if err := target.Model(&models.CheckinEvent{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("dry run wrote %d events (%v)", count, err)
	}
	if err := target.Model(&models.CheckinIdentity{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("dry run wrote %d identities (%v)", count, err)
	}

	applied, err := runBackfill(source, target, backfillOptions{Apply: true, BatchSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if applied.Inserted != 4 || applied.WouldInsert != 4 || applied.AlreadyExists != 0 {
		t.Fatalf("unexpected apply stats: %+v", applied)
	}

	var events []models.CheckinEvent
	if err := target.Order("occurred_at ASC").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 4 {
		t.Fatalf("event count = %d, want 4", len(events))
	}
	if events[0].MemberUUID == "" || events[0].MemberUUID != events[1].MemberUUID || events[0].UserID != 7 || events[1].UserID != 7 {
		t.Fatalf("linked historical identity is missing or unstable: %+v %+v", events[0], events[1])
	}
	for _, event := range events[:2] {
		if event.LinkedAtTap || event.IdentityResolution != historicalResolution || event.Decision != "unknown" || event.Source != historicalSource {
			t.Fatalf("historical certainty was overstated: %+v", event)
		}
	}
	for _, event := range events[2:] {
		if event.UserID != 0 || event.MemberUUID != "" || event.IdentityResolution != "unresolved" {
			t.Fatalf("unresolved historical event became correlatable: %+v", event)
		}
	}

	rerun, err := runBackfill(source, target, backfillOptions{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if rerun.Inserted != 0 || rerun.AlreadyExists != 4 || rerun.WouldInsert != 0 {
		t.Fatalf("rerun was not idempotent: %+v", rerun)
	}
}

func TestBackfillLeavesNormalizedOwnershipConflictsUnresolved(t *testing.T) {
	source := backfillTestDB(t, "ambiguous-source")
	target := backfillTestDB(t, "ambiguous-target")
	if err := source.AutoMigrate(&sourceSwipe{}); err != nil {
		t.Fatal(err)
	}
	if err := target.AutoMigrate(&backfillTestUser{}, &models.CheckinIdentity{}, &models.CheckinEvent{}); err != nil {
		t.Fatal(err)
	}
	users := []backfillTestUser{
		{ID: 1, CardID: stringRef("abcdef0123456789")},
		{ID: 2, CardID: stringRef("ABCDEF0123456789")},
	}
	if err := target.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	if err := source.Create(&sourceSwipe{ID: 1, CreatedAt: time.Now().UTC(), CardNumber: "abcdef0123456789"}).Error; err != nil {
		t.Fatal(err)
	}

	stats, err := runBackfill(source, target, backfillOptions{Apply: true})
	if err != nil {
		t.Fatal(err)
	}
	if stats.AmbiguousCard != 1 || stats.Unresolved != 1 || stats.Linked != 0 {
		t.Fatalf("unexpected ambiguous-card stats: %+v", stats)
	}
	var event models.CheckinEvent
	if err := target.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.UserID != 0 || event.MemberUUID != "" {
		t.Fatalf("ambiguous card was assigned an identity: %+v", event)
	}
}

func TestBackfillRejectsInvalidRangeAndBatchSize(t *testing.T) {
	start := time.Now().UTC()
	end := start.Add(-time.Hour)
	for _, options := range []backfillOptions{
		{Start: &start, End: &end},
		{BatchSize: -1},
		{BatchSize: maxBatchSize + 1},
	} {
		if err := validateBackfillOptions(&options); err == nil {
			t.Fatalf("invalid options accepted: %+v", options)
		}
	}
}
