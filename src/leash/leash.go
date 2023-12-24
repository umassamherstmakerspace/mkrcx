package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	leash_api "github.com/mkrcx/mkrcx/src/leash/api"
	leash_frontend "github.com/mkrcx/mkrcx/src/leash/frontend"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	models "github.com/mkrcx/mkrcx/src/shared/models"
)

const SYSTEM_USER_EMAIL = "makerspace@umass.edu"
const HOST = ":8000"

func main() {
	db, err := gorm.Open(mysql.Open(os.Getenv("DB")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	models.SetupValidator()

	// Migrate the schema
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.APIKey{})
	db.AutoMigrate(&models.Training{})
	db.AutoMigrate(&models.UserUpdate{})
	db.AutoMigrate(&models.Hold{})

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
	keys := leash_auth.InitalizeJWT()

	// Initalize RBAC
	enforcer := leash_auth.InitalizeCasbin(db)

	app.Use(leash_auth.LocalsMiddleware(db, keys, google, enforcer))

	frontend_dir := os.Getenv("FRONTEND_DIR")

	api := app.Group("/api")

	leash_api.RegisterAPIEndpoints(api)

	auth := api.Group("/auth")

	leash_auth.RegisterAuthenticationEndpoints(auth)

	leash_frontend.SetupFrontend(app, "/", frontend_dir)

	log.Printf("Starting server on port %s\n", HOST)
	app.Listen(HOST)
}
