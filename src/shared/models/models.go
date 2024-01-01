package models

import (
	"fmt"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/go-playground/validator/v10"
	val "github.com/go-playground/validator/v10/non-standard/validators"
	"github.com/gofiber/fiber/v2"
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

	Permissions []string `gorm:"-"`
}

func (u *User) LoadPermissions(enforcer *casbin.Enforcer) {
	for _, p := range enforcer.GetPermissionsForUser(fmt.Sprintf("user:%d", u.ID)) {
		u.Permissions = append(u.Permissions, p[1])
	}
}

type APIKey struct {
	Key         string `gorm:"column:api_key;primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	UserID      uint           `gorm:"foreignKey:user_id"`
	Description string
	FullAccess  bool

	Permissions []string `gorm:"-"`
}

func (a *APIKey) LoadPermissions(enforcer *casbin.Enforcer) {
	for _, p := range enforcer.GetPermissionsForUser("apikey:" + a.Key) {
		a.Permissions = append(a.Permissions, p[1])
	}
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

func GetBodyMiddleware[V interface{}](c *fiber.Ctx) error {
	var req V

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	errors := ValidateStruct(req)
	if errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errors)
	}

	c.Locals("body", req)
	return c.Next()
}

func GetQueryMiddleware[V interface{}](c *fiber.Ctx) error {
	var req V

	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	errors := ValidateStruct(req)
	if errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errors)
	}

	c.Locals("query", req)
	return c.Next()
}
