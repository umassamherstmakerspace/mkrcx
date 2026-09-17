package leash_backend_api

import (
	"strings"
	"time"

	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Only source-verified staff field changes affect the assessment. Connectivity,
// print outcomes, overrides without a condition edit and old unknown comparisons do not.
// Each field has its own source time so an out-of-order note cannot mask a condition edit.
func applyPrinterStationAssessment(tx *gorm.DB, event models.PrinterHistoryEvent) error {
	if event.EventType != "staff_runtime_changed" || event.HiddenReason != "" {
		return nil
	}
	conditionChanged := event.ConditionChanged != nil && *event.ConditionChanged
	noteChanged := event.NoteChanged != nil && *event.NoteChanged
	if !conditionChanged && !noteChanged {
		return nil
	}
	line, note, hasNote := strings.Cut(event.Detail, "\nNote: ")
	// Empty notes are trimmed by the collector's clean_text().
	if !hasNote && strings.HasSuffix(event.Detail, "\nNote:") {
		line = strings.TrimSuffix(event.Detail, "\nNote:")
		hasNote = true
	}
	condition := map[string]string{"available": "working", "needs_attention": "limited", "out_of_service": "out"}[strings.TrimPrefix(line, "Condition: ")]
	if !strings.HasPrefix(line, "Condition: ") || condition == "" || !printerText(note, 2000) {
		return nil
	}
	var record models.PrinterRecord
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&record, "id = ?", event.PrinterID).Error; err != nil {
		return err
	}
	newer := func(at *time.Time) bool {
		if at == nil {
			return event.RecordedAt.After(record.UpdatedAt)
		}
		return event.RecordedAt.After(*at)
	}
	updates := map[string]interface{}{}
	if conditionChanged && newer(record.ConditionSetAt) {
		updates["condition"] = condition
		updates["condition_set_at"] = event.RecordedAt
	}
	if noteChanged && hasNote && newer(record.NoteSetAt) {
		updates["note"] = note
		updates["note_set_at"] = event.RecordedAt
	}
	if len(updates) == 0 {
		return nil
	}
	// Freeze both fallback clocks before UpdatedAt changes, including newly created records.
	if record.ConditionSetAt == nil && updates["condition_set_at"] == nil {
		updates["condition_set_at"] = record.UpdatedAt
	}
	if record.NoteSetAt == nil && updates["note_set_at"] == nil {
		updates["note_set_at"] = record.UpdatedAt
	}
	updates["manual"] = true
	updates["version"] = record.Version + 1
	updates["updated_at"] = time.Now().UTC()
	updates["updated_by"] = event.SourceID
	// The existing station history row is the audit entry; don't duplicate it as a website edit.
	return tx.Model(&models.PrinterRecord{}).Where("id = ?", record.ID).UpdateColumns(updates).Error
}
