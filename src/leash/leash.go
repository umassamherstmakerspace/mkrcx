package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	models "github.com/spectrum-control/spectrum/src/shared/models"
)

type ctxUserKey struct{}

const SYSTEM_USER_EMAIL = "makerspace@umass.edu"
const HOST = ":8000"

func main() {
	db, err := gorm.Open(mysql.Open(os.Getenv("API")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&models.APIKey{})
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Training{})

	// Debug
	makeMakerspaceSystemUser(db)

	app := fiber.New()

	// Create a new user
	app.Post("/users", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiUser := c.Locals(ctxUserKey{}).(models.User)

		// Make sure API user is system user
		if apiUser.Email != SYSTEM_USER_EMAIL {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		// Get the user's email and name from the request body
		email := c.FormValue("email")
		name := c.FormValue("name")

		// Check if the user already exists
		{
			var user models.User
			res := db.First(&user, "email = ?", email)
			if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
				// The user already exists
				return c.Status(fiber.StatusConflict).SendString("User already exists")
			}
		}

		// Create a new user in the database
		user := models.User{
			Email:   email,
			Name:    name,
			ID:      uuid.NewString(),
			Enabled: false,
		}
		db.Create(&user)

		// Write a success message to the response
		return c.SendString("User created successfully")
	}))

	// Add completed training to a user
	app.Post("/training", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiUser := c.Locals(ctxUserKey{}).(models.User)

		if !apiUser.Admin {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		// Get the user's email and training type from the request body
		email := c.FormValue("email")
		trainingType := c.FormValue("training_type")

		// Check if the user exists
		var user models.User
		res := db.First(&user, "email = ?", email)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusNotFound).SendString("User not found")
		}

		// Create a new training in the database
		training := models.Training{
			UserID:       user.ID,
			TrainingType: trainingType,
			AddedBy:      apiUser.ID,
		}

		db.Create(&training)

		// If the user has completed the trainings "orientation" and "docusign", enable the user
		userTrainingEnable(db, user)

		// Write a success message to the response
		return c.SendString("Training added successfully")
	}))

	// Used for debugging
	// List all users
	app.Get("/users", func(c *fiber.Ctx) error {
		var users []models.User
		db.Find(&users)

		// Write the users to the response as json
		msg, err := json.Marshal(users)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Error marshalling users")
		}
		return c.SendString(string(msg))
	})

	log.Printf("Starting server on port %s\n", HOST)
	app.Listen(HOST)
}

func userTrainingEnable(db *gorm.DB, user models.User) {
	var trainings []models.Training
	db.Find(&trainings, "user_id = ?", user.ID)
	orientationCompleted := false
	docusignCompleted := false
	for _, training := range trainings {
		if training.TrainingType == "orientation" {
			orientationCompleted = true
		}
		if training.TrainingType == "docusign" {
			docusignCompleted = true
		}
	}

	if orientationCompleted && docusignCompleted {
		user.Enabled = true
		db.Save(&user)
	}

}

func makeMakerspaceSystemUser(db *gorm.DB) {
	var systemUser models.User
	res := db.First(&systemUser, "email = ?", SYSTEM_USER_EMAIL)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		// Create a system user
		user := models.User{
			Email:   SYSTEM_USER_EMAIL,
			Name:    "System",
			ID:      uuid.NewString(),
			Admin:   true,
			Enabled: true,
		}

		// Create a default API key
		apiKey := models.APIKey{
			Key:         uuid.NewString(),
			UserID:      user.ID,
			Description: "Default API key",
		}
		db.Create(&user)
		db.Create(&apiKey)

		log.Printf("API key: %s\n", apiKey.Key)
		return
	}

}

func apiKeyAuthMiddleware(db *gorm.DB, next fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the API key from the request header
		apiKey := c.Get("API-Key")
		var apiKeyRecord models.APIKey
		res := db.First(&apiKeyRecord, "key = ?", apiKey)

		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The API key is not valid
			return c.Status(fiber.StatusUnauthorized).SendString("Invalid API key")
		}

		var user models.User
		res = db.First(&user, "id = ?", apiKeyRecord.UserID)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusUnauthorized).SendString("User not found")
		}

		c.Locals(ctxUserKey{}, user)
		return next(c)
	}
}
