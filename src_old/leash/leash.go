package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	val "github.com/go-playground/validator/v10/non-standard/validators"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	models "github.com/spectrum-control/spectrum/src/shared/models"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/webhook"
	"github.com/disgoorg/snowflake/v2"
)

type ctxAuthKey struct{}

type UserRole int

const (
	USER_ROLE_MEMBER UserRole = iota
	USER_ROLE_VOLUNTEER
	USER_ROLE_STAFF
	USER_ROLE_ADMIN
	USER_ROLE_SERVICE
)

func parseUserRole(role string) (UserRole, error) {
	switch role {
	case "member":
		return USER_ROLE_MEMBER, nil
	case "volunteer":
		return USER_ROLE_VOLUNTEER, nil
	case "staff":
		return USER_ROLE_STAFF, nil
	case "admin":
		return USER_ROLE_ADMIN, nil
	case "service":
		return USER_ROLE_SERVICE, nil
	default:
		return 0, errors.New("invalid role")
	}
}

func tryPath(file string, dir string) (string, error) {
	f := path.Join(dir, file)
	_, err := os.Stat(f)

	if err != nil {
		return "", err
	}

	return f, nil
}

type Authenticator int

const (
	AUTHENTICATOR_LOGGED_OUT Authenticator = iota
	AUTHENTICATOR_USER
	AUTHENTICATOR_APIKEY
)

type Authentication struct {
	User          models.User
	Authenticator Authenticator
	Data          interface{}
}

func (auth *Authentication) Authenticate(minimumRole UserRole, permissions ...string) error {
	switch auth.Authenticator {
	case AUTHENTICATOR_LOGGED_OUT:
		return errors.New("not logged in")
	case AUTHENTICATOR_USER:
		role, err := parseUserRole(auth.User.Role)
		if err != nil {
			return err
		}

		if role < minimumRole {
			return errors.New("insufficient permissions")
		}
	case AUTHENTICATOR_APIKEY:
		apiKey := auth.Data.(models.APIKey)
		for _, permission := range permissions {
			if !models.APIKeyValidate(apiKey, permission) {
				return errors.New("insufficient permissions")
			}
		}
	}
	return nil
}

type UserIDReq struct {
	ID    uint   `json:"id" xml:"id" form:"id" query:"id" validate:"required_without=Email"`
	Email string `json:"email" xml:"email" form:"email" query:"email" validate:"required_without=ID"`
}

const SYSTEM_USER_EMAIL = "makerspace@umass.edu"
const HOST = ":8000"

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

func main() {
	db, err := gorm.Open(mysql.Open(os.Getenv("DB")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	err = validate.RegisterValidation("notblank", val.NotBlank)
	if err != nil {
		panic(err)
	}

	// Migrate the schema
	db.AutoMigrate(&models.APIKey{})
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Training{})
	db.AutoMigrate(&models.UserUpdate{})

	app := fiber.New()

	app.Use(cors.New())

	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return os.Getenv("ENVIRONMENT") == "development"
		},
	}))

	URL := os.Getenv("URL")

	// Google OAuth2
	google := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  URL + "/auth/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}

	// JWT Key

	//read text from file keyfile
	key_file := os.Getenv("KEY_FILE")
	if _, err := os.Stat(key_file); os.IsNotExist(err) {
		raw, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			log.Fatal(err)
		}

		key, err := jwk.FromRaw(raw)
		if err != nil {
			log.Fatal(err)
		}

		key.Set(jwk.KeyIDKey, "sig-"+strconv.FormatInt(time.Now().Unix(), 10))
		key.Set(jwk.AlgorithmKey, jwa.RS256)
		key.Set(jwk.KeyUsageKey, jwk.ForSignature)

		keys := jwk.NewSet()
		keys.AddKey(key)

		buf, err := json.MarshalIndent(keys, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		err = os.WriteFile(key_file, buf, 0600)
		if err != nil {
			log.Fatal(err)
		}
	}

	keyFile, err := os.Open(key_file)
	if err != nil {
		log.Fatal(err)
	}

	defer keyFile.Close()

	keyBytes, err := io.ReadAll(keyFile)
	if err != nil {
		log.Fatal(err)
	}

	keys, err := jwk.Parse(keyBytes)
	if err != nil {
		fmt.Printf("failed to parse private key: %s\n", err)
	}

	privateKey, _ := keys.Key(0)

	publicKey, err := privateKey.PublicKey()
	if err != nil {
		fmt.Printf("failed to get public key: %s\n", err)
	}

	// Discord Webhook
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	var webhookClient webhook.Client
	if webhookURL != "" {
		webhookClient, err = webhook.NewWithURL(webhookURL)
		if err != nil {
			fmt.Printf("failed to create webhook: %s\n", err)
		}
	}

	frontend_dir := os.Getenv("FRONTEND_DIR")

	// Create a new user
	app.Post("/api/users", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)

		if authentication.Authenticate(USER_ROLE_STAFF, "leash.users:write") != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Email    string `json:"email" xml:"email" form:"email" validate:"required,email"`
			Name     string `json:"name" xml:"name" form:"name" validate:"required"`
			Role     string `json:"role" xml:"role" form:"role" validate:"required,oneof=member volunteer staff admin"`
			Type     string `json:"type" xml:"type" form:"type" validate:"required,oneof=undergrad grad faculty staff alumni other"`
			GradYear int    `json:"grad_year" xml:"grad_year" form:"grad_year" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
			Major    string `json:"major" xml:"major" form:"major" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
		}
		// Get the user's email and training type from the request body
		var req request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		// Check if the user already exists
		{
			var user models.User
			res := db.Find(&user, "email = ?", req.Email)
			if res.RowsAffected > 0 {
				// The user already exists
				return c.Status(fiber.StatusConflict).SendString("User already exists")
			}
		}

		// Create a new user in the database
		user := models.User{
			Email:          req.Email,
			Name:           req.Name,
			Role:           req.Role,
			Type:           req.Type,
			GraduationYear: req.GradYear,
			Major:          req.Major,
			Enabled:        false,
		}
		db.Create(&user)

		// Send a discord webhook
		if webhookClient != nil {
			embed := discord.NewEmbedBuilder().
				SetTitle("New User").
				SetDescription("A new user has been created").
				SetColor(0x00ff00).
				AddField("Name", req.Name, true).
				AddField("Email", req.Email, true).
				SetTimestamp(time.Now()).
				Build()

			_, err := webhookClient.CreateEmbeds([]discord.Embed{embed})
			if err != nil {
				fmt.Printf("failed to send webhook: %s\n", err)
			}
		}

		// Write a success message to the response
		return c.SendString("User created successfully")
	}))

	// Update a user
	app.Put("/api/users", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)
		apiUser := authentication.User

		if authentication.Authenticate(USER_ROLE_STAFF, "leash.users:write") != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			UserIDReq
			Name     *string `json:"name" xml:"name" form:"name" validate:"omitempty"`
			NewEmail *string `json:"new_email" xml:"new_email" form:"new_email" validate:"omitempty,email"`
			Enabled  *bool   `json:"enabled" xml:"enabled" form:"enabled" validate:"omitempty"`
			Role     *string `json:"role" xml:"role" form:"role" validate:"omitempty,oneof=member volunteer staff admin"`
			Type     *string `json:"type" xml:"type" form:"type" validate:"omitempty,oneof=undergrad grad faculty staff alumni other"`
			GradYear *int    `json:"grad_year" xml:"grad_year" form:"grad_year" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
			Major    *string `json:"major" xml:"major" form:"major" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
		}

		// Get the user's email and training type from the request body
		var req request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		// Check if the user exists
		var user models.User
		res := db.Model(&models.User{}).Where("email = ?", req.Email).Or("id = ?", req.ID).First(&user)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		if req.Role != nil {
			if authentication.Authenticate(USER_ROLE_ADMIN, "leash.users:write") != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
			}
		}

		// Update the user in the database
		if req.NewEmail != nil {
			updateUser(db, user, apiUser, "email", user.Email, *req.NewEmail, true)
			user.Email = *req.NewEmail
		}
		if req.Name != nil {
			updateUser(db, user, apiUser, "name", user.Name, *req.Name, true)
			user.Name = *req.Name
		}
		if req.Role != nil {
			updateUser(db, user, apiUser, "role", user.Role, *req.Role, true)
			user.Role = *req.Role
		}
		if req.Type != nil {
			updateUser(db, user, apiUser, "type", user.Type, *req.Type, true)
			user.Type = *req.Type
		}
		if req.GradYear != nil {
			updateUser(db, user, apiUser, "graduation_year", strconv.Itoa(user.GraduationYear), strconv.Itoa(*req.GradYear), true)
			user.GraduationYear = *req.GradYear
		}
		if req.Major != nil {
			updateUser(db, user, apiUser, "major", user.Major, *req.Major, true)
			user.Major = *req.Major
		}

		res = db.Save(&user)
		if res.Error != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Database error")
		}

		// Write a success message to the response
		return c.SendString("User updated successfully")
	}))

	// Search for a user
	app.Get("/api/users/search", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)

		if authentication.Authenticate(USER_ROLE_STAFF, "leash.users:search") != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			Query          string `query:"q" validate:"required_without=allow_empty_body,required_unless=allow_empty_body true"`
			Limit          int    `query:"limit" validate:"required,min=1,max=1000"`
			Offset         int    `query:"offset" validate:"required,min=0"`
			OnlyEnabled    bool   `query:"only_enabled"`
			AllowEmptyBody bool   `query:"allow_empty_body" validate:"required_without=query"`
			WithTrainings  bool   `query:"with_trainings"`
			WithUpdates    bool   `query:"with_updates"`
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

		con := db.Model(&models.User{})

		if req.WithTrainings {
			if authentication.Authenticate(USER_ROLE_STAFF, "leash.trainings:read") != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
			}
			con = con.Preload("Trainings")
		}

		if req.WithUpdates {
			if authentication.Authenticate(USER_ROLE_STAFF, "leash.updates:read") != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
			}
			con = con.Preload("UserUpdates")
		}

		// match the query against the name and email fields
		var users []models.User
		var count int64

		var searchQuery string
		if req.OnlyEnabled {
			searchQuery += "`enabled` = 1 AND "
		}
		searchQuery += "(`name` LIKE @q) OR (`email` LIKE @q)"

		con = con.Where(searchQuery, map[string]interface{}{"q": "%" + req.Query + "%"})
		con.Offset(req.Offset).Limit(req.Limit).Find(&users)
		con.Count(&count)

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
	app.Get("/api/users/", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)

		if authentication.Authenticate(USER_ROLE_STAFF, "leash.users:read") != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			UserIDReq
			WithTrainings bool `query:"with_trainings"`
			WithUpdates   bool `query:"with_updates"`
		}
		// Get the user's email from the request body
		var req request
		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		con := db.Model(&models.User{})

		if req.WithTrainings {
			if authentication.Authenticate(USER_ROLE_STAFF, "leash.trainings:read") != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
			}
			con = con.Preload("Trainings")
		}

		if req.WithUpdates {
			if authentication.Authenticate(USER_ROLE_STAFF, "leash.updates:read") != nil {
				return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
			}
			con = con.Preload("UserUpdates")
		}

		if req.Email != "" {
			con = con.Where("email = ?", req.Email)
		} else {
			con = con.Where("id = ?", req.ID)
		}

		// Check if the user exists
		var user models.User
		res := con.First(&user)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		return c.JSON(user)
	}))

	// Get a user from their email or id
	app.Get("/api/users/self", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)

		type request struct {
			WithTrainings bool `query:"with_trainings"`
		}
		// Get the user's email from the request body
		var req request
		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		con := db.Model(&models.User{})

		if req.WithTrainings {
			con = con.Preload("Trainings")
		}

		con = con.Where("id = ?", authentication.User.ID)

		// Check if the user exists
		var user models.User
		res := con.First(&user)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		return c.JSON(user)
	}))

	// Get a user from their email or id
	app.Put("/api/users/self", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)
		user := authentication.User

		type request struct {
			Name     *string `json:"name" xml:"name" form:"name" validate:"omitempty"`
			GradYear *int    `json:"grad_year" xml:"grad_year" form:"grad_year" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
			Major    *string `json:"major" xml:"major" form:"major" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
		}

		// Get the user's email and training type from the request body
		var req request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		// Update the user in the database
		if req.Name != nil {
			updateUser(db, user, user, "name", user.Name, *req.Name, true)
			user.Name = *req.Name
		}
		if req.GradYear != nil {
			updateUser(db, user, user, "graduation_year", strconv.Itoa(user.GraduationYear), strconv.Itoa(*req.GradYear), true)
			user.GraduationYear = *req.GradYear
		}
		if req.Major != nil {
			updateUser(db, user, user, "major", user.Major, *req.Major, true)
			user.Major = *req.Major
		}

		res := db.Save(&user)
		if res.Error != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Database error")
		}

		// Write a success message to the response
		return c.SendString("User updated successfully")
	}))

	// Add completed training to a user
	app.Post("/api/training", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)
		apiUser := authentication.User

		if authentication.Authenticate(USER_ROLE_STAFF, "leash.trainings:write") != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			UserIDReq
			TrainingType string `json:"training_type" xml:"training_type" form:"training_type" validate:"required,notblank,lowercase"`
		}
		// Get the user's email and training type from the request body
		var req request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		con := db.Model(&models.User{})

		if req.Email != "" {
			con = con.Where("email = ?", req.Email)
		} else {
			con = con.Where("id = ?", req.ID)
		}

		// Check if the user exists
		var user models.User
		res := con.First(&user)
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
		userTrainingEnable(db, user, webhookClient, URL, privateKey)

		// Write a success message to the response
		return c.SendString("Training added successfully")
	}))

	// Delete a training from a user
	app.Delete("/api/training", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)
		apiUser := authentication.User

		if authentication.Authenticate(USER_ROLE_STAFF, "leash.trainings:write") != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			UserIDReq
			TrainingType string `json:"training_type" xml:"training_type" form:"training_type" validate:"required,notblank,lowercase"`
		}
		// Get the user's email and training type from the request body
		var req request
		if err := c.BodyParser(&req); err != nil {
			fmt.Println(err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		con := db.Model(&models.User{})

		if req.Email != "" {
			con = con.Where("email = ?", req.Email)
		} else {
			con = con.Where("id = ?", req.ID)
		}

		// Check if the user exists
		var user models.User
		res := con.First(&user)
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
	app.Get("/api/training", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)

		if authentication.Authenticate(USER_ROLE_STAFF, "leash.trainings:read") != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			UserIDReq
			IncludeDeleted bool `query:"include_deleted"`
		}
		// Get the user's email from the request body
		var req request
		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		con := db.Model(&models.User{})

		if req.Email != "" {
			con = con.Where("email = ?", req.Email)
		} else {
			con = con.Where("id = ?", req.ID)
		}

		// Check if the user exists
		var user models.User
		res := con.First(&user)
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

	// Get a user's updates
	app.Get("/api/updates", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		// Get api user from the request context
		authentication := c.Locals(ctxAuthKey{}).(Authentication)

		if authentication.Authenticate(USER_ROLE_STAFF, "leash.updates:read") != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		type request struct {
			UserIDReq
		}
		// Get the user's email from the request body
		var req request
		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		con := db.Model(&models.User{})
		if req.Email != "" {
			con = con.Where("email = ?", req.Email)
		} else {
			con = con.Where("id = ?", req.ID)
		}

		// Check if the user exists
		var user models.User
		res := con.First(&user)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		var updates []models.UserUpdate
		db.Model(&models.UserUpdate{}).Where("user_id = ?", user.ID).Find(&updates)

		// Write the updates to the response
		return c.JSON(updates)
	}))

	app.Get("/auth/login", func(c *fiber.Ctx) error {
		var req struct {
			Return string `query:"return"`
		}

		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		if req.Return == "" {
			req.Return = "/"
		}

		tok, err := jwt.NewBuilder().
			Issuer(`github.com/lestrrat-go/jwx`).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(5*time.Minute)).
			Claim("return", req.Return).
			Build()

		if err != nil {
			fmt.Printf("failed to build token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, privateKey))
		if err != nil {
			fmt.Printf("failed to sign token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		url := google.AuthCodeURL(string(signed))
		return c.Redirect(url)
	})

	// Login Flow
	app.Get("/auth/callback", func(c *fiber.Ctx) error {
		var req struct {
			Code  string `query:"code" validate:"required"`
			State string `query:"state" validate:"required"`
		}

		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		ret := "/"
		{
			tok, err := jwt.ParseString(req.State, jwt.WithKey(jwa.RS256, publicKey))
			if err != nil {
				fmt.Printf("failed to parse token: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid state")
			}

			if err := jwt.Validate(tok); err != nil {
				fmt.Printf("failed to validate token: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid state")
			}

			val, valid := tok.Get("return")
			if !valid {
				fmt.Printf("failed to get return value: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid state")
			}

			ret = val.(string)
		}

		userinfo := &struct {
			Email string `json:"email" validate:"required,email"`
		}{}

		{
			tok, err := google.Exchange(c.Context(), req.Code)
			if err != nil {
				fmt.Printf("failed to exchange token: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid code")
			}

			client := google.Client(c.Context(), tok)
			resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
			if err != nil {
				fmt.Printf("failed to get userinfo: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid code")
			}
			defer resp.Body.Close()

			err = json.NewDecoder(resp.Body).Decode(userinfo)
			if err != nil {
				fmt.Printf("failed to decode userinfo: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid code")
			}

			{
				errors := ValidateStruct(userinfo)
				if errors != nil {
					return c.Status(fiber.StatusBadRequest).JSON(errors)
				}
			}
		}

		var user models.User
		res := db.First(&user, "email = ?", userinfo.Email)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			c.Set("Content-Type", "text/html")
			return c.Status(fiber.StatusUnauthorized).SendString(
				fmt.Sprintf(`
				<html>
					<head>
						<title>Unauthorized</title>
					</head>

					<body>
						<h1>Unauthorized</h1>
						<br>
						<p>You need to create an account before you can log in.</p>
						<br>
						<p>If you already have an account, please log in with the email you used to create your account.</p>
						<br>
						<a href="/auth/login?return=%s">Retry Login</a>
					</body>
				</html>
			`, ret))
		}

		if !user.Enabled {
			// The user is not enabled
			c.Set("Content-Type", "text/html")
			return c.Status(fiber.StatusUnauthorized).SendString(
				fmt.Sprintf(`
				<html>
					<head>
						<title>Unauthorized</title>
					</head>

					<body>
						<h1>Unauthorized</h1>
						<br>
						<p>Your account is not enabled. Please sign the docusign form and finish the orientation or contact an administrator to enable your account.</p>
						<br>
						<a href="/auth/login?return=%s">Retry Login</a>
					</body>
				</html>
			`, ret))
		}

		if user.Role == "service" {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		tok, err := jwt.NewBuilder().
			Issuer(`github.com/lestrrat-go/jwx`).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(24*time.Hour)).
			Claim("email", userinfo.Email).
			Build()
		if err != nil {
			fmt.Printf("failed to build token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, privateKey))
		if err != nil {
			fmt.Printf("failed to sign token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		cookie := new(fiber.Cookie)
		cookie.Name = "token"
		cookie.Value = string(signed)
		cookie.Expires = tok.Expiration()

		c.Cookie(cookie)
		return c.Redirect(ret)
	})

	app.Get("/auth/validate", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		authentication := c.Locals(ctxAuthKey{}).(Authentication)

		if authentication.Authenticator != AUTHENTICATOR_USER {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		return c.SendString("Authorized")
	}))

	app.Get("/auth/logout", func(c *fiber.Ctx) error {
		c.ClearCookie("token")
		return c.Redirect("/")
	})

	app.Get("/auth/refresh", authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		authentication := c.Locals(ctxAuthKey{}).(Authentication)

		if authentication.Authenticator != AUTHENTICATOR_USER {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		tok, err := jwt.NewBuilder().
			Issuer(`github.com/lestrrat-go/jwx`).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(24*time.Hour)).
			Claim("email", authentication.User.Email).
			Build()

		if err != nil {
			fmt.Printf("failed to build token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, privateKey))
		if err != nil {
			fmt.Printf("failed to sign token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(struct {
			Token     string    `json:"token"`
			ExpiresAt time.Time `json:"expires_at"`
		}{
			Token:     string(signed),
			ExpiresAt: tok.Expiration(),
		})
	}))

	app.Get("/discord/enable", cookieAuthMiddleware(publicKey, authMiddleware(db, publicKey, func(c *fiber.Ctx) error {
		authentication := c.Locals(ctxAuthKey{}).(Authentication)

		if authentication.Authenticate(USER_ROLE_STAFF, "leash.users:write") != nil {
			return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
		}

		var req struct {
			Token string `query:"token" validate:"required"`
		}

		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		var user_id int
		var message_id snowflake.ID
		{
			tok, err := jwt.ParseString(req.Token, jwt.WithKey(jwa.RS256, publicKey))
			if err != nil {
				fmt.Printf("failed to parse token: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
			}

			if err := jwt.Validate(tok); err != nil {
				fmt.Printf("failed to validate token: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
			}

			val, valid := tok.Get("user_id")
			if !valid {
				fmt.Printf("failed to get id value: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
			}

			user_id, err = strconv.Atoi(fmt.Sprintf("%v", val))
			if err != nil {
				fmt.Printf("failed to convert id value: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
			}

			val, valid = tok.Get("message_id")
			if !valid {
				fmt.Printf("failed to get message id value: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
			}

			message_id, err = snowflake.Parse(fmt.Sprintf("%v", val))
			if err != nil {
				fmt.Printf("failed to convert message id value: %s\n", err)
				return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
			}
		}

		var user models.User
		res := db.First(&user, "id = ?", user_id)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			return c.Status(fiber.StatusBadRequest).SendString("User not found")
		}

		user.Enabled = true
		db.Save(&user)

		// Create a new update in the database
		update := models.UserUpdate{
			UserID:   user.ID,
			EditedBy: authentication.User.ID,
			Field:    "enabled",
			OldValue: "false",
			NewValue: "true",
		}

		db.Create(&update)

		// Send a discord webhook
		if webhookClient != nil {
			embed := discord.NewEmbedBuilder().
				SetTitle("User Enabled").
				SetDescription("User has been enabled.").
				SetColor(0xff00B0).
				AddField("Name", user.Name, true).
				AddField("Email", user.Email, true).
				AddField("Enabled By", authentication.User.Name, false).
				SetTimestamp(time.Now()).
				Build()

			_, err := webhookClient.UpdateEmbeds(message_id, []discord.Embed{embed})
			if err != nil {
				fmt.Printf("failed to send webhook: %s\n", err)
			}
		}

		return c.Redirect("/")
	})))

	app.Use("/", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api") {
			return c.Next()
		}

		if strings.HasPrefix(c.Path(), "/auth") {
			return c.Next()
		}

		if strings.HasPrefix(c.Path(), "/discord") {
			return c.Next()
		}

		// path := c.Path()
		request := path.Clean(c.Path())
		if request != c.Path() {
			return c.Redirect(request, fiber.StatusMovedPermanently)
		}

		if path.Ext(path.Base(c.Path())) == "" {
			file, err := tryPath(c.Path()+".html", frontend_dir)
			if err == nil {
				return c.SendFile(file)
			}

			file, err = tryPath(path.Join(c.Path(), "index.html"), frontend_dir)
			if err == nil {
				return c.SendFile(file)
			}
		} else {
			file, err := tryPath(c.Path(), frontend_dir)
			if err == nil {
				return c.SendFile(file)
			}
		}

		return c.SendStatus(fiber.StatusNotFound)
	})

	log.Printf("Starting server on port %s\n", HOST)
	app.Listen(HOST)
}

func userTrainingEnable(db *gorm.DB, user models.User, webhookClient webhook.Client, URL string, privateKey jwk.Key) {
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
		// Send a discord webhook
		if webhookClient != nil {
			message, err := webhookClient.CreateContent("Awaiting Token Generation")
			if err != nil {
				fmt.Printf("failed to send webhook: %s\n", err)
			}

			fmt.Println(message)

			token, err := jwt.NewBuilder().
				Issuer(`github.com/lestrrat-go/jwx`).
				IssuedAt(time.Now()).
				Claim("user_id", user.ID).
				Claim("message_id", message.ID).
				Build()

			if err != nil {
				fmt.Printf("failed to build token: %s\n", err)
				return
			}

			signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, privateKey))
			if err != nil {
				fmt.Printf("failed to sign token: %s\n", err)
				return
			}

			embed := discord.NewEmbedBuilder().
				SetTitle("User Awaiting Verification").
				SetDescription("A user has completed the orientation and docusign trainings and is awaiting verification.").
				SetColor(0xffa000).
				AddField("Name", user.Name, true).
				AddField("Email", user.Email, true).
				AddField("Verification Link", fmt.Sprintf(URL+"/discord/enable?token=%s", signed), false).
				SetTimestamp(time.Now()).
				Build()

			m := ""
			_, err = webhookClient.UpdateMessage(message.ID, discord.WebhookMessageUpdate{
				Embeds:  &[]discord.Embed{embed},
				Content: &m,
			})

			if err != nil {
				fmt.Printf("failed to update webhook: %s\n", err)
			}
		}
	}

}

func validateToken(token string, publicKey jwk.Key) bool {
	tok, err := jwt.ParseString(token, jwt.WithKey(jwa.RS256, publicKey))
	if err != nil {
		return false
	}

	if err := jwt.Validate(tok); err != nil {
		return false
	}

	_, v := tok.Get("email")
	return v
}

func cookieAuthMiddleware(publicKey jwk.Key, next fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the token from the cookie
		cookie := c.Cookies("token")

		if !validateToken(cookie, publicKey) {
			return c.Redirect("/auth/login?return=" + c.OriginalURL())
		}

		cookie = c.Cookies("token")
		c.Locals("Authorization", "Bearer "+cookie)

		return next(c)
	}
}

func authMiddleware(db *gorm.DB, publicKey jwk.Key, next fiber.Handler) fiber.Handler {
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

		// Get the token from the request header
		authentication, err := func() (Authentication, error) {
			authentication := Authentication{
				Authenticator: AUTHENTICATOR_LOGGED_OUT,
			}

			authLocal := c.Locals("Authorization")

			var authorization string
			if authLocal == nil {
				authorization = c.Get("Authorization")
			} else {
				authorization = authLocal.(string)
			}

			if authorization == "" {
				return authentication, errors.New("no authorization header")
			}

			// Get the token from the authorization header
			token := strings.TrimPrefix(authorization, "Bearer ")

			// Parse the token
			tok, err := jwt.ParseString(token, jwt.WithKey(jwa.RS256, publicKey))
			if err != nil {
				return authentication, err
			}

			// Validate the token
			if err := jwt.Validate(tok); err != nil {
				return authentication, err
			}

			// Get the email from the token
			email, valid := tok.Get("email")
			if !valid {
				return authentication, errors.New("token does not contain email")
			}

			// Check if the user exists
			var user models.User
			res := db.First(&user, "email = ?", email)
			if errors.Is(res.Error, gorm.ErrRecordNotFound) {
				// The user does not exist
				return authentication, errors.New("user not found")
			}

			if !user.Enabled {
				// The user is not enabled
				return authentication, errors.New("user not enabled")
			}

			if user.Role == "service" {
				return authentication, errors.New("service account")
			}

			authentication = Authentication{
				Authenticator: AUTHENTICATOR_USER,
				User:          user,
			}

			return authentication, nil
		}()

		if err != nil {
			// Get the api key from the request header
			authentication, err = func() (Authentication, error) {
				authentication := Authentication{
					Authenticator: AUTHENTICATOR_LOGGED_OUT,
				}

				apiKey := c.Get("API-Key")
				if apiKey == "" {
					return authentication, errors.New("no API-Key header")
				}

				var apiKeyRecord = models.APIKey{ID: apiKey}

				res := db.First(&apiKeyRecord)
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					// The API key is not valid
					return authentication, errors.New("invalid API key")
				}

				fmt.Println(apiKeyRecord.ID)

				var user models.User
				res = db.First(&user, "id = ?", apiKeyRecord.UserID)
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					// The user does not exist
					return authentication, errors.New("user not found")
				}

				authentication = Authentication{
					Authenticator: AUTHENTICATOR_APIKEY,
					User:          user,
					Data:          apiKeyRecord,
				}

				return authentication, nil
			}()

			if err != nil {
				authentication = Authentication{
					Authenticator: AUTHENTICATOR_LOGGED_OUT,
				}
			}
		}

		c.Locals(ctxAuthKey{}, authentication)
		return next(c)
	}
}

func updateUser(db *gorm.DB, user models.User, editedBy models.User, field string, oldValue string, newValue string, accepted bool) {
	update := models.UserUpdate{
		Field:    field,
		NewValue: newValue,
		OldValue: oldValue,
		UserID:   user.ID,
		EditedBy: editedBy.ID,
	}

	db.Create(&update)
}
