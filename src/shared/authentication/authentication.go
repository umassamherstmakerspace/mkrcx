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
	Enforcer      *casbin.Enforcer
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

func (a Authentication) tryAuthorize(subject string, object string, action string) error {
	val, err := a.Enforcer.Enforce(subject, object, action)
	if err != nil {
		return err
	}

	if !val {
		return errors.New("not authorized")
	}

	return nil
}

func (a Authentication) Authorize(permissionObeject string, permissionAction string) error {
	if a.IsLoggedOut() {
		return errors.New("not logged in")
	}

	if a.IsAPIKey() {
		err := a.tryAuthorize(fmt.Sprintf("apikey:%d"+a.Data.(models.APIKey).Key), permissionObeject, permissionAction)
		if err != nil {
			return err
		}
	}

	subjects := []string{
		fmt.Sprintf("user:%d", a.User.ID),
		fmt.Sprintf("role:%s", a.User.Role),
	}

	for _, subject := range subjects {
		err := a.tryAuthorize(subject, permissionObeject, permissionAction)
		if err != nil {
			return err
		}
	}

	return nil
}

func SignInAuthentication(user models.User, c *fiber.Ctx) Authentication {
	return Authentication{
		Authenticator: AUTHENTICATOR_USER,
		User:          user,
		Enforcer:      GetEnforcer(c),
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

	// Get the token from the request header
	authentication, err := func() (Authentication, error) {
		authentication := Authentication{
			Authenticator: AUTHENTICATOR_LOGGED_OUT,
			Enforcer:      GetEnforcer(c),
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
			Enforcer:      GetEnforcer(c),
		}

		return authentication, nil
	}()

	if err != nil {
		// Get the api key from the request header
		authentication, err = func() (Authentication, error) {
			authentication := Authentication{
				Authenticator: AUTHENTICATOR_LOGGED_OUT,
				Enforcer:      GetEnforcer(c),
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
				Enforcer:      GetEnforcer(c),
			}

			return authentication, nil
		}()

		if err != nil {
			authentication = Authentication{
				Authenticator: AUTHENTICATOR_LOGGED_OUT,
				Enforcer:      GetEnforcer(c),
			}
		}
	}

	c.Locals(ctxAuthKey{}, authentication)
	return c.Next()
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

func InitalizeCasbin(db *gorm.DB) *casbin.Enforcer {
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		log.Fatalf("error: adapter: %s", err)
	}

	model, err := model.NewModelFromString(`
	[request_definition]
	r = sub, obj, act

	[policy_definition]
	p = sub, obj, act

	[policy_effect]
	e = some(where (p.eft == allow))

	[matchers]
	m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
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
