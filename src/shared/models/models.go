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

var enforcer *casbin.Enforcer

// SetupEnforcer sets up the enforcer for the model AfterFind hooks
func SetupEnforcer(e *casbin.Enforcer) {
	enforcer = e
}

type Model struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:",omitempty"`
}

type User struct {
	Model
	Email          string `gorm:"unique"`
	PendingEmail   string `gorm:"unique" json:",omitempty"`
	CardID         uint64 `gorm:"unique"`
	Name           string
	Role           string
	Type           string
	GraduationYear int
	Major          string

	Trainings     []Training     `json:",omitempty"`
	Holds         []Hold         `json:",omitempty"`
	APIKeys       []APIKey       `json:",omitempty"`
	UserUpdates   []UserUpdate   `json:",omitempty"`
	Notifications []Notification `json:",omitempty"`

	Permissions []string `gorm:"-"`
}

// AfterFind GORM hook that loads the permissions for a user from casbin
func (u *User) AfterFind(tx *gorm.DB) (err error) {
	u.Permissions = []string{}
	for _, p := range enforcer.GetPermissionsForUser(fmt.Sprintf("user:%d", u.ID)) {
		u.Permissions = append(u.Permissions, p[1])
	}

	return nil
}

type APIKey struct {
	Key         string `gorm:"column:api_key;primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index" json:",omitempty"`
	UserID      uint
	Description string
	FullAccess  bool

	Permissions []string `gorm:"-"`
}

// AfterFind GORM hook that loads the permissions for an api key from casbin
func (a *APIKey) AfterFind(tx *gorm.DB) (err error) {
	a.Permissions = []string{}
	for _, p := range enforcer.GetPermissionsForUser("apikey:" + a.Key) {
		a.Permissions = append(a.Permissions, p[1])
	}

	return nil
}

type Training struct {
	Model
	UserID       uint
	TrainingType string
	AddedBy      uint
	RemovedBy    uint `json:",omitempty"`
}

type Hold struct {
	Model
	UserID    uint
	HoldType  string
	Reason    string
	HoldStart *time.Time
	HoldEnd   *time.Time
	AddedBy   uint
	RemovedBy uint `json:",omitempty"`
	Priority  int
}

type UserUpdate struct {
	Model
	UserID   uint
	EditedBy uint
	Field    string
	NewValue string
	OldValue string
}

type Notification struct {
	Model
	UserID    uint
	AddedBy   uint
	RemovedBy uint `json:",omitempty"`
	Title     string
	Message   string
	Link      string
	Group     string
}

type Session struct {
	SessionID string `gorm:"column:api_key;primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:",omitempty"`
	UserID    uint
	ExpiresAt time.Time
}

var validate = validator.New()

type ErrorResponse struct {
	FailedField string
	Tag         string
	Value       string
}

// ValidateStruct validates a struct and returns a list of errors
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

// SetupValidator sets up the validator with custom validation tags
func SetupValidator() error {
	// Add custom validation tags
	return validate.RegisterValidation("notblank", val.NotBlank)
}

// GetBodyMiddleware is a middleware that parses the body into a struct and validates it
func GetBodyMiddleware[V interface{}](c *fiber.Ctx) error {
	var req V

	// Parse the body into the req struct
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	// Validate the struct
	errors := ValidateStruct(req)
	if errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errors)
	}

	// If everything is good, set the body in the locals
	c.Locals("body", req)
	return c.Next()
}

// GetQueryMiddleware is a middleware that parses the query into a struct and validates it
func GetQueryMiddleware[V interface{}](c *fiber.Ctx) error {
	var req V

	// Parse the query into the req struct
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	// Validate the struct
	errors := ValidateStruct(req)
	if errors != nil {
		return c.Status(fiber.StatusBadRequest).JSON(errors)
	}

	// If everything is good, set the query in the locals
	c.Locals("query", req)
	return c.Next()
}
