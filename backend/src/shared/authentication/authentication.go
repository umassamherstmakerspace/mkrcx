package leash_authentication

import (
	"crypto/hmac"
	"crypto/md5"
	"errors"
	"fmt"
	"hash"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gofiber/fiber/v2"
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
	Enforcer      EnforcerWrapper
}

// ParseSessionToken parses the session token and returns the user and session string
func ParseSessionToken(db *gorm.DB, keys *Keys, token string) (*models.User, string, error) {
	// Parse the token
	tok, err := keys.Parse(token, []string{"leash", "session"})
	if err != nil {
		return nil, "", fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
	}

	// Get the email from the token
	email, valid := tok.Get("email")
	if !valid {
		return nil, "", fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
	}

	// Get the session from the token
	s, valid := tok.Get("session")
	if !valid {
		return nil, "", fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
	}

	session_str, ok := s.(string)
	if !ok {
		return nil, "", fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
	}

	// Check if the user exists
	var user = &models.User{
		Email: email.(string),
	}

	if res := db.Limit(1).Where(&user).Find(&user); res.Error != nil || res.RowsAffected == 0 {
		// The user does not exist
		return nil, "", fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
	}

	return user, session_str, nil
}

// IsLoggedOut returns true if the user in the current context is logged out
func (a Authentication) IsLoggedOut() bool {
	return a.Authenticator == AUTHENTICATOR_LOGGED_OUT
}

// IsUser returns true if the user in the current context is using a session
func (a Authentication) IsUser() bool {
	return a.Authenticator == AUTHENTICATOR_USER
}

// IsAPIKey returns true if the user in the current context is using an API key
func (a Authentication) IsAPIKey() bool {
	return a.Authenticator == AUTHENTICATOR_APIKEY
}

// Authorize returns nil if the user in the current context is authorized to perform the given action
func (a Authentication) Authorize(permission string) error {
	if a.IsLoggedOut() {
		return errors.New("not logged in")
	}

	if a.IsAPIKey() {
		if !a.Enforcer.HasPermissionForAPIKey(a.Data.(models.APIKey), permission) {
			return errors.New("not authorized")
		}
	}

	if a.Enforcer.HasPermissionForUser(a.User, permission) {
		return nil
	}

	return errors.New("not authorized")
}

type EnforcerWrapper struct {
	Enforcer *casbin.Enforcer
}

// HasPermissionForAPIKey returns true if the api key supplied is authorized to perform the given action
func (e EnforcerWrapper) HasPermissionForAPIKey(apikey models.APIKey, permission string) bool {
	if apikey.FullAccess {
		return true
	}

	val, err := e.Enforcer.Enforce(fmt.Sprintf("apikey:%s", apikey.Key), permission)
	if err != nil {
		return false
	}

	return val
}

// HasPermissionForUser returns true if the user supplied is authorized to perform the given action
func (e EnforcerWrapper) HasPermissionForUser(user models.User, permission string) bool {
	val, err := e.Enforcer.Enforce("role:"+user.Role, permission)
	if err != nil {
		return false
	}

	if val {
		return true
	}

	val, err = e.Enforcer.Enforce(fmt.Sprintf("user:%d", user.ID), permission)
	if err != nil {
		return false
	}

	return val
}

// SavePolicy saves the policy
func (e EnforcerWrapper) SavePolicy() error {
	return e.Enforcer.SavePolicy()
}

// AddPermissionsForUser adds the permissions for the user supplied
func (e EnforcerWrapper) SetPermissionsForUser(user models.User, permissions []string) {
	user_id := fmt.Sprintf("user:%d", user.ID)
	e.Enforcer.DeletePermissionsForUser(user_id)
	for _, permission := range permissions {
		e.Enforcer.AddPermissionForUser(user_id, permission)
	}
}

// SetPermissionsForAPIKey sets the permissions for the api key supplied
func (e EnforcerWrapper) SetPermissionsForAPIKey(apikey models.APIKey, permissions []string) {
	apikey_id := fmt.Sprintf("apikey:%s", apikey.Key)
	e.Enforcer.DeletePermissionsForUser(apikey_id)
	for _, permission := range permissions {
		e.Enforcer.AddPermissionForUser(apikey_id, permission)
	}
}

// SignInAuthentication returns an Authentication struct for the user supplied (used for signing in)
func SignInAuthentication(user models.User, c *fiber.Ctx) Authentication {
	return Authentication{
		Authenticator: AUTHENTICATOR_USER,
		User:          user,
		Enforcer: EnforcerWrapper{
			Enforcer: GetEnforcer(c),
		},
	}
}

const (
	ctxAuthKey         string = "auth"
	ctxDBKey           string = "db"
	ctxKeysKey         string = "keys"
	ctxHMACSecretKey   string = "hmac_secret"
	ctxExternalAuthKey string = "external_auth"
	ctxEnforcerKey     string = "enforcer"
)

// GetAuthentication returns the Authentication struct for the current context
func GetAuthentication(c *fiber.Ctx) Authentication {
	return c.Locals(ctxAuthKey).(Authentication)
}

// AuthenticateHeader takes the value of the Authorization header and returns the signin status
func AuthenticateHeader(authorization string, db *gorm.DB, keys *Keys, e *casbin.Enforcer) (Authentication, error) {
	// Get the enforcer
	enforcer := EnforcerWrapper{
		Enforcer: e,
	}

	authentication := Authentication{
		Authenticator: AUTHENTICATOR_LOGGED_OUT,
		Enforcer:      enforcer,
	}

	// If user has supplied an authorization header, use it
	if strings.HasPrefix(authorization, "Bearer ") {
		// Get the token from the authorization header
		token := strings.TrimPrefix(authorization, "Bearer ")

		user, session_str, err := ParseSessionToken(db, keys, token)
		if err != nil {
			return authentication, err
		}

		var session = models.Session{
			SessionID: session_str,
		}

		// Get the session
		res := db.Limit(1).Where(&session).Find(&session)
		if res.Error != nil || res.RowsAffected == 0 {
			return authentication, errors.New("session not found")
		}

		// Check if the session is expired
		if session.ExpiresAt.Before(time.Now()) {
			db.Delete(&session)
			return authentication, errors.New("session expired")
		}

		authentication = Authentication{
			Authenticator: AUTHENTICATOR_USER,
			User:          *user,
			Data:          session_str,
			Enforcer:      enforcer,
		}
	} else if strings.HasPrefix(authorization, "API-Key ") {
		// Get the api key from the authorization header
		key := strings.TrimPrefix(authorization, "API-Key ")

		// Check if the api key exists
		var apiKey = models.APIKey{
			Key: key,
		}

		if res := db.Limit(1).Where(&apiKey).Find(&apiKey); res.Error != nil || res.RowsAffected == 0 {
			// The api key does not exist
			return authentication, errors.New("API Key not found")
		}

		// Check if the user exists
		var user = models.User{
			ID: apiKey.UserID,
		}

		if res := db.Limit(1).Where(&user).Find(&user); res.Error != nil || res.RowsAffected == 0 {
			// The user does not exist
			return authentication, errors.New("user not found")
		}

		authentication = Authentication{
			Authenticator: AUTHENTICATOR_APIKEY,
			User:          user,
			Data:          apiKey,
			Enforcer:      enforcer,
		}
	}

	return authentication, nil
}

// AuthenticationMiddleware is the middleware that handles authentication
func AuthenticationMiddleware(c *fiber.Ctx) error {
	db := GetDB(c)

	// Make sure DB is alive
	sql, err := db.DB()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Database connection error")
	}

	err = sql.Ping()

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Database connection error")
	}

	authentication, err := AuthenticateHeader(c.Get("Authorization"), db, GetKeys(c), GetEnforcer(c))
	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	c.Locals(ctxAuthKey, authentication)
	return c.Next()
}

// AuthorizationMiddleware is the middleware that handles authorization
func AuthorizationMiddleware(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if the user is authorized to perform the action
		authentication := GetAuthentication(c)
		if authentication.Authorize(permission) != nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		return c.Next()
	}
}

// SetPermissionPrefixMiddleware is the middleware that sets the permission prefix
func SetPermissionPrefixMiddleware(prefix string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("permission_prefix", prefix)
		return c.Next()
	}
}

// ConcatPermissionPrefixMiddleware is the middleware that concatenates the permission prefix
func ConcatPermissionPrefixMiddleware(prefix string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permission_prefix := c.Locals("permission_prefix").(string)
		c.Locals("permission_prefix", permission_prefix+"."+prefix)
		return c.Next()
	}
}

// PrefixAuthorizationMiddleware is the middleware that handles authorization with a prefix
func PrefixAuthorizationMiddleware(action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permission := c.Locals("permission_prefix").(string) + ":" + action
		return AuthorizationMiddleware(permission)(c)
	}
}

// InitializeCasbin initializes the casbin enforcer
func InitializeCasbin(db *gorm.DB) (*casbin.Enforcer, error) {
	// Initialize the adapter from the DB
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, err
	}

	// Initialize the RBAC model
	model, err := model.NewModelFromString(`
	[request_definition]
	r = sub, perm

	[policy_definition]
	p = sub, perm

	[role_definition]
	g = _, _

	[policy_effect]
	e = some(where (p.eft == allow))

	[matchers]
	m = g(r.sub, p.sub) && r.perm == p.perm
	`)

	if err != nil {
		return nil, err
	}

	// Create the enforcer from the model and adapter
	enforcer, err := casbin.NewEnforcer(model, adapter)
	if err != nil {
		return nil, err
	}

	return enforcer, nil
}

// LocalsMiddleware is the middleware that sets the locals for common objects
func LocalsMiddleware(db *gorm.DB, keys *Keys, hmacSecret []byte, externalAuth ExternalAuthenticator, enforcer *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(ctxDBKey, db)
		c.Locals(ctxKeysKey, keys)
		c.Locals(ctxHMACSecretKey, hmacSecret)
		c.Locals(ctxExternalAuthKey, externalAuth)
		c.Locals(ctxEnforcerKey, enforcer)
		return c.Next()
	}
}

// GetDB returns the database from the current context
func GetDB(c *fiber.Ctx) *gorm.DB {
	return c.Locals(ctxDBKey).(*gorm.DB)
}

// GetKeys returns the keys from the current context
func GetKeys(c *fiber.Ctx) *Keys {
	return c.Locals(ctxKeysKey).(*Keys)
}

func GetHMAC(c *fiber.Ctx) hash.Hash {
	return hmac.New(md5.New, c.Locals(ctxHMACSecretKey).([]byte))
}

// GetGoogle returns the google oauth2 config from the current context
func GetExternalAuth(c *fiber.Ctx) ExternalAuthenticator {
	return c.Locals(ctxExternalAuthKey).(ExternalAuthenticator)
}

// GetEnforcer returns the casbin enforcer from the current context
func GetEnforcer(c *fiber.Ctx) *casbin.Enforcer {
	return c.Locals(ctxEnforcerKey).(*casbin.Enforcer)
}
