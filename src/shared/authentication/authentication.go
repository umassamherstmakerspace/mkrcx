package leash_authentication

import (
	"errors"
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gofiber/fiber/v2"
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
	Enforcer      EnforcerWrapper
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
	e *casbin.Enforcer
}

// HasPermissionForAPIKey returns true if the api key supplied is authorized to perform the given action
func (e EnforcerWrapper) HasPermissionForAPIKey(apikey models.APIKey, permission string) bool {
	val, err := e.e.Enforce(fmt.Sprintf("apikey:%s", apikey.Key), permission)
	if err != nil {
		return false
	}

	return val
}

// HasPermissionForUser returns true if the user supplied is authorized to perform the given action
func (e EnforcerWrapper) HasPermissionForUser(user models.User, permission string) bool {
	val, err := e.e.Enforce(fmt.Sprintf("user:%d", user.ID), permission)
	if err != nil {
		return false
	}

	return val
}

// AddPermissionForUser adds a permission for the user supplied
func (e EnforcerWrapper) AddPermissionForUser(user models.User, permission string) {
	e.e.AddPermissionForUser(fmt.Sprintf("user:%d", user.ID), permission)
}

// SetUserRole sets the role for the user supplied
func (e EnforcerWrapper) SetUserRole(user models.User, role string) {
	user_id := fmt.Sprintf("user:%d", user.ID)
	e.e.DeleteRolesForUser(user_id)
	e.e.AddRoleForUser(user_id, "role:"+role)

	e.e.SavePolicy()
}

// RemoveUserRole removes the role for the user supplied
func (e EnforcerWrapper) RemoveUserRole(user models.User) {
	user_id := fmt.Sprintf("user:%d", user.ID)
	e.e.DeleteRolesForUser(user_id)
}

// SetPermissionsForAPIKey sets the permissions for the api key supplied
func (e EnforcerWrapper) SetPermissionsForAPIKey(apikey models.APIKey, permissions []string) {
	apikey_id := fmt.Sprintf("apikey:%s", apikey.Key)
	e.e.DeletePermissionsForUser(apikey_id)
	for _, permission := range permissions {
		e.e.AddPermissionForUser(apikey_id, permission)
	}
}

// SetAPIKeyFullAccess sets the full access flag for the api key supplied
func (e EnforcerWrapper) SetAPIKeyFullAccess(apikey models.APIKey, full_access bool) {
	apikey_id := fmt.Sprintf("apikey:%s", apikey.Key)
	user_id := fmt.Sprintf("user:%d", apikey.UserID)
	e.e.DeleteRolesForUser(apikey_id)
	if full_access {
		e.e.AddRoleForUser(apikey_id, user_id)
	}
}

// SignInAuthentication returns an Authentication struct for the user supplied (used for signing in)
func SignInAuthentication(user models.User, c *fiber.Ctx) Authentication {
	return Authentication{
		Authenticator: AUTHENTICATOR_USER,
		User:          user,
		Enforcer: EnforcerWrapper{
			e: GetEnforcer(c),
		},
	}
}

type ctxAuthKey struct{}

// GetAuthentication returns the Authentication struct for the current context
func GetAuthentication(c *fiber.Ctx) Authentication {
	return c.Locals(ctxAuthKey{}).(Authentication)
}

// AuthenticationMiddleware is the middleware that handles authentication
func AuthenticationMiddleware(c *fiber.Ctx) error {
	db := GetDB(c)
	keys := GetKeys(c)

	// Make sure DB is alive
	sql, err := db.DB()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Database connection error")
	}

	err = sql.Ping()

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Database connection error")
	}

	// Get the enforcer

	enforcer := EnforcerWrapper{
		e: GetEnforcer(c),
	}

	authentication := Authentication{
		Authenticator: AUTHENTICATOR_LOGGED_OUT,
		Enforcer:      enforcer,
	}

	// Get the authorization header
	authorization := c.Get("Authorization")

	// If user has supplied an authorization header, use it
	if strings.HasPrefix(authorization, "Bearer ") {
		// Get the token from the authorization header
		token := strings.TrimPrefix(authorization, "Bearer ")

		// Parse the token
		tok, err := keys.Parse(token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
		}

		// Get the email from the token
		email, valid := tok.Get("email")
		if !valid {
			return fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
		}

		// Check if the user exists
		var user = models.User{
			Email: email.(string),
		}

		if res := db.Limit(1).Where(&user).Find(&user); res.Error != nil || res.RowsAffected == 0 {
			// The user does not exist
			return fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
		}

		authentication = Authentication{
			Authenticator: AUTHENTICATOR_USER,
			User:          user,
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
			return fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
		}

		// Check if the user exists
		var user = models.User{}
		user.ID = apiKey.UserID

		if res := db.Limit(1).Where(&user).Find(&user); res.Error != nil || res.RowsAffected == 0 {
			// The user does not exist
			return fiber.NewError(fiber.StatusUnauthorized, "Authorization header error")
		}

		authentication = Authentication{
			Authenticator: AUTHENTICATOR_APIKEY,
			User:          user,
			Data:          apiKey,
			Enforcer:      enforcer,
		}
	}

	c.Locals(ctxAuthKey{}, authentication)
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

// InitalizeCasbin initalizes the casbin enforcer
func InitalizeCasbin(db *gorm.DB) (*casbin.Enforcer, error) {
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

type ctxDBKey struct{}
type ctxKeysKey struct{}
type ctxGoogleKey struct{}
type ctxEnforcerKey struct{}

// LocalsMiddleware is the middleware that sets the locals for common objects
func LocalsMiddleware(db *gorm.DB, keys *Keys, google *oauth2.Config, enforcer *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(ctxDBKey{}, db)
		c.Locals(ctxKeysKey{}, keys)
		c.Locals(ctxGoogleKey{}, google)
		c.Locals(ctxEnforcerKey{}, enforcer)
		return c.Next()
	}
}

// GetDB returns the database from the current context
func GetDB(c *fiber.Ctx) *gorm.DB {
	return c.Locals(ctxDBKey{}).(*gorm.DB)
}

// GetKeys returns the keys from the current context
func GetKeys(c *fiber.Ctx) *Keys {
	return c.Locals(ctxKeysKey{}).(*Keys)
}

// GetGoogle returns the google oauth2 config from the current context
func GetGoogle(c *fiber.Ctx) *oauth2.Config {
	return c.Locals(ctxGoogleKey{}).(*oauth2.Config)
}

// GetEnforcer returns the casbin enforcer from the current context
func GetEnforcer(c *fiber.Ctx) *casbin.Enforcer {
	return c.Locals(ctxEnforcerKey{}).(*casbin.Enforcer)
}
