package main

import (
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	leash_helpers "github.com/mkrcx/mkrcx/src/leash/helpers"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
)

const DEFAULT_HOST = ":8000"

func main() {
	// dotenv Setup
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize DB
	db, err := gorm.Open(mysql.Open(os.Getenv("DB")), &gorm.Config{})
	if err != nil {
		log.Panicln(err)
	}

	log.Println("Migrating database schema...")
	err = leash_helpers.MigrateSchema(db)
	if err != nil {
		log.Panicln(err)
	}

	// Google OAuth2
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("FRONTEND_URL") + "/auth/callback"
	externalAuth := leash_auth.GetGoogleAuthenticator(googleClientID, googleClientSecret, googleRedirectURL)

	// JWT Key
	log.Println("Initializing JWT Keys...")
	key_file := os.Getenv("KEY_FILE")
	set, err := leash_auth.CreateOrGetKeysFromFile(key_file)
	if err != nil {
		log.Panicln(err)
	}

	keys, err := leash_auth.CreateKeys(set)
	if err != nil {
		log.Panicln(err)
	}

	// Initialize RBAC
	log.Println("Initializing RBAC...")
	enforcer, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		log.Panicln(err)
	}

	leash_helpers.SetupCasbin(enforcer)

	// Create App
	log.Println("Initializing Fiber...")
	host := os.Getenv("HOST")
	if host == "" {
		host = DEFAULT_HOST
	}

	app := fiber.New()

	log.Println("Setting up middleware...")
	leash_helpers.SetupMiddlewares(app, db, keys, externalAuth, enforcer)

	log.Println("Setting up routes...")
	leash_helpers.SetupRoutes(app)

	log.Printf("Starting server on port %s\n", host)
	app.Listen(host)
}
