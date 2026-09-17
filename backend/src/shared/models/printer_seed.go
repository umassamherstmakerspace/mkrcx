package models

import (
	_ "embed"
	"encoding/json"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

//go:embed printer-seed.json
var printerSeed []byte

func MigratePrinterRegistry(db *gorm.DB) error {
	if err := db.AutoMigrate(&PrinterRecord{}, &PrinterRecordEvent{}, &PrinterHistoryEvent{}, &PrinterSummary{}); err != nil {
		return err
	}
	historyDB := db
	if db.Dialector.Name() == "mysql" {
		// Source reports can contain emoji even when the legacy database defaults to utf8mb3.
		historyDB = db.Set("gorm:table_options", "CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")
	}
	if err := historyDB.AutoMigrate(&PrinterHistoricalEntry{}); err != nil {
		return err
	}
	var records []PrinterRecord
	if err := json.Unmarshal(printerSeed, &records); err != nil {
		return err
	}
	for i := range records {
		records[i].Manual = true
		records[i].UpdatedAt = time.Now().UTC()
		if records[i].Host != "" {
			records[i].HostKey = &records[i].Host
			records[i].MACKey = &records[i].MAC
		}
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error; err != nil {
		return err
	}
	// Separate maintenance from fleet membership, preserving visibility, version and note dates.
	return db.Transaction(func(tx *gorm.DB) error {
		for _, legacy := range []string{"testing", "repair"} {
			if err := tx.Model(&PrinterRecord{}).Where("lifecycle = ?", legacy).UpdateColumns(map[string]interface{}{
				"lifecycle": "active", "maintenance": legacy,
			}).Error; err != nil {
				return err
			}
		}
		// Carry existing visible conditions/notes into the independent website assessment.
		// This is a baseline, not a new staff verification or a change to printer controls.
		var legacy []PrinterRecord
		if err := tx.Where("manual = ?", false).Find(&legacy).Error; err != nil {
			return err
		}
		for _, record := range legacy {
			condition := record.ObservedCondition
			if condition == "" {
				condition = "unknown"
			}
			updates := map[string]interface{}{"condition": condition, "note": record.ObservedNote, "manual": true}
			if record.ConditionObservedAt != nil {
				updates["updated_at"] = *record.ConditionObservedAt
			}
			if err := tx.Model(&PrinterRecord{}).Where("id = ? AND manual = ?", record.ID, false).UpdateColumns(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
