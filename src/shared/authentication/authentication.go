package leash_authentication

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber"
	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

type Authenticator int

const (
	AUTHENTICATOR_LOGGED_OUT Authenticator = iota
	AUTHENTICATOR_USER
	AUTHENTICATOR_APIKEY
)

type Authentication struct {
	Authenticator Authenticator
	User          models.User
	Data          interface{}
}

type ctxAuthKey struct{}

func authenticationMiddleware(db *gorm.DB, publicKey jwk.Key, next fiber.Handler) fiber.Handler {
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
