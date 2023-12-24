package leash_authentication

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"golang.org/x/oauth2"
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

func (a Authentication) IsLoggedOut() bool {
	return a.Authenticator == AUTHENTICATOR_LOGGED_OUT
}

func (a Authentication) IsUser() bool {
	return a.Authenticator == AUTHENTICATOR_USER
}

func (a Authentication) IsAPIKey() bool {
	return a.Authenticator == AUTHENTICATOR_APIKEY
}

func (a Authentication) Authorize(permissionObeject string, permissionAction string) error {
	if a.IsLoggedOut() {
		return errors.New("not logged in")
	}

	return nil
}

type ctxAuthKey struct{}

func GetAuthentication(c *fiber.Ctx) Authentication {
	return c.Locals(ctxAuthKey{}).(Authentication)
}

func AuthenticationMiddleware(db *gorm.DB, keys Keys, next fiber.Handler) fiber.Handler {
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
			tok, err := keys.Parse(token)
			if err != nil {
				return authentication, errors.New("invalid token")
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

				var apiKeyRecord = models.APIKey{Key: apiKey}

				res := db.First(&apiKeyRecord)
				if errors.Is(res.Error, gorm.ErrRecordNotFound) {
					// The API key is not valid
					return authentication, errors.New("invalid API key")
				}

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

func AuthorizationMiddleware(permissionObject string, permissionAction string, next fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authentication := GetAuthentication(c)
		if authentication.Authorize(permissionObject, permissionAction) != nil {
			return c.Status(401).SendString("Unauthorized")
		}

		return next(c)
	}
}

func RegisterAuthenticationEndpoints(auth_ep fiber.Router, db *gorm.DB, keys Keys, google *oauth2.Config) {
	auth_ep.Get("/login", func(c *fiber.Ctx) error {
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
			Issuer(`mkrcx`).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(5*time.Minute)).
			Claim("return", req.Return).
			Build()

		if err != nil {
			fmt.Printf("failed to build token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := keys.Sign(tok)
		if err != nil {
			fmt.Printf("failed to sign token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		url := google.AuthCodeURL(string(signed))
		return c.Redirect(url)
	})

	auth_ep.Get("/callback", func(c *fiber.Ctx) error {
		var req struct {
			Code  string `query:"code" validate:"required"`
			State string `query:"state" validate:"required"`
		}

		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid request body")
		}

		{
			errors := models.ValidateStruct(req)
			if errors != nil {
				return c.Status(fiber.StatusBadRequest).JSON(errors)
			}
		}

		ret := "/"
		{
			tok, err := keys.Parse(req.State)
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
				errors := models.ValidateStruct(userinfo)
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
			Issuer(`mkrcx`).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(24*time.Hour)).
			Claim("email", userinfo.Email).
			Build()
		if err != nil {
			fmt.Printf("failed to build token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := keys.Sign(tok)
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

	auth_ep.Get("/logout", func(c *fiber.Ctx) error {
		c.ClearCookie("token")
		return c.Redirect("/")
	})

	auth_ep.Get("/validate", AuthenticationMiddleware(db, keys, func(c *fiber.Ctx) error {
		authentication := GetAuthentication(c)

		if !authentication.IsUser() {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		return c.SendString("Authorized")
	}))

	auth_ep.Get("/auth/refresh", AuthenticationMiddleware(db, keys, func(c *fiber.Ctx) error {
		authentication := GetAuthentication(c)

		if !authentication.IsUser() {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		tok, err := jwt.NewBuilder().
			Issuer(`mkrcx`).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(24*time.Hour)).
			Claim("email", authentication.User.Email).
			Build()

		if err != nil {
			fmt.Printf("failed to build token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := keys.Sign(tok)
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
}
