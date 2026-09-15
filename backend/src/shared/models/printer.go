package models

import "time"

// PrinterRecord follows a physical machine; display names and network addresses can change.
// Staff records and observations have separate versions and timestamps.
type PrinterRecord struct {
	ID        string  `gorm:"primaryKey;size:80" json:"id"`
	Name      string  `gorm:"size:120;not null" json:"name"`
	Model     string  `gorm:"size:80;not null" json:"model"`
	MachineID string  `gorm:"size:120" json:"machineId"`
	Location  string  `gorm:"size:120" json:"location"`
	Lifecycle string  `gorm:"size:24;not null" json:"lifecycle"`
	Host      string  `gorm:"size:64" json:"host"`
	MAC       string  `gorm:"size:17" json:"mac"`
	HostKey   *string `gorm:"uniqueIndex;size:64" json:"-"`
	MACKey    *string `gorm:"uniqueIndex;size:17" json:"-"`
	Condition string  `gorm:"size:24" json:"condition"`
	Note      string  `gorm:"type:text" json:"note"`
	// Manual condition/note edits remain authoritative until explicitly returned to station reports.
	Manual              bool       `json:"manual"`
	Version             uint64     `gorm:"not null" json:"version"`
	UpdatedAt           time.Time  `json:"updatedAt"`
	UpdatedBy           string     `gorm:"size:160" json:"updatedBy"`
	ObservedCondition   string     `gorm:"size:24" json:"-"`
	ObservedNote        string     `gorm:"type:text" json:"-"`
	ConditionObservedAt *time.Time `json:"-"`
	LastSeen            *time.Time `json:"lastSeen"`
	HistorySyncedAt     *time.Time `json:"-"`
	FetchedAt           *time.Time `json:"-"`
	Telemetry           string     `gorm:"type:text" json:"-"`
}

type PrinterRecordEvent struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	PrinterID string    `gorm:"size:80;index" json:"printerId"`
	Version   uint64    `json:"version"`
	Actor     string    `gorm:"size:160" json:"actor"`
	CreatedAt time.Time `json:"createdAt"`
	Record    string    `gorm:"type:text" json:"record"`
}

// A projection of an immutable station event, retained independently of live telemetry.
type PrinterHistoryEvent struct {
	SourceID   string    `gorm:"primaryKey;size:160" json:"sourceId"`
	PrinterID  string    `gorm:"size:80;index" json:"printerId"`
	RecordedAt time.Time `gorm:"index" json:"recordedAt"`
	EventType  string    `gorm:"size:80" json:"eventType"`
	Detail     string    `gorm:"type:text" json:"detail"`
	File       string    `gorm:"size:1000" json:"file,omitempty"`
	Material   string    `gorm:"size:120" json:"material,omitempty"`
}
