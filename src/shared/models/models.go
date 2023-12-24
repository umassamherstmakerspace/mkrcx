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
	PendingEmail   string `gorm:"unique"`
	CardID         uint64 `gorm:"unique"`
	Name           string
	Role           string
	Type           string
	GraduationYear int
	Major          string
	Enabled        bool

	Trainings   []Training   `json:",omitempty"`
	Holds       []Hold       `json:",omitempty"`
	APIKeys     []APIKey     `json:"-"`
	UserUpdates []UserUpdate `json:",omitempty"`
}

type APIKey struct {
	Key         string `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	UserID      uint           `gorm:"foreignKey:user_id"`
	Description string
	Permissions string
}

type Training struct {
	Model
	UserID       uint `gorm:"foreignKey:user_id"`
	TrainingType string
	AddedBy      uint `gorm:"foreignKey:user_id"`
	RemovedBy    uint `gorm:"foreignKey:user_id"`
}

type Hold struct {
	Model
	UserID    uint `gorm:"foreignKey:user_id"`
	HoldType  string
	Reason    string
	HoldStart *time.Time
	HoldEnd   *time.Time
	AddedBy   uint `gorm:"foreignKey:user_id"`
	RemovedBy uint `gorm:"foreignKey:user_id"`
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
