package main

import (
	"errors"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	models "github.com/spectrum-control/spectrum/src/shared/models"
)

type ctxUserKey struct{}
type ctxAPIKey struct{}

const SYSTEM_USER_EMAIL = "makerspace@umass.edu"
const HOST = ":8000"

func main() {
	db, err := gorm.Open(mysql.Open(os.Getenv("DB")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&models.APIKey{})
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Training{})

	app := fiber.New()

	// Create a new user
	app.Post("/api/users", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.users:write") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type response struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		// Get the user's email and training type from the request body
		var body response
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if body.Email == "" || body.Name == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		// Check if the user already exists
		{
			var user models.User
			res := db.First(&user, "email = ?", body.Email)
			if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
				// The user already exists
				return c.Status(fiber.StatusConflict).SendString("User already exists")
			}
		}

		// Create a new user in the database
		user := models.User{
			Email:   body.Email,
			Name:    body.Name,
			Enabled: false,
		}
		db.Create(&user)

		// Write a success message to the response
		return c.SendString("User created successfully")
	}))

	// Search for a user
	app.Get("/api/users/search", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.users:search") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Query       string `query:"q"`
			Limit       int    `query:"limit"`
			Offset      int    `query:"offset"`
			OnlyEnabled bool   `query:"enabled"`
		}
		req := request{
			Limit:       10,
			Offset:      0,
			OnlyEnabled: true,
		}

		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if req.Query == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if req.Limit == 0 {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		// match the query against the name and email fields
		var users []models.User
		var count int64

		var searchQuery string
		if req.OnlyEnabled {
			searchQuery += "`enabled` = 1 AND "
		}
		searchQuery += "`name` LIKE @q OR `email` LIKE @q"

		db.Where(searchQuery, map[string]interface{}{"q": "%" + req.Query + "%"}).Offset(req.Offset).Limit(req.Limit).Find(&users)
		db.Model(&models.User{}).Where(searchQuery, map[string]interface{}{"q": "%" + req.Query + "%"}).Count(&count)

		type response struct {
			Count int64         `json:"count"`
			Users []models.User `json:"users"`
		}

		return c.JSON(response{
			Count: count,
			Users: users,
		})
	}))

	// Add completed training to a user
	app.Post("/api/training", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiUser := c.Locals(ctxUserKey{}).(models.User)
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.trainings:write") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type response struct {
			Email        string `json:"email"`
			TrainingType string `json:"training_type"`
		}
		// Get the user's email and training type from the request body
		var body response
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if body.TrainingType == "" && body.Email == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		// Check if the user exists
		var user models.User
		res := db.First(&user, "email = ?", body.Email)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		// Create a new training in the database
		training := models.Training{
			UserID:       user.ID,
			TrainingType: body.TrainingType,
			AddedBy:      apiUser.ID,
		}

		db.Create(&training)

		// If the user has completed the trainings "orientation" and "docusign", enable the user
		userTrainingEnable(db, user)

		// Write a success message to the response
		return c.SendString("Training added successfully")
	}))

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

func apiKeyAuthMiddleware(db *gorm.DB, next fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the API key from the request header
		apiKey := c.Get("API-Key")
		var apiKeyRecord models.APIKey
		apiKeyRecord.ID = apiKey
		res := db.First(&apiKeyRecord)

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
		c.Locals(ctxAPIKey{}, apiKeyRecord)
		return next(c)
	}
}
