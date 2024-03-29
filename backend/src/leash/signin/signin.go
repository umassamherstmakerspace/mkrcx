package leash_signin

import (
	"fmt"
	"time"

	"github.com/disgoorg/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	leash_api "github.com/mkrcx/mkrcx/src/leash/api"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

const userTokenExpiration = 7 * 24 * time.Hour

// NoAPIKeyMiddleware is a middleware that checks if the user has an API key
func NoAPIKeyMiddleware(c *fiber.Ctx) error {
	authentication := leash_auth.GetAuthentication(c)

	// Check if the user has an API key
	if authentication.IsAPIKey() {
		return c.SendStatus(fiber.StatusBadRequest)
	}

	return c.Next()
}

// RegisterAuthenticationEndpoints registers the authentication endpoints
func RegisterAuthenticationEndpoints(auth_ep fiber.Router) {
	auth_ep.Use(leash_auth.AuthenticationMiddleware)
	auth_ep.Use(NoAPIKeyMiddleware)

	// Endpoint to initialize login in
	type signinRequest struct {
		Return string `query:"return"`
		State  string `query:"state"`
	}

	auth_ep.Get("/login", models.GetQueryMiddleware[signinRequest], func(c *fiber.Ctx) error {
		keys := leash_auth.GetKeys(c)
		externalAuth := leash_auth.GetExternalAuth(c)
		req := c.Locals("query").(signinRequest)

		// Default return to /
		if req.Return == "" {
			req.Return = "/"
		}

		// Create a token to store the return location signed by the server
		tok, err := jwt.NewBuilder().
			Issuer(leash_auth.ISSUER).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(5*time.Minute)).
			Audience([]string{"leash", "login-callback"}).
			Claim("return", req.Return).
			Claim("state", req.State).
			Build()

		if err != nil {
			log.Error("Failed to build the login token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := keys.Sign(tok)
		if err != nil {
			log.Error("Failed to sign the login token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		url := externalAuth.GetAuthURL(string(signed))
		return c.Redirect(url)
	})

	// Endpoint to handle the callback from google
	type signinCallbackRequest struct {
		Code  string `query:"code" validate:"required"`
		State string `query:"state" validate:"required"`
	}

	auth_ep.Get("/callback", models.GetQueryMiddleware[signinCallbackRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		keys := leash_auth.GetKeys(c)
		externalAuth := leash_auth.GetExternalAuth(c)
		req := c.Locals("query").(signinCallbackRequest)

		// Parse the state token
		tok, err := keys.Parse(req.State, []string{"leash", "login-callback"})
		if err != nil {
			log.Error("Failed to parse state token: %s\n", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid state")
		}

		// Get the return location from the state token
		val, valid := tok.Get("return")
		if !valid {
			log.Error("Failed to get return from state token: %s\n", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid state")
		}

		ret, ok := val.(string)
		if !ok {
			log.Error("Failed to convert return from state token: %s\n", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid state")
		}

		// Get the return state from the state token
		val, valid = tok.Get("state")
		if !valid {
			log.Error("Failed to get state from state token: %s\n", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid state")
		}

		state, ok := val.(string)
		if !ok {
			log.Error("Failed to convert state from state token: %s\n", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid state")
		}

		email, err := externalAuth.Callback(c.Context(), req.Code)
		if err != nil {
			log.Error("Failed to get email from external auth: %s\n", err)
			return c.Status(fiber.StatusBadRequest).SendString("Invalid code")
		}

		// Check if the user exists
		var user models.User
		res := db.Limit(1).Where(models.User{Email: email}).Or(models.User{PendingEmail: &email}).Find(&user)
		if res.Error != nil || res.RowsAffected == 0 {
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

		// Check if the user signed in with a pending email
		if user.PendingEmail != nil && *user.PendingEmail == email {
			var err error
			user, err = leash_api.UpdatePendingEmail(user, c)

			if err != nil {
				log.Error("Failed to update pending email: %s\n", err)
				return c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		// Create a new authentication
		authenticator := leash_auth.SignInAuthentication(user, c)

		// Check if user has permission to login
		if authenticator.Authorize("leash:login") != nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		session_id := uuid.New().String()

		// Create a session token
		tok, err = jwt.NewBuilder().
			Issuer(leash_auth.ISSUER).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(userTokenExpiration)).
			Audience([]string{"leash", "session"}).
			Claim("email", email).
			Claim("session", session_id).
			Build()
		if err != nil {
			log.Error("Failed to build the session token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := keys.Sign(tok)
		if err != nil {
			log.Error("Failed to sign the session token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		session := models.Session{
			SessionID: session_id,
			UserID:    user.ID,
			ExpiresAt: tok.Expiration(),
		}

		// Create the session
		res = db.Create(&session)
		if res.Error != nil {
			log.Error("Failed to create session: %s\n", res.Error)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.Redirect(ret + "?token=" + string(signed) + "&expires_at=" + session.ExpiresAt.Format(time.RFC3339) + "&state=" + state)
	})

	// Endpoint to logout
	type logoutRequest struct {
		Return string `query:"return"`
		Token  string `query:"token" validate:"required"`
	}
	auth_ep.Get("/logout", models.GetQueryMiddleware[logoutRequest], func(c *fiber.Ctx) error {
		req := c.Locals("query").(logoutRequest)

		// Default return to /
		if req.Return == "" {
			req.Return = "/"
		}

		_, session_str, err := leash_auth.ParseSessionToken(leash_auth.GetDB(c), leash_auth.GetKeys(c), req.Token)
		if err != nil {
			return err
		}

		// Delete the session
		res := leash_auth.GetDB(c).Delete(&models.Session{SessionID: session_str})
		if res.Error != nil {
			log.Error("Failed to delete session: %s\n", res.Error)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.Redirect(req.Return)
	})

	// Endpoint to validate the session token
	auth_ep.Get("/validate", func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		authentication := leash_auth.GetAuthentication(c)

		// This should only be called with a valid user session token
		if !authentication.IsUser() {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		var session = models.Session{
			SessionID: authentication.Data.(string),
		}

		// Get the session
		res := db.Limit(1).Where(&session).Find(&session)
		if res.Error != nil || res.RowsAffected == 0 {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		// Check if the session is expired
		if session.ExpiresAt.Before(time.Now()) {
			db.Delete(&session)
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		return c.SendString("Authorized")
	})

	// Endpoint to refresh the session token
	auth_ep.Get("/refresh", func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		keys := leash_auth.GetKeys(c)
		authentication := leash_auth.GetAuthentication(c)

		// This should only be called with a valid user session token
		if !authentication.IsUser() {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		var session = models.Session{
			SessionID: authentication.Data.(string),
		}

		// Get the session
		res := db.Limit(1).Where(&session).Find(&session)
		if res.Error != nil || res.RowsAffected == 0 {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		// Check if the session is expired
		if session.ExpiresAt.Before(time.Now()) {
			db.Delete(&session)
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		// Create a new session token
		tok, err := jwt.NewBuilder().
			Issuer(leash_auth.ISSUER).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(userTokenExpiration)).
			Audience([]string{"leash", "session"}).
			Claim("email", authentication.User.Email).
			Claim("session", authentication.Data).
			Build()

		if err != nil {
			log.Error("Failed to build the session token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := keys.Sign(tok)
		if err != nil {
			log.Error("Failed to sign the session token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		// Update the session
		session.ExpiresAt = tok.Expiration()
		res = db.Save(&session)
		if res.Error != nil {
			log.Error("Failed to update session: %s\n", res.Error)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		// Return the new session token and expiration
		return c.JSON(struct {
			Token     string    `json:"token"`
			ExpiresAt time.Time `json:"expires_at"`
		}{
			Token:     string(signed),
			ExpiresAt: tok.Expiration(),
		})
	})
}
