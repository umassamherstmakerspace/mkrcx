package leash_helpers

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	leash_api "github.com/mkrcx/mkrcx/src/leash/api"
	leash_signin "github.com/mkrcx/mkrcx/src/leash/signin"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

// SetupCasbin sets up the casbin RBAC for Leash
func SetupCasbin(enforcer *casbin.SyncedEnforcer) error {
	var setupErr error
	check := func(_ bool, err error) {
		if setupErr == nil && err != nil {
			setupErr = err
		}
	}
	// Roles
	member := "leash:member"
	volunteer := "leash:volunteer"
	staff := "leash:staff"
	admin := "leash:admin"

	// Delete Leash permission roles
	check(enforcer.DeleteRole(member))
	check(enforcer.DeleteRole(volunteer))
	check(enforcer.DeleteRole(staff))
	check(enforcer.DeleteRole(admin))

	// Create Leash permission role hierarchy
	check(enforcer.AddRoleForUser(admin, staff))
	check(enforcer.AddRoleForUser(staff, volunteer))
	check(enforcer.AddRoleForUser(volunteer, member))

	// Link Leash permission roles to mkr.cx roles
	check(enforcer.AddRoleForUser("role:admin", "leash:admin"))
	check(enforcer.AddRoleForUser("role:staff", "leash:staff"))
	check(enforcer.AddRoleForUser("role:volunteer", "leash:volunteer"))
	check(enforcer.AddRoleForUser("role:member", "leash:member"))

	// User Target Permissions
	check(enforcer.AddPermissionForUser(member, "leash.users:target_self"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users:target_others"))

	// User Base EPs
	check(enforcer.AddPermissionForUser(admin, "leash.users:create"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.service:create"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users:search"))

	// User Get EPs
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.get:email"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.get:card"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.get:checkin"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.get.trainings:list"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.get.holds:list"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.get.apikeys:list"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.get.updates:list"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.get.notifications:list"))

	// Self EPs
	check(enforcer.AddPermissionForUser(member, "leash.users.self:get"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self:update"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.self:update_card_id"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.self:update_role"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.self:service_update"))
	// --No self delete EP--
	check(enforcer.AddPermissionForUser(member, "leash.users.self:checkin"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self:permissions"))
	//   Updates
	check(enforcer.AddPermissionForUser(member, "leash.users.self.updates:list"))
	//   Trainings
	check(enforcer.AddPermissionForUser(member, "leash.users.self.trainings:target"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.trainings:list"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.trainings:get"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.self.trainings:create"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.self.trainings:delete"))
	//   Holds
	check(enforcer.AddPermissionForUser(member, "leash.users.self.holds:target"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.holds:list"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.self.holds:create"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.holds:get"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.self.holds:delete"))
	//   API Keys
	check(enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:target"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:list"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:create"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:get"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:update"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:delete"))
	//   Notifications
	check(enforcer.AddPermissionForUser(member, "leash.users.self.notifications:target"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.notifications:list"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.notifications:get"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.notifications:delete"))
	check(enforcer.AddPermissionForUser(member, "leash.users.self.notifications:create"))

	// Others EPs
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others:get"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others:update"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others:update_card_id"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others:update_role"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others:service_update"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others:delete"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others:checkin"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others:permissions"))
	//   Updates
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.updates:list"))
	//   Trainings
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.trainings:target"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.trainings:list"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.trainings:get"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.trainings:create"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.trainings:delete"))
	//   Holds
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.holds:target"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.holds:list"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.holds:create"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.holds:get"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.holds:delete"))
	//   API Keys
	check(enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:target"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:list"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:create"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:get"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:delete"))
	check(enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:update"))
	//   Notifications
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.notifications:target"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.notifications:list"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.notifications:get"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.notifications:delete"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.users.others.notifications:create"))

	// Training EPs
	check(enforcer.AddPermissionForUser(volunteer, "leash.trainings:target"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.trainings:get"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.trainings:delete"))

	// Hold EPs
	check(enforcer.AddPermissionForUser(volunteer, "leash.holds:target"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.holds:get"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.holds:delete"))

	// API Key EPs
	check(enforcer.AddPermissionForUser(admin, "leash.apikeys:target"))
	check(enforcer.AddPermissionForUser(admin, "leash.apikeys:get"))
	check(enforcer.AddPermissionForUser(admin, "leash.apikeys:delete"))
	check(enforcer.AddPermissionForUser(admin, "leash.apikeys:update"))

	// Notification EPs

	check(enforcer.AddPermissionForUser(volunteer, "leash.notifications:get"))
	check(enforcer.AddPermissionForUser(volunteer, "leash.notifications:delete"))

	// Sign In EPs
	check(enforcer.AddPermissionForUser(member, "leash:login"))

	// Feed administration and broad staff-reader permissions. Service users can
	// instead receive a least-privilege feed-specific permission such as
	// leash.feeds.signin:append.
	check(enforcer.AddPermissionForUser(staff, "leash.feeds:list"))
	check(enforcer.AddPermissionForUser(staff, "leash.feeds:read"))
	check(enforcer.AddPermissionForUser(admin, "leash.feeds:create"))
	check(enforcer.AddPermissionForUser(admin, "leash.feeds:delete"))
	check(enforcer.AddPermissionForUser(admin, "leash.feeds:manage"))
	check(enforcer.AddPermissionForUser(admin, "leash.feeds:append"))

	if setupErr != nil {
		return fmt.Errorf("configure Casbin policy: %w", setupErr)
	}
	models.SetupEnforcer(enforcer)
	return nil
}

func MigrateSchema(db *gorm.DB) error {
	err := models.SetupValidator()
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.User{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.APIKey{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.Training{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.UserUpdate{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.Hold{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.Session{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.Notification{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.Feed{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.FeedMessage{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.CheckinIdentity{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.CheckinEvent{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&models.CheckinExportAudit{})
	if err != nil {
		return err
	}

	return nil
}

func SetupMiddlewares(app *fiber.App, db *gorm.DB, keys *leash_auth.Keys, hmacSecret []byte, externalAuth leash_auth.ExternalAuthenticator, enforcer *casbin.SyncedEnforcer) {
	// Allow all origins in development
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, Idempotency-Key",
		AllowMethods: "*",
	}))

	app.Use(leash_auth.LocalsMiddleware(db, keys, hmacSecret, externalAuth, enforcer))
}

func SetupRoutes(app *fiber.App, feedRuntime ...*leash_api.FeedRuntime) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Welcome to the Leash!")
	})

	api := app.Group("/api", leash_auth.SetPermissionPrefixMiddleware("leash"))

	leash_api.RegisterAPIEndpoints(api, feedRuntime...)

	auth := app.Group("/auth")

	leash_signin.RegisterAuthenticationEndpoints(auth)
}
