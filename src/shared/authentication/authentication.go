package leash_authentication

import (
	"errors"
	"fmt"
	"log"
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

func (a Authentication) IsLoggedOut() bool {
	return a.Authenticator == AUTHENTICATOR_LOGGED_OUT
}

func (a Authentication) IsUser() bool {
	return a.Authenticator == AUTHENTICATOR_USER
}

func (a Authentication) IsAPIKey() bool {
	return a.Authenticator == AUTHENTICATOR_APIKEY
}

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

func (e EnforcerWrapper) HasPermissionForAPIKey(apikey models.APIKey, permission string) bool {
	val, err := e.e.Enforce(fmt.Sprintf("apikey:%s", apikey.Key), permission)
	if err != nil {
		return false
	}

	return val
}

func (e EnforcerWrapper) HasPermissionForUser(user models.User, permission string) bool {
	val, err := e.e.Enforce(fmt.Sprintf("user:%d", user.ID), permission)
	if err != nil {
		return false
	}

	return val
}

func (e EnforcerWrapper) AddPermissionForUser(user models.User, permission string) {
	e.e.AddPermissionForUser(fmt.Sprintf("user:%d", user.ID), permission)
}

func (e EnforcerWrapper) SetUserRole(user models.User, role string) {
	user_id := fmt.Sprintf("user:%d", user.ID)
	e.e.DeleteRolesForUser(user_id)
	e.e.AddRoleForUser(user_id, "role:"+role)

	e.e.SavePolicy()
}

func (e EnforcerWrapper) RemoveUserRole(user models.User) {
	user_id := fmt.Sprintf("user:%d", user.ID)
	e.e.DeleteRolesForUser(user_id)
}

func (e EnforcerWrapper) SetPermissionsForAPIKey(apikey models.APIKey, permissions []string) {
	apikey_id := fmt.Sprintf("apikey:%s", apikey.Key)
	e.e.DeletePermissionsForUser(apikey_id)
	for _, permission := range permissions {
		e.e.AddPermissionForUser(apikey_id, permission)
	}
}

func (e EnforcerWrapper) SetAPIKeyFullAccess(apikey models.APIKey, full_access bool) {
	apikey_id := fmt.Sprintf("apikey:%s", apikey.Key)
	user_id := fmt.Sprintf("user:%d", apikey.UserID)
	e.e.DeleteRolesForUser(apikey_id)
	if full_access {
		e.e.AddRoleForUser(apikey_id, user_id)
	}
}

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

func GetAuthentication(c *fiber.Ctx) Authentication {
	return c.Locals(ctxAuthKey{}).(Authentication)
}

func AuthenticationMiddleware(c *fiber.Ctx) error {
	db := GetDB(c)
	keys := GetKeys(c)
	// Make sure DB is alive
	sql, err := db.DB()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Database connection error")
	}

	err = sql.Ping()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Database connection error")
	}

	enforcer := EnforcerWrapper{
		e: GetEnforcer(c),
	}

	// Get the token from the request header
	authentication, err := func() (Authentication, error) {
		authentication := Authentication{
			Authenticator: AUTHENTICATOR_LOGGED_OUT,
			Enforcer:      enforcer,
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
			Enforcer:      enforcer,
		}

		return authentication, nil
	}()

	if err != nil {
		// Get the api key from the request header
		authentication, err = func() (Authentication, error) {
			authentication := Authentication{
				Authenticator: AUTHENTICATOR_LOGGED_OUT,
				Enforcer:      enforcer,
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
				Enforcer:      enforcer,
			}

			return authentication, nil
		}()

		if err != nil {
			authentication = Authentication{
				Authenticator: AUTHENTICATOR_LOGGED_OUT,
				Enforcer:      enforcer,
			}
		}
	}

	c.Locals(ctxAuthKey{}, authentication)
	return c.Next()
}

func AuthorizationMiddleware(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authentication := GetAuthentication(c)
		if authentication.Authorize(permission) != nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		return c.Next()
	}
}

func SetPermissionPrefixMiddleware(prefix string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("permission_prefix", prefix)
		return c.Next()
	}
}

func ConcatPermissionPrefixMiddleware(prefix string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permission_prefix := c.Locals("permission_prefix").(string)
		c.Locals("permission_prefix", permission_prefix+"."+prefix)
		return c.Next()
	}
}

func PrefixAuthorizationMiddleware(action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permission := c.Locals("permission_prefix").(string) + ":" + action
		return AuthorizationMiddleware(permission)(c)
	}
}

func InitalizeCasbin(db *gorm.DB) *casbin.Enforcer {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		log.Fatalf("error: adapter: %s", err)
	}

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
		log.Fatalf("error: model: %s", err)
	}
	enforcer, err := casbin.NewEnforcer(model, adapter)
	if err != nil {
		log.Fatalf("error: enforcer: %s", err)
	}

	return enforcer
}

type ctxDBKey struct{}
type ctxKeysKey struct{}
type ctxGoogleKey struct{}
type ctxEnforcerKey struct{}

func LocalsMiddleware(db *gorm.DB, keys Keys, google *oauth2.Config, enforcer *casbin.Enforcer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(ctxDBKey{}, db)
		c.Locals(ctxKeysKey{}, keys)
		c.Locals(ctxGoogleKey{}, google)
		c.Locals(ctxEnforcerKey{}, enforcer)
		return c.Next()
	}
}

func GetDB(c *fiber.Ctx) *gorm.DB {
	return c.Locals(ctxDBKey{}).(*gorm.DB)
}

func GetKeys(c *fiber.Ctx) Keys {
	return c.Locals(ctxKeysKey{}).(Keys)
}

func GetGoogle(c *fiber.Ctx) *oauth2.Config {
	return c.Locals(ctxGoogleKey{}).(*oauth2.Config)
}

func GetEnforcer(c *fiber.Ctx) *casbin.Enforcer {
	return c.Locals(ctxEnforcerKey{}).(*casbin.Enforcer)
}
