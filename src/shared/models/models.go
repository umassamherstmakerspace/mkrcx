package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID      string `gorm:"primaryKey"`
	Admin   bool
	Email   string `gorm:"unique"`
	Name    string
	Enabled bool

	Trainings []Training
	APIKeys   []APIKey
}

type APIKey struct {
	gorm.Model
	UserID      string
	Key         string `gorm:"unique"`
	Description string
}

type Training struct {
	gorm.Model
	UserID       string
	TrainingType string
	AddedBy      string
}
