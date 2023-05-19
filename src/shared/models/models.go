package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email          string `gorm:"unique"`
	Admin          bool
	FirstName      string
	LastName       string
	GraduationYear int
	Type           string
	Major          string
	Enabled        bool

	Trainings []Training
	APIKeys   []APIKey
}

type APIKey struct {
	gorm.Model
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
	gorm.Model
	UserID       uint `gorm:"foreignKey:user_id"`
	TrainingType string
	AddedBy      uint `gorm:"foreignKey:user_id"`
}
