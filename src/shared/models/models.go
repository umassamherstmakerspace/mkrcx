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
}

type APIKey struct {
	gorm.Model
	User        User
	UserID      string
	Key         string `gorm:"unique"`
	Description string
}

type Training struct {
	gorm.Model
	User         User
	UserID       string
	TrainingType string
	AddedBy      string
}
