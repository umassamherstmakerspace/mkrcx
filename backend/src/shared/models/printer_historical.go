package models

import "time"

// Reviewed display aliases supplement account names without rewriting source evidence.
// Loaded only for staff history; aliases never grant authentication or permissions.
type PrinterIdentityAlias struct {
	Alias  string `gorm:"primaryKey;size:200" json:"-"`
	Name   string `gorm:"size:200;not null" json:"-"`
	Source string `gorm:"type:text" json:"-"`
}

// Imported source evidence is separate from measured station outcomes and current state.
// Sources and import IDs remain private operational provenance, not public fleet data.
type PrinterHistoricalEntry struct {
	SourceID         string          `gorm:"primaryKey;size:160" json:"sourceId"`
	PrinterID        string          `gorm:"size:80;index:idx_printer_historical_page,priority:1" json:"printerId"`
	RecordedAt       time.Time       `gorm:"index:idx_printer_historical_page,priority:2" json:"recordedAt"`
	DateOnly         bool            `json:"dateOnly"`
	Kind             string          `gorm:"size:24;index" json:"kind"`
	Body             string          `gorm:"type:text" json:"body"`
	Reporter         string          `gorm:"size:200" json:"reporter,omitempty"`
	PreparedBy       string          `gorm:"size:200" json:"preparedBy,omitempty"`
	Person           string          `gorm:"size:200" json:"person,omitempty"`
	File             string          `gorm:"size:1000" json:"file,omitempty"`
	Material         string          `gorm:"size:120" json:"material,omitempty"`
	MeterHours       *float64        `json:"meterHours,omitempty"`
	EstimatedSeconds *float64        `json:"estimatedSeconds,omitempty"`
	Sources          []PrinterSource `gorm:"serializer:json;type:text" json:"-"`
	ImportID         string          `gorm:"size:80;index" json:"-"`
}
