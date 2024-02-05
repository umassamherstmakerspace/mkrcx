package commands

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/erikgeiser/promptkit/textinput"
	"github.com/go-playground/validator/v10"
	"github.com/google/subcommands"
	"github.com/joho/godotenv"
	leash_helpers "github.com/mkrcx/mkrcx/src/leash/helpers"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type NewServiceUserCmd struct{}

func (*NewServiceUserCmd) Name() string     { return "new_service" }
func (*NewServiceUserCmd) Synopsis() string { return "Create a new service user" }
func (*NewServiceUserCmd) Usage() string {
	return `new_service:
	  Create a new service user
  `
}

func (p *NewServiceUserCmd) SetFlags(f *flag.FlagSet) {}

func (p *NewServiceUserCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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

	// Initialize RBAC
	log.Println("Initializing RBAC...")
	e, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		log.Panicln(err)
	}

	leash_helpers.SetupCasbin(e)

	enforcer := leash_auth.EnforcerWrapper{
		Enforcer: e,
	}

	validate := validator.New()

	user := models.User{
		Role: "service",
		Type: "other",
	}

	nameInput := textinput.New("Name:")
	nameInput.Validate = func(value string) error {
		val := struct {
			Value string `validate:"required"`
		}{value}
		errs := validate.Struct(val)
		if errs != nil {
			return fmt.Errorf("invalid name: %v", errs)
		}

		return nil
	}

	name, err := nameInput.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return subcommands.ExitFailure
	}

	user.Name = name

	tagInput := textinput.New("Service Tag:")
	tagInput.Validate = func(value string) error {
		val := struct {
			Value string `validate:"required"`
		}{value}
		errs := validate.Struct(val)
		if errs != nil {
			return fmt.Errorf("invalid service tag: %v", errs)
		}

		return nil
	}

	tag, err := tagInput.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return subcommands.ExitFailure
	}

	user.Email = tag + "@mkrcx"

	permissionsInput := textinput.New("Permissions (seperated by commas and/or spaces):")
	permissionsInput.Validate = func(value string) error {
		return nil
	}

	permissions, err := permissionsInput.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return subcommands.ExitFailure
	}

	user.Permissions = []string{}

	permissions = strings.ReplaceAll(permissions, ",", " ")
	for _, permission := range strings.Fields(permissions) {
		if permission == "" {
			continue
		}

		user.Permissions = append(user.Permissions, permission)
	}

	fmt.Println("User:", user)

	// Create User
	log.Println("Creating service user...")
	err = db.Create(&user).Error
	if err != nil {
		log.Panicln(err)
	}

	enforcer.SetPermissionsForUser(user, user.Permissions)

	log.Println("Service user created successfully!")

	return subcommands.ExitSuccess
}
