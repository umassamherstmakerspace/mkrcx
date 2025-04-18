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
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:",omitempty"`
}

type User struct {
	Model
	ID           uint    `gorm:"primarykey"`
	Email        string  `gorm:"unique"`
	PendingEmail *string `gorm:"unique" json:",omitempty"`
	CardID       *string `gorm:"unique"`
	Name         string
	Pronouns     string
	Role         string
	Type         string

	// Student-like fields
	GraduationYear int
	Major          string

	// Employee-like fields
	Department string
	JobTitle   string

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
	perms, err := enforcer.GetPermissionsForUser(fmt.Sprintf("user:%d", u.ID))
	if err != nil {
		return err
	}

	for _, p := range perms {
		u.Permissions = append(u.Permissions, p[1])
	}

	return nil
}

// AfterCreate GORM hook that sets the permissions if they are nil
func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	if u.Permissions == nil {
		u.Permissions = []string{}
	}

	return nil
}

type APIKey struct {
	Model
	Key         string `gorm:"column:api_key;primaryKey;size:36"`
	UserID      uint
	Description string
	FullAccess  bool

	Permissions []string `gorm:"-"`
}

// AfterFind GORM hook that loads the permissions for an api key from casbin
func (a *APIKey) AfterFind(tx *gorm.DB) (err error) {
	a.Permissions = []string{}
	perms, err := enforcer.GetPermissionsForUser("apikey:" + a.Key)
	if err != nil {
		return err
	}

	for _, p := range perms {
		a.Permissions = append(a.Permissions, p[1])
	}

	return nil
}

// AfterCreate GORM hook that sets the permissions if they are nil
func (a *APIKey) AfterCreate(tx *gorm.DB) (err error) {
	if a.Permissions == nil {
		a.Permissions = []string{}
	}

	return nil
}

type Training struct {
	Model
	ID        uint `gorm:"primarykey"`
	UserID    uint
	Name      string
	Level     string
	AddedBy   uint
	RemovedBy uint `json:",omitempty"`
}

type Hold struct {
	Model
	ID             uint `gorm:"primarykey"`
	UserID         uint
	Name           string
	Reason         string
	Start          *time.Time
	End            *time.Time
	ResolutionLink string `json:",omitempty"`
	AddedBy        uint
	RemovedBy      uint `json:",omitempty"`
	Priority       int
}

// AfterFind GORM hook that clears expired holds
func (h *Hold) AfterFind(tx *gorm.DB) (err error) {
	if h.End != nil && h.End.Before(time.Now()) {
		h.RemovedBy = h.AddedBy
		h.DeletedAt = gorm.DeletedAt{Time: *h.End, Valid: true}
		tx.Save(h)
	}

	return nil
}

type UserUpdate struct {
	Model
	ID       uint `gorm:"primarykey"`
	UserID   uint
	EditedBy uint
	Field    string
	NewValue string
	OldValue string
}

type Notification struct {
	Model
	ID        uint `gorm:"primarykey"`
	UserID    uint
	AddedBy   uint
	RemovedBy uint `json:",omitempty"`
	Title     string
	Message   string
	Link      string
	Group     string
}

type Session struct {
	Model
	SessionID string `gorm:"column:api_key;primaryKey"`
	UserID    uint
	ExpiresAt time.Time
}

type Feed struct {
	Model
	ID       uint `gorm:"primarykey"`
	Name     string
	Messages []FeedMessage
}

type FeedMessage struct {
	Model
	ID                   uint `gorm:"primarykey"`
	FeedId               uint
	AddedBy              uint
	LogLevel             uint
	UserID               uint
	Title                string
	Message              string
	PendingUserSpecifier string `json:",omitempty"`
	PendingUserData      string `json:",omitempty"`
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
