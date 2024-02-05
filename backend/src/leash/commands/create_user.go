package commands

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/erikgeiser/promptkit/selection"
	"github.com/erikgeiser/promptkit/textinput"
	"github.com/go-playground/validator/v10"
	"github.com/google/subcommands"
	"github.com/joho/godotenv"
	leash_helpers "github.com/mkrcx/mkrcx/src/leash/helpers"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"github.com/muesli/termenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type NewUserCmd struct{}

func (*NewUserCmd) Name() string     { return "new_user" }
func (*NewUserCmd) Synopsis() string { return "Create a new user" }
func (*NewUserCmd) Usage() string {
	return `new_user:
	  Create a new user
  `
}

func (p *NewUserCmd) SetFlags(f *flag.FlagSet) {}

func (p *NewUserCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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

	validate := validator.New()

	var user models.User

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

	emailInput := textinput.New("Email:")
	emailInput.Validate = func(value string) error {
		val := struct {
			Value string `validate:"required,email"`
		}{value}
		errs := validate.Struct(val)
		if errs != nil {
			return fmt.Errorf("invalid email: %v", errs)
		}

		return nil
	}

	email, err := emailInput.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return subcommands.ExitFailure
	}

	user.Email = email

	type selectionData struct {
		Value string
		Label string
	}

	blue := termenv.String().Foreground(termenv.ANSI256Color(32)) //nolint:gomnd
	selectionDataSelectedChoice := func(c *selection.Choice[selectionData]) string {
		return blue.Bold().Styled(c.Value.Label)
	}
	selectionDataUnselectedChoice := func(c *selection.Choice[selectionData]) string {
		return c.Value.Label
	}

	roles := []selectionData{
		{"admin", "Admin"},
		{"staff", "Staff"},
		{"volunteer", "Volunteer"},
		{"member", "Member"},
	}
	roleSelect := selection.New("Role:", roles)
	roleSelect.SelectedChoiceStyle = selectionDataSelectedChoice
	roleSelect.UnselectedChoiceStyle = selectionDataUnselectedChoice
	role, err := roleSelect.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return subcommands.ExitFailure
	}

	user.Role = role.Value

	types := []selectionData{
		{"undergrad", "Undergraduate Student"},
		{"grad", "Graduate Student"},
		{"alumni", "Alumni"},
		{"faculty", "Faculty"},
		{"staff", "Staff"},
		{"other", "Other"},
	}

	typeSelect := selection.New("Type:", types)
	typeSelect.SelectedChoiceStyle = selectionDataSelectedChoice
	typeSelect.UnselectedChoiceStyle = selectionDataUnselectedChoice
	typ, err := typeSelect.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return subcommands.ExitFailure
	}

	user.Type = typ.Value

	graduationYearInput := textinput.New("Graduation Year:")
	graduationYearInput.Validate = func(value string) error {
		val := struct {
			Value string `validate:"omitempty,numeric,min=0,max=9999"`
		}{value}
		errs := validate.Struct(val)
		if errs != nil {
			return fmt.Errorf("invalid graduation year: %v", errs)
		}

		return nil
	}

	graduationYear, err := graduationYearInput.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return subcommands.ExitFailure
	}

	user.GraduationYear, err = strconv.Atoi(graduationYear)
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return subcommands.ExitFailure
	}

	majorInput := textinput.New("Major:")
	majorInput.Validate = func(value string) error {
		return nil
	}

	major, err := majorInput.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return subcommands.ExitFailure
	}

	user.Major = major

	// Create User
	log.Println("Creating user...")
	err = db.Create(&user).Error
	if err != nil {
		log.Panicln(err)
	}

	log.Println("User created successfully")

	return subcommands.ExitSuccess
}
