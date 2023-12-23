package main

import (
	"errors"
	"log"
	"os"
	"path"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	leash_api "github.com/mkrcx/mkrcx/src/leash/api"
	leash_frontend "github.com/mkrcx/mkrcx/src/leash/frontend"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	models "github.com/mkrcx/mkrcx/src/shared/models"
)

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

type UserIDReq struct {
	ID    uint   `json:"id" xml:"id" form:"id" query:"id" validate:"required_without=Email"`
	Email string `json:"email" xml:"email" form:"email" query:"email" validate:"required_without=ID"`
}

const SYSTEM_USER_EMAIL = "makerspace@umass.edu"
const HOST = ":8000"

func main() {
	db, err := gorm.Open(mysql.Open(os.Getenv("DB")), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	models.SetupValidator()

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
	keys := leash_auth.InitalizeJWT()

	// Discord Webhook
	// webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	// var webhookClient webhook.Client
	// if webhookURL != "" {
	// 	webhookClient, err = webhook.NewWithURL(webhookURL)
	// 	if err != nil {
	// 		fmt.Printf("failed to create webhook: %s\n", err)
	// 	}
	// }

	frontend_dir := os.Getenv("FRONTEND_DIR")

	api := app.Group("/api")

	leash_api.RegisterAPIEndpoints(api, db, keys)

	auth := api.Group("/auth")

	leash_auth.RegisterAuthenticationEndpoints(auth, db, keys, google)

	// app.Get("/discord/enable", cookieAuthMiddleware(publicKey, leash_auth.AuthenticationMiddleware(db, keys, func(c *fiber.Ctx) error {
	// 	authentication := leash_auth.GetAuthentication(c)

	// 	if authentication.Authorize("leash.users:write") != nil {
	// 		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	// 	}

	// 	var req struct {
	// 		Token string `query:"token" validate:"required"`
	// 	}

	// 	if err := c.QueryParser(&req); err != nil {
	// 		return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
	// 	}

	// 	{
	// 		errors := models.ValidateStruct(req)
	// 		if errors != nil {
	// 			return c.Status(fiber.StatusBadRequest).JSON(errors)
	// 		}
	// 	}

	// 	var user_id int
	// 	var message_id snowflake.ID
	// 	{
	// 		tok, err := jwt.ParseString(req.Token, jwt.WithKey(jwa.RS256, publicKey))
	// 		if err != nil {
	// 			fmt.Printf("failed to parse token: %s\n", err)
	// 			return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
	// 		}

	// 		if err := jwt.Validate(tok); err != nil {
	// 			fmt.Printf("failed to validate token: %s\n", err)
	// 			return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
	// 		}

	// 		val, valid := tok.Get("user_id")
	// 		if !valid {
	// 			fmt.Printf("failed to get id value: %s\n", err)
	// 			return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
	// 		}

	// 		user_id, err = strconv.Atoi(fmt.Sprintf("%v", val))
	// 		if err != nil {
	// 			fmt.Printf("failed to convert id value: %s\n", err)
	// 			return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
	// 		}

	// 		val, valid = tok.Get("message_id")
	// 		if !valid {
	// 			fmt.Printf("failed to get message id value: %s\n", err)
	// 			return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
	// 		}

	// 		message_id, err = snowflake.Parse(fmt.Sprintf("%v", val))
	// 		if err != nil {
	// 			fmt.Printf("failed to convert message id value: %s\n", err)
	// 			return c.Status(fiber.StatusBadRequest).SendString("Invalid token")
	// 		}
	// 	}

	// 	var user models.User
	// 	res := db.First(&user, "id = ?", user_id)
	// 	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
	// 		// The user does not exist
	// 		return c.Status(fiber.StatusBadRequest).SendString("User not found")
	// 	}

	// 	user.Enabled = true
	// 	db.Save(&user)

	// 	// Create a new update in the database
	// 	update := models.UserUpdate{
	// 		UserID:   user.ID,
	// 		EditedBy: authentication.User.ID,
	// 		Field:    "enabled",
	// 		OldValue: "false",
	// 		NewValue: "true",
	// 	}

	// 	db.Create(&update)

	// 	// Send a discord webhook
	// 	if webhookClient != nil {
	// 		embed := discord.NewEmbedBuilder().
	// 			SetTitle("User Enabled").
	// 			SetDescription("User has been enabled.").
	// 			SetColor(0xff00B0).
	// 			AddField("Name", user.Name, true).
	// 			AddField("Email", user.Email, true).
	// 			AddField("Enabled By", authentication.User.Name, false).
	// 			SetTimestamp(time.Now()).
	// 			Build()

	// 		_, err := webhookClient.UpdateEmbeds(message_id, []discord.Embed{embed})
	// 		if err != nil {
	// 			fmt.Printf("failed to send webhook: %s\n", err)
	// 		}
	// 	}

	// 	return c.Redirect("/")
	// })))

	leash_frontend.SetupFrontend(app, "/", frontend_dir)

	log.Printf("Starting server on port %s\n", HOST)
	app.Listen(HOST)
}

// func userTrainingEnable(db *gorm.DB, user models.User, webhookClient webhook.Client, URL string, privateKey jwk.Key) {
// 	var trainings []models.Training
// 	db.Find(&trainings, "user_id = ?", user.ID)
// 	orientationCompleted := false
// 	docusignCompleted := false
// 	for _, training := range trainings {
// 		if training.TrainingType == "orientation" {
// 			orientationCompleted = true
// 		}
// 		if training.TrainingType == "docusign" {
// 			docusignCompleted = true
// 		}
// 	}

// 	if orientationCompleted && docusignCompleted {
// 		// Send a discord webhook
// 		if webhookClient != nil {
// 			message, err := webhookClient.CreateContent("Awaiting Token Generation")
// 			if err != nil {
// 				fmt.Printf("failed to send webhook: %s\n", err)
// 			}

// 			fmt.Println(message)

// 			token, err := jwt.NewBuilder().
// 				Issuer(`github.com/lestrrat-go/jwx`).
// 				IssuedAt(time.Now()).
// 				Claim("user_id", user.ID).
// 				Claim("message_id", message.ID).
// 				Build()

// 			if err != nil {
// 				fmt.Printf("failed to build token: %s\n", err)
// 				return
// 			}

// 			signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, privateKey))
// 			if err != nil {
// 				fmt.Printf("failed to sign token: %s\n", err)
// 				return
// 			}

// 			embed := discord.NewEmbedBuilder().
// 				SetTitle("User Awaiting Verification").
// 				SetDescription("A user has completed the orientation and docusign trainings and is awaiting verification.").
// 				SetColor(0xffa000).
// 				AddField("Name", user.Name, true).
// 				AddField("Email", user.Email, true).
// 				AddField("Verification Link", fmt.Sprintf(URL+"/discord/enable?token=%s", signed), false).
// 				SetTimestamp(time.Now()).
// 				Build()

// 			m := ""
// 			_, err = webhookClient.UpdateMessage(message.ID, discord.WebhookMessageUpdate{
// 				Embeds:  &[]discord.Embed{embed},
// 				Content: &m,
// 			})

// 			if err != nil {
// 				fmt.Printf("failed to update webhook: %s\n", err)
// 			}
// 		}
// 	}

// }

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
