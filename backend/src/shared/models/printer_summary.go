package models

import "time"

type PrinterSource struct {
	URL   string `json:"url"`
	Label string `json:"label"`
}

// PrinterSummary is a source-linked report prepared during staff review.
// Its report date does not assert a repair-completion time or change current state.
type PrinterSummary struct {
	SourceID   string          `gorm:"primaryKey;size:160" json:"sourceId"`
	PrinterID  string          `gorm:"size:80;index" json:"printerId"`
	ReportDate string          `gorm:"size:10;index" json:"reportDate"`
	Body       string          `gorm:"type:text" json:"body"`
	Sources    []PrinterSource `gorm:"serializer:json;type:text" json:"sources"`
	PreparedBy string          `gorm:"size:200" json:"preparedBy"`
	Actor      string          `gorm:"size:160" json:"-"`
	ImportedAt time.Time       `json:"importedAt"`
}
