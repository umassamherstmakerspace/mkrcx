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
	if err := db.AutoMigrate(&PrinterRecord{}, &PrinterRecordEvent{}); err != nil {
		return err
	}
	var records []PrinterRecord
	if err := json.Unmarshal(printerSeed, &records); err != nil {
		return err
	}
	for i := range records {
		records[i].UpdatedAt = time.Now().UTC()
		if records[i].Host != "" {
			records[i].HostKey = &records[i].Host
			records[i].MACKey = &records[i].MAC
		}
	}
	// Existing records, including retired printers, survive migrations unchanged.
	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&records).Error
}
