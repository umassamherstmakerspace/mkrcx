package leash_signin

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/lestrrat-go/jwx/v2/jwt"
	leash_api "github.com/mkrcx/mkrcx/src/leash/api"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

func RegisterAuthenticationEndpoints(auth_ep fiber.Router) {
	auth_ep.Use(leash_auth.AuthenticationMiddleware)

	auth_ep.Get("/login", func(c *fiber.Ctx) error {
		keys := leash_auth.GetKeys(c)
		google := leash_auth.GetGoogle(c)
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
		db := leash_auth.GetDB(c)
		keys := leash_auth.GetKeys(c)
		google := leash_auth.GetGoogle(c)
		type request struct {
			Code  string `query:"code" validate:"required"`
			State string `query:"state" validate:"required"`
		}

		return models.GetQueryMiddleware(request{}, func(c *fiber.Ctx) error {
			req := c.Locals("query").(request)

			ret := "/"
			{
				tok, err := keys.Parse(req.State)
				if err != nil {
					fmt.Printf("failed to parse token: %s\n", err)
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
			res := db.First(&user, "email = ? OR pending_email = ?", userinfo.Email, userinfo.Email)
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

			if user.PendingEmail == userinfo.Email {
				var err error
				user, err = leash_api.UpdatePendingEmail(user, c)

				if err != nil {
					fmt.Printf("failed to update pending email: %s\n", err)
					return c.SendStatus(fiber.StatusInternalServerError)
				}
			}

			authenticator := leash_auth.SignInAuthentication(user, c)

			if authenticator.Authorize("leash", "login") != nil {
				return c.SendStatus(fiber.StatusUnauthorized)
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
		})(c)
	})

	auth_ep.Get("/logout", func(c *fiber.Ctx) error {
		c.ClearCookie("token")
		return c.Redirect("/")
	})

	auth_ep.Get("/validate", func(c *fiber.Ctx) error {
		authentication := leash_auth.GetAuthentication(c)

		if !authentication.IsUser() {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		return c.SendString("Authorized")
	})

	auth_ep.Get("/refresh", func(c *fiber.Ctx) error {
		keys := leash_auth.GetKeys(c)
		authentication := leash_auth.GetAuthentication(c)

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
	})
}
