package main

import (
	"errors"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

	app.Use(cors.New())

	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return os.Getenv("ENVIRONMENT") == "development"
		},
	}))

	// Create a new user
	app.Post("/api/users", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.users:write") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Email     string `json:"email" xml:"email" form:"email"`
			FirstName string `json:"first_name" xml:"first_name" form:"first_name"`
			LastName  string `json:"last_name" xml:"last_name" form:"last_name"`
			Role      string `json:"role" xml:"role" form:"role"`
			Type      string `json:"type" xml:"type" form:"type"`
			GradYear  int    `json:"grad_year" xml:"grad_year" form:"grad_year"`
			Major     string `json:"major" xml:"major" form:"major"`
		}
		// Get the user's email and training type from the request body
		var req request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if req.Email == "" || req.FirstName == "" || req.LastName == "" || req.Type == "" || req.GradYear == 0 || req.Major == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		// Check if the user already exists
		{
			var user models.User
			res := db.First(&user, "email = ?", req.Email)
			if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
				// The user already exists
				return c.Status(fiber.StatusConflict).SendString("User already exists")
			}
		}

		// Create a new user in the database
		user := models.User{
			Email:          req.Email,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			Role:           req.Role,
			Type:           req.Type,
			GraduationYear: req.GradYear,
			Major:          req.Major,
			Enabled:        false,
		}
		db.Create(&user)

		// Write a success message to the response
		return c.SendString("User created successfully")
	}))

	// Update a user
	app.Put("/api/users", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.users:write") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Email     string `json:"email" xml:"email" form:"email"`
			ID        uint   `json:"id" xml:"id" form:"id"`
			NewEmail  string `json:"new_email" xml:"new_email" form:"new_email"`
			FirstName string `json:"first_name" xml:"first_name" form:"first_name"`
			LastName  string `json:"last_name" xml:"last_name" form:"last_name"`
			Role      string `json:"role" xml:"role" form:"role"`
			Type      string `json:"type" xml:"type" form:"type"`
			GradYear  int    `json:"grad_year" xml:"grad_year" form:"grad_year"`
			Major     string `json:"major" xml:"major" form:"major"`
		}

		// Get the user's email and training type from the request body
		var req request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if (req.Email == "" && req.ID == 0) || (req.Email != "" && req.ID != 0) {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		// Check if the user exists
		var user models.User
		res := db.Model(&models.User{}).Where("email = ?", req.Email).Or("id = ?", req.ID).First(&user)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		// Update the user in the database
		if req.NewEmail != "" {
			user.Email = req.NewEmail
		}
		if req.FirstName != "" {
			user.FirstName = req.FirstName
		}
		if req.LastName != "" {
			user.LastName = req.LastName
		}
		if req.Role != "" {
			user.Role = req.Role
		}
		if req.Type != "" {
			user.Type = req.Type
		}
		if req.GradYear != 0 {
			user.GraduationYear = req.GradYear
		}
		if req.Major != "" {
			user.Major = req.Major
		}

		res = db.Save(&user)
		if res.Error != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Database error")
		}

		// Write a success message to the response
		return c.SendString("User updated successfully")
	}))

	// Search for a user
	app.Get("/api/users/search", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.users:search") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Query          string `query:"q"`
			Limit          int    `query:"limit"`
			Offset         int    `query:"offset"`
			OnlyEnabled    bool   `query:"enabled"`
			AllowEmptyBody bool   `query:"allow_empty_body"`
		}
		req := request{
			Limit:       10,
			Offset:      0,
			OnlyEnabled: true,
		}

		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if req.Query == "" && !req.AllowEmptyBody {
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
		searchQuery += "((CONCAT_WS(\" \", `first_name`, `last_name`) LIKE @q) OR (`email` LIKE @q))"

		db.Model(&models.User{}).Preload("Trainings").Where(searchQuery, map[string]interface{}{"q": "%" + req.Query + "%"}).Offset(req.Offset).Limit(req.Limit).Find(&users)

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

	// Get a user from their email or id
	app.Get("/api/users/", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.users:read") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Email string `query:"email"`
			ID    uint   `query:"id"`
		}
		// Get the user's email from the request body
		var req request
		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if (req.Email == "" && req.ID == 0) || (req.Email != "" && req.ID != 0) {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		// Check if the user exists
		var user models.User
		res := db.Model(&models.User{}).Preload("Trainings").Where("email = ?", req.Email).Or("id = ?", req.ID).First(&user)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		return c.JSON(user)
	}))

	// Add completed training to a user
	app.Post("/api/training", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiUser := c.Locals(ctxUserKey{}).(models.User)
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.trainings:write") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Email        string `json:"email" xml:"email" form:"email"`
			ID           uint   `json:"id" xml:"id" form:"id"`
			TrainingType string `json:"training_type" xml:"training_type" form:"training_type"`
		}
		// Get the user's email and training type from the request body
		var req request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if req.TrainingType == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if (req.Email == "" && req.ID == 0) || (req.Email != "" && req.ID != 0) {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		// Check if the user exists
		var user models.User
		res := db.Model(&models.User{}).Where("email = ?", req.Email).Or("id = ?", req.ID).First(&user)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		// Check if the training already exists
		var training models.Training
		res = db.Model(&models.Training{}).Where("user_id = ? AND training_type = ?", user.ID, req.TrainingType).First(&training)
		if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The training already exists
			return c.Status(fiber.StatusConflict).SendString("Training already exists")
		}

		// Create a new training in the database
		training = models.Training{
			UserID:       user.ID,
			TrainingType: req.TrainingType,
			AddedBy:      apiUser.ID,
		}

		db.Create(&training)

		// If the user has completed the trainings "orientation" and "docusign", enable the user
		userTrainingEnable(db, user)

		// Write a success message to the response
		return c.SendString("Training added successfully")
	}))

	// Delete a training from a user
	app.Delete("/api/training", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiUser := c.Locals(ctxUserKey{}).(models.User)
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.trainings:write") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Email        string `json:"email" xml:"email" form:"email"`
			ID           uint   `json:"id" xml:"id" form:"id"`
			TrainingType string `json:"training_type" xml:"training_type" form:"training_type"`
		}
		// Get the user's email and training type from the request body
		var req request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if req.TrainingType == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if (req.Email == "" && req.ID == 0) || (req.Email != "" && req.ID != 0) {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		// Check if the user exists
		var user models.User
		res := db.Model(&models.User{}).Where("email = ?", req.Email).Or("id = ?", req.ID).First(&user)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		// Check if the training exists
		var training models.Training
		res = db.Model(&models.Training{}).Where("user_id = ? AND training_type = ?", user.ID, req.TrainingType).First(&training)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The training does not exist
			return c.Status(fiber.StatusConflict).SendString("Training has not been added")
		}

		// Update the training in the database
		training.RemovedBy = apiUser.ID
		db.Save(&training)
		// Delete the training from the db
		db.Delete(&training)

		// Write a success message to the response
		return c.SendString("Training removed successfully")
	}))

	// Get a user's trainings
	app.Get("/api/training", apiKeyAuthMiddleware(db, func(c *fiber.Ctx) error {
		// Get api user from the request context
		apiKey := c.Locals(ctxAPIKey{}).(models.APIKey)

		if !models.APIKeyValidate(apiKey, "leash.trainings:read") {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Email          string `query:"email"`
			ID             uint   `query:"id"`
			IncludeDeleted bool   `query:"include_deleted"`
		}
		// Get the user's email from the request body
		var req request
		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if (req.Email == "" && req.ID == 0) || (req.Email != "" && req.ID != 0) {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		// Check if the user exists
		var user models.User
		res := db.Model(&models.User{}).Where("email = ?", req.Email).Or("id = ?", req.ID).First(&user)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		var trainings []models.Training
		if req.IncludeDeleted {
			db.Model(&models.Training{}).Unscoped().Where("user_id = ?", user.ID).Find(&trainings)
		} else {
			db.Model(&models.Training{}).Where("user_id = ?", user.ID).Find(&trainings)
		}

		// Write the trainings to the response
		return c.JSON(trainings)
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
		// Make sure DB is alive
		sql, err := db.DB()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Database connection error")
		}
		err = sql.Ping()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Database connection error")
		}

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
