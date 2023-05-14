package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email   string `gorm:"unique"`
	Admin   bool
	Name    string
	Enabled bool

	Trainings []Training
	APIKeys   []APIKey
}

type APIKey struct {
	gorm.Model
	UserID      uint
	ID          string `gorm:"unique"`
	Description string
}

type Training struct {
	gorm.Model
	UserID       uint `gorm:"foreignKey:user_id"`
	TrainingType string
	AddedBy      uint `gorm:"foreignKey:user_id"`
}
