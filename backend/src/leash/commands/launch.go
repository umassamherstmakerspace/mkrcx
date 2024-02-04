package commands

import (
	"context"
	"flag"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
	"github.com/google/subcommands"
	"github.com/joho/godotenv"
	leash_helpers "github.com/mkrcx/mkrcx/src/leash/helpers"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
)

const DEFAULT_HOST = ":8000"

type LaunchCmd struct{}

func (*LaunchCmd) Name() string     { return "launch" }
func (*LaunchCmd) Synopsis() string { return "Launch the Leash server" }
func (*LaunchCmd) Usage() string {
	return `launch:
	  Launch the Leash server
  `
}

func (p *LaunchCmd) SetFlags(f *flag.FlagSet) {}

func (p *LaunchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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
	if googleClientID == "" {
		log.Panicln("GOOGLE_CLIENT_ID is not set")
	}

	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if googleClientSecret == "" {
		log.Panicln("GOOGLE_CLIENT_SECRET is not set")
	}

	googleRedirectURL := os.Getenv("LEASH_URL") + "/auth/callback"
	externalAuth := leash_auth.GetGoogleAuthenticator(googleClientID, googleClientSecret, googleRedirectURL)

	// JWT Key
	log.Println("Initializing JWT Keys...")
	key_file := os.Getenv("KEY_FILE")
	if key_file == "" {
		log.Panicln("KEY_FILE is not set")
	}
	set, err := leash_auth.CreateOrGetKeysFromFile(key_file)
	if err != nil {
		log.Panicln(err)
	}

	keys, err := leash_auth.CreateKeys(set)
	if err != nil {
		log.Panicln(err)
	}

	hmacSecret := os.Getenv("HMAC_SECRET")
	if hmacSecret == "" {
		log.Panicln("HMAC_SECRET is not set")
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
	leash_helpers.SetupMiddlewares(app, db, keys, []byte(hmacSecret), externalAuth, enforcer)

	log.Println("Setting up routes...")
	leash_helpers.SetupRoutes(app)

	log.Printf("Starting server on port %s\n", host)
	app.Listen(host)

	return subcommands.ExitSuccess
}
