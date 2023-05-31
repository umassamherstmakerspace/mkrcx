package models

import (
	"time"

	"gorm.io/gorm"
)

type Model struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:",omitempty"`
}

type User struct {
	Model
	Email          string `gorm:"unique"`
	Admin          bool
	FirstName      string
	LastName       string
	Role           string
	Type           string
	GraduationYear int
	Major          string
	Enabled        bool

	Trainings []Training
	APIKeys   []APIKey `json:"-"`
}

type APIKey struct {
	Model
	UserID      uint
	ID          string `gorm:"unique"`
	Description string
	Scope       string
}

func APIKeyValidate(key APIKey, permission string) bool {
	scope := key.Scope
	permits := false
	permissionIdx := 0
	commaScan := false
	for _, c := range scope {
		if commaScan {
			if c == ',' {
				commaScan = false
			}
			continue
		}
		if c == '*' {
			return true
		}
		if permission[permissionIdx] == byte(c) {
			permissionIdx++
			if permissionIdx == len(permission) {
				return true
			}
		} else {
			permissionIdx = 0
			commaScan = true
		}
		if permissionIdx == len(permission) {
			break
		}
	}
	return permits
}

type Training struct {
	Model
	UserID       uint `gorm:"foreignKey:user_id"`
	TrainingType string
	AddedBy      uint `gorm:"foreignKey:user_id"`
	RemovedBy    uint `gorm:"foreignKey:user_id"`
}
