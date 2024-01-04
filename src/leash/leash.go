package main

import (
	"log"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	leash_api "github.com/mkrcx/mkrcx/src/leash/api"
	leash_frontend "github.com/mkrcx/mkrcx/src/leash/frontend"
	leash_helpers "github.com/mkrcx/mkrcx/src/leash/helpers"
	leash_signin "github.com/mkrcx/mkrcx/src/leash/signin"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	models "github.com/mkrcx/mkrcx/src/shared/models"
)

const DEFAULT_HOST = ":8000"

func main() {
	db, err := gorm.Open(mysql.Open(os.Getenv("DB")), &gorm.Config{})
	if err != nil {
		log.Panicln("failed to connect database")
	}

	models.SetupValidator()

	// Migrate the schema
	log.Println("Migrating database schema...")
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.APIKey{})
	db.AutoMigrate(&models.Training{})
	db.AutoMigrate(&models.UserUpdate{})
	db.AutoMigrate(&models.Hold{})
	db.AutoMigrate(&models.Session{})

	// Google OAuth2
	google := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("FRONTEND_URL") + "/auth/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}

	// JWT Key
	log.Println("Initalizing JWT Keys...")
	key_file := os.Getenv("KEY_FILE")
	keys, err := leash_auth.InitalizeJWT(key_file)
	if err != nil {
		log.Panicln(err)
	}

	// Initalize RBAC
	log.Println("Initalizing RBAC...")
	enforcer, err := leash_auth.InitalizeCasbin(db)
	if err != nil {
		log.Panicln(err)
	}

	models.SetupEnforcer(enforcer)

	leash_helpers.SetupCasbin(enforcer)

	log.Println("Migrating User Roles...")
	err = leash_helpers.MigrateUserRoles(db, enforcer)
	if err != nil {
		log.Panicln(err)
	}

	log.Println("Migrating API Key Access...")
	err = leash_helpers.MigrateAPIKeyAccess(db, enforcer)
	if err != nil {
		log.Panicln(err)
	}

	// Create App
	log.Println("Initalizing Fiber...")
	host := os.Getenv("HOST")
	if host == "" {
		host = DEFAULT_HOST
	}

	app := fiber.New()

	// Use CORS
	app.Use(cors.New())

	// Allow all origins in development
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return os.Getenv("ENVIRONMENT") == "development"
		},
	}))

	app.Use(leash_auth.LocalsMiddleware(db, keys, google, enforcer))

	frontend_dir := os.Getenv("FRONTEND_DIR")

	log.Println("Setting up routes...")

	api := app.Group("/api", leash_auth.SetPermissionPrefixMiddleware("leash"))

	leash_api.RegisterAPIEndpoints(api)

	auth := app.Group("/auth")

	leash_signin.RegisterAuthenticationEndpoints(auth)

	leash_frontend.SetupFrontend(app, "/", frontend_dir)

	log.Printf("Starting server on port %s\n", host)
	app.Listen(host)
}
