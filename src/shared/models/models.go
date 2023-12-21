package models

import (
	"time"

	"github.com/go-playground/validator/v10"
	val "github.com/go-playground/validator/v10/non-standard/validators"
	"gorm.io/gorm"
)

type Model struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type User struct {
	Model
	Email          string `gorm:"unique"`
	Admin          bool
	Name           string
	Role           string
	Type           string
	GraduationYear int
	Major          string
	Enabled        bool

	Trainings   []Training   `json:",omitempty"`
	APIKeys     []APIKey     `json:"-"`
	UserUpdates []UserUpdate `json:",omitempty"`
}

type APIKey struct {
	ID          string `gorm:"primarykey,unique"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	UserID      uint
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

type UserUpdate struct {
	Model
	UserID   uint `gorm:"foreignKey:user_id"`
	EditedBy uint `gorm:"foreignKey:user_id"`
	Field    string
	NewValue string
	OldValue string
}

var validate = validator.New()

type ErrorResponse struct {
	FailedField string
	Tag         string
	Value       string
}

func ValidateStruct(s interface{}) []*ErrorResponse {
	var errors []*ErrorResponse
	err := validate.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.FailedField = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errors = append(errors, &element)
		}
	}
	return errors
}

func SetupValidator() error {
	return validate.RegisterValidation("notblank", val.NotBlank)
}
