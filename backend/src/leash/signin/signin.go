package leash_signin

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/disgoorg/log"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	leash_api "github.com/mkrcx/mkrcx/src/leash/api"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const (
	userTokenExpiration   = 7 * 24 * time.Hour
	loginCodeExpiration   = time.Minute
	loginReturnOriginsEnv = "LEASH_LOGIN_RETURN_ORIGINS"
	loginNonceCookieName  = "mkrcx_oauth_nonce"
)

var errInvalidLoginCode = errors.New("invalid or expired login code")

type loginReturnPolicy struct {
	origins map[string]struct{}
}

func newLoginReturnPolicy(raw string) (loginReturnPolicy, error) {
	policy := loginReturnPolicy{origins: make(map[string]struct{})}
	for _, candidate := range strings.Split(raw, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		parsed, err := url.Parse(candidate)
		if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return loginReturnPolicy{}, fmt.Errorf("%s must contain comma-separated origins", loginReturnOriginsEnv)
		}
		if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHostname(parsed.Hostname())) {
			return loginReturnPolicy{}, fmt.Errorf("%s origins must use HTTPS except for loopback development", loginReturnOriginsEnv)
		}
		policy.origins[origin(parsed)] = struct{}{}
	}
	return policy, nil
}

func isLoopbackHostname(hostname string) bool {
	switch strings.ToLower(hostname) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func origin(value *url.URL) string {
	return (&url.URL{Scheme: strings.ToLower(value.Scheme), Host: strings.ToLower(value.Host)}).String()
}

func (policy loginReturnPolicy) validate(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("login return URL is invalid")
	}
	if !parsed.IsAbs() {
		if parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
			return "", errors.New("login return URL must be an internal path or an allowed origin")
		}
		return parsed.String(), nil
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHostname(parsed.Hostname())) {
		return "", errors.New("login return URL must use HTTPS")
	}
	if _, allowed := policy.origins[origin(parsed)]; !allowed {
		return "", errors.New("login return URL origin is not allowed")
	}
	return parsed.String(), nil
}

func newLoginCode() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	code := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(code))
	return code, fmt.Sprintf("%x", digest), nil
}

func loginCodeHash(code string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil || len(decoded) != 32 {
		return "", errInvalidLoginCode
	}
	digest := sha256.Sum256([]byte(code))
	return fmt.Sprintf("%x", digest), nil
}

func newLoginNonce() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	nonce := base64.RawURLEncoding.EncodeToString(raw)
	digest := sha256.Sum256([]byte(nonce))
	return nonce, fmt.Sprintf("%x", digest), nil
}

func validLoginNonce(nonce, expectedHash string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(nonce)
	if err != nil || len(decoded) != 32 {
		return false
	}
	digest := sha256.Sum256([]byte(nonce))
	actualHash := fmt.Sprintf("%x", digest)
	return subtle.ConstantTimeCompare([]byte(actualHash), []byte(expectedHash)) == 1
}

func loginCookieSecure(c *fiber.Ctx) bool {
	return !isLoopbackHostname(c.Hostname())
}

func setLoginNonceCookie(c *fiber.Ctx, nonce string) {
	c.Cookie(&fiber.Cookie{
		Name: loginNonceCookieName, Value: nonce, Path: "/", HTTPOnly: true,
		Secure: loginCookieSecure(c), SameSite: fiber.CookieSameSiteLaxMode,
		Expires: time.Now().Add(5 * time.Minute),
	})
}

func clearLoginNonceCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name: loginNonceCookieName, Value: "", Path: "/", HTTPOnly: true,
		Secure: loginCookieSecure(c), SameSite: fiber.CookieSameSiteLaxMode,
		Expires: time.Unix(1, 0), MaxAge: -1,
	})
}

func claimLoginCode(db *gorm.DB, code string, now time.Time) (models.Session, error) {
	hash, err := loginCodeHash(strings.TrimSpace(code))
	if err != nil {
		return models.Session{}, errInvalidLoginCode
	}
	var session models.Session
	result := db.Where("login_code_hash = ? AND login_code_consumed_at IS NULL", hash).First(&session)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return models.Session{}, errInvalidLoginCode
	}
	if result.Error != nil {
		return models.Session{}, result.Error
	}
	if session.LoginCodeExpiresAt == nil || !session.LoginCodeExpiresAt.After(now) || !session.ExpiresAt.After(now) {
		return models.Session{}, errInvalidLoginCode
	}
	consumedAt := now.UTC()
	result = db.Model(&models.Session{}).
		Where("api_key = ? AND login_code_hash = ? AND login_code_consumed_at IS NULL AND login_code_expires_at > ?", session.SessionID, hash, now).
		Update("login_code_consumed_at", consumedAt)
	if result.Error != nil {
		return models.Session{}, result.Error
	}
	if result.RowsAffected != 1 {
		return models.Session{}, errInvalidLoginCode
	}
	session.LoginCodeConsumedAt = &consumedAt
	return session, nil
}

func loginRedirect(returnURL, code, state string) (string, error) {
	parsed, err := url.Parse(returnURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("code", code)
	query.Set("state", state)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

type sessionTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func createSessionToken(keys *leash_auth.Keys, user models.User, session models.Session, issuedAt time.Time) (sessionTokenResponse, error) {
	tok, err := jwt.NewBuilder().
		Issuer(leash_auth.ISSUER).
		IssuedAt(issuedAt).
		Expiration(session.ExpiresAt).
		Audience([]string{"leash", "session"}).
		Claim("email", user.Email).
		Claim("session", session.SessionID).
		Build()
	if err != nil {
		return sessionTokenResponse{}, err
	}
	signed, err := keys.Sign(tok)
	if err != nil {
		return sessionTokenResponse{}, err
	}
	return sessionTokenResponse{Token: string(signed), ExpiresAt: session.ExpiresAt}, nil
}

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
	returnPolicy, err := newLoginReturnPolicy(os.Getenv(loginReturnOriginsEnv))
	if err != nil {
		panic(err)
	}
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
		validatedReturn, validationErr := returnPolicy.validate(req.Return)
		if validationErr != nil {
			return fiber.NewError(fiber.StatusBadRequest, validationErr.Error())
		}
		req.Return = validatedReturn
		nonce, nonceHash, err := newLoginNonce()
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		// Create a token to store the return location signed by the server
		tok, err := jwt.NewBuilder().
			Issuer(leash_auth.ISSUER).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(5*time.Minute)).
			Audience([]string{"leash", "login-callback"}).
			Claim("return", req.Return).
			Claim("state", req.State).
			Claim("nonce", nonceHash).
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

		setLoginNonceCookie(c, nonce)
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
		nonceValue, nonceFound := tok.Get("nonce")
		nonceHash, nonceIsString := nonceValue.(string)
		cookieNonce := c.Cookies(loginNonceCookieName)
		clearLoginNonceCookie(c)
		if !nonceFound || !nonceIsString || !validLoginNonce(cookieNonce, nonceHash) {
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
		ret, err = returnPolicy.validate(ret)
		if err != nil {
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
			retryURL := "/auth/login?return=" + url.QueryEscape(ret)
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
						<a href="%s">Retry Login</a>
					</body>
				</html>
			`, html.EscapeString(retryURL)))
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

		code, codeHash, err := newLoginCode()
		if err != nil {
			log.Error("Failed to create the login exchange code: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		now := time.Now().UTC()
		codeExpiresAt := now.Add(loginCodeExpiration)
		session := models.Session{
			SessionID:          uuid.New().String(),
			UserID:             user.ID,
			ExpiresAt:          now.Add(userTokenExpiration),
			LoginCodeHash:      &codeHash,
			LoginCodeExpiresAt: &codeExpiresAt,
		}

		// Create the session
		res = db.Create(&session)
		if res.Error != nil {
			log.Error("Failed to create session: %s\n", res.Error)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		redirectURL, err := loginRedirect(ret, code, state)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		c.Set(fiber.HeaderCacheControl, "no-store")
		c.Set("Referrer-Policy", "no-referrer")
		return c.Redirect(redirectURL)
	})

	type loginExchangeRequest struct {
		Code string `json:"code" validate:"required,notblank,max=128"`
	}
	auth_ep.Post("/exchange", models.GetBodyMiddleware[loginExchangeRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		now := time.Now().UTC()
		req := c.Locals("body").(loginExchangeRequest)
		session, err := claimLoginCode(db, req.Code, now)
		if errors.Is(err, errInvalidLoginCode) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired login code")
		}
		if err != nil {
			return fiber.ErrInternalServerError
		}

		var user models.User
		if result := db.First(&user, session.UserID); result.Error != nil {
			return fiber.ErrInternalServerError
		}
		if leash_auth.SignInAuthentication(user, c).Authorize("leash:login") != nil {
			return fiber.ErrUnauthorized
		}
		response, err := createSessionToken(leash_auth.GetKeys(c), user, session, now)
		if err != nil {
			return fiber.ErrInternalServerError
		}
		c.Set(fiber.HeaderCacheControl, "no-store")
		return c.JSON(response)
	})

	// Logout is authenticated by the bearer header so the session token never
	// appears in a URL, browser history, or access log.
	auth_ep.Post("/logout", func(c *fiber.Ctx) error {
		authentication := leash_auth.GetAuthentication(c)
		if !authentication.IsUser() {
			return fiber.ErrUnauthorized
		}
		sessionID, ok := authentication.Data.(string)
		if !ok || sessionID == "" {
			return fiber.ErrUnauthorized
		}
		res := leash_auth.GetDB(c).Delete(&models.Session{SessionID: sessionID})
		if res.Error != nil {
			log.Error("Failed to delete session: %s\n", res.Error)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendStatus(fiber.StatusNoContent)
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
