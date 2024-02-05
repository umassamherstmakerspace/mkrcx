package commands

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/erikgeiser/promptkit/selection"
	"github.com/erikgeiser/promptkit/textinput"
	"github.com/go-playground/validator/v10"
	"github.com/google/subcommands"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	leash_helpers "github.com/mkrcx/mkrcx/src/leash/helpers"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"github.com/muesli/termenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type NewApiKeyCmd struct{}

func (*NewApiKeyCmd) Name() string     { return "new_apikey" }
func (*NewApiKeyCmd) Synopsis() string { return "Create a new api key for a service user" }
func (*NewApiKeyCmd) Usage() string {
	return `new_apikey:
	  Create a new api key for a service user
  `
}

func (p *NewApiKeyCmd) SetFlags(f *flag.FlagSet) {}

func (p *NewApiKeyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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

	users := []models.User{}
	db.Where(models.User{Role: "service"}).Find(&users)

	blue := termenv.String().Foreground(termenv.ANSI256Color(32)) //nolint:gomnd

	userSelect := selection.New("Select User:", users)
	userSelect.SelectedChoiceStyle = func(c *selection.Choice[models.User]) string {
		return (blue.Bold().Styled(c.Value.Name) + " " + termenv.String("("+c.Value.Email+")").Faint().String())
	}
	userSelect.UnselectedChoiceStyle = func(c *selection.Choice[models.User]) string {
		return c.Value.Name + " " + termenv.String("("+c.Value.Email+")").Faint().String()
	}
	userSelect.FinalChoiceStyle = func(c *selection.Choice[models.User]) string {
		return (blue.Bold().Styled(c.Value.Name) + " " + termenv.String("("+c.Value.Email+")").Faint().String())
	}
	userSelect.FilterPrompt = "Filter users by name or service tag:"
	userSelect.Filter = func(filter string, choice *selection.Choice[models.User]) bool {
		filter = strings.ToLower(filter)
		return strings.HasPrefix(strings.ToLower(choice.Value.Email), filter) || strings.HasPrefix(strings.ToLower(choice.Value.Name), filter)
	}

	selectedUser, err := userSelect.RunPrompt()
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	apikey := models.APIKey{
		Key:    uuid.New().String(),
		UserID: selectedUser.ID,
	}

	descriptor := textinput.New("Description:")
	descriptor.Validate = func(value string) error {
		return nil
	}

	description, err := descriptor.RunPrompt()
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	apikey.Description = description

	fullAccessChoice := confirmation.New("Full Access:", confirmation.Undecided)

	fullAccess, err := fullAccessChoice.RunPrompt()
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	apikey.FullAccess = fullAccess

	if !fullAccess {
		permissionsInput := textinput.New("Permissions (seperated by commas and/or spaces):")
		permissionsInput.Validate = func(value string) error {
			return nil
		}

		permissions, err := permissionsInput.RunPrompt()
		if err != nil {
			fmt.Printf("Error: %v\n", err)

			return subcommands.ExitFailure
		}

		apikey.Permissions = []string{}

		permissions = strings.ReplaceAll(permissions, ",", " ")
		for _, permission := range strings.Fields(permissions) {
			if permission == "" {
				continue
			}

			apikey.Permissions = append(apikey.Permissions, permission)
		}
	}

	log.Println("Creating API Key...")

	log.Println("API Key:", apikey)

	err = db.Create(&apikey).Error
	if err != nil {
		log.Panicln(err)
	}

	enforcer.SetPermissionsForAPIKey(apikey, apikey.Permissions)

	log.Println("API Key created successfully!")

	_ = selectedUser
	_ = enforcer
	_ = validate

	// fmt.Println("User:", user)

	// // Create User
	// log.Println("Creating service user...")
	// err = db.Create(&user).Error
	// if err != nil {
	// 	log.Panicln(err)
	// }

	// enforcer.SetPermissionsForUser(user, user.Permissions)

	log.Println("Service user created successfully!")

	return subcommands.ExitSuccess
}
