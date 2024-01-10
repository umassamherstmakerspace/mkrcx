package leash_helpers

import (
	"os"

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
func SetupCasbin(enforcer *casbin.Enforcer) {
	// Roles
	member := "leash:member"
	volunteer := "leash:volunteer"
	staff := "leash:staff"
	admin := "leash:admin"

	// Delete Leash permission roles
	enforcer.DeleteRole(member)
	enforcer.DeleteRole(volunteer)
	enforcer.DeleteRole(staff)
	enforcer.DeleteRole(admin)

	// Create Leash permission role hierarchy
	enforcer.AddRoleForUser(admin, staff)
	enforcer.AddRoleForUser(staff, volunteer)
	enforcer.AddRoleForUser(volunteer, member)

	// Link Leash permission roles to mkr.cx roles
	enforcer.AddRoleForUser("role:admin", "leash:admin")
	enforcer.AddRoleForUser("role:staff", "leash:staff")
	enforcer.AddRoleForUser("role:volunteer", "leash:volunteer")
	enforcer.AddRoleForUser("role:member", "leash:member")

	// User Target Permissions
	enforcer.AddPermissionForUser(member, "leash.users:target_self")
	enforcer.AddPermissionForUser(volunteer, "leash.users:target_others")

	// User Base EPs
	enforcer.AddPermissionForUser(admin, "leash.users:create")
	enforcer.AddPermissionForUser(volunteer, "leash.users:search")

	// User Get EPs
	enforcer.AddPermissionForUser(volunteer, "leash.users.get:email")
	enforcer.AddPermissionForUser(admin, "leash.users.get:card")

	// Self EPs
	enforcer.AddPermissionForUser(member, "leash.users.self:read")
	enforcer.AddPermissionForUser(member, "leash.users.self:update")
	enforcer.AddPermissionForUser(admin, "leash.users.self:update_card_id")
	enforcer.AddPermissionForUser(admin, "leash.users.self:update_role")
	// --No self delete EP--
	//   Updates
	enforcer.AddPermissionForUser(member, "leash.users.self.updates:list")
	//   Trainings
	enforcer.AddPermissionForUser(member, "leash.users.self.trainings:list")
	enforcer.AddPermissionForUser(member, "leash.users.self.trainings:get")
	enforcer.AddPermissionForUser(volunteer, "leash.users.self.trainings:create")
	enforcer.AddPermissionForUser(volunteer, "leash.users.self.trainings:delete")
	//   Holds
	enforcer.AddPermissionForUser(member, "leash.users.self.holds:list")
	enforcer.AddPermissionForUser(volunteer, "leash.users.self.holds:create")
	enforcer.AddPermissionForUser(member, "leash.users.self.holds:get")
	enforcer.AddPermissionForUser(volunteer, "leash.users.self.holds:delete")
	//   API Keys
	enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:list")
	enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:create")
	enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:get")
	enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:update")
	enforcer.AddPermissionForUser(member, "leash.users.self.apikeys:delete")
	//   Notifications
	enforcer.AddPermissionForUser(member, "leash.users.self.notifications:list")
	enforcer.AddPermissionForUser(member, "leash.users.self.notifications:get")
	enforcer.AddPermissionForUser(member, "leash.users.self.notifications:delete")
	enforcer.AddPermissionForUser(member, "leash.users.self.notifications:create")

	// Others EPs
	enforcer.AddPermissionForUser(volunteer, "leash.users.others:read")
	enforcer.AddPermissionForUser(staff, "leash.users.others:update")
	enforcer.AddPermissionForUser(admin, "leash.users.others:update_card_id")
	enforcer.AddPermissionForUser(admin, "leash.users.others:update_role")
	enforcer.AddPermissionForUser(admin, "leash.users.others:delete")
	//   Updates
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.updates:list")
	//   Trainings
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.trainings:list")
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.trainings:get")
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.trainings:create")
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.trainings:delete")
	//   Holds
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.holds:list")
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.holds:create")
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.holds:get")
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.holds:delete")
	//   API Keys
	enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:list")
	enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:create")
	enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:get")
	enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:delete")
	enforcer.AddPermissionForUser(admin, "leash.users.others.apikeys:update")
	//   Notifications
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.notifications:list")
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.notifications:get")
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.notifications:delete")
	enforcer.AddPermissionForUser(volunteer, "leash.users.others.notifications:create")

	// Service EPs
	enforcer.AddPermissionForUser(volunteer, "leash.users.service:read")
	enforcer.AddPermissionForUser(admin, "leash.users.service:update")
	enforcer.AddPermissionForUser(admin, "leash.users.service:update_card_id")
	enforcer.AddPermissionForUser(admin, "leash.users.service:update_role")
	enforcer.AddPermissionForUser(admin, "leash.users.service:delete")
	//   Updates
	enforcer.AddPermissionForUser(volunteer, "leash.users.service.updates:list")
	//   Trainings
	enforcer.AddPermissionForUser(volunteer, "leash.users.service.trainings:list")
	enforcer.AddPermissionForUser(volunteer, "leash.users.service.trainings:get")
	enforcer.AddPermissionForUser(admin, "leash.users.service.trainings:create")
	enforcer.AddPermissionForUser(admin, "leash.users.service.trainings:delete")
	//   Holds
	enforcer.AddPermissionForUser(volunteer, "leash.users.service.holds:list")
	enforcer.AddPermissionForUser(admin, "leash.users.service.holds:create")
	enforcer.AddPermissionForUser(volunteer, "leash.users.service.holds:get")
	enforcer.AddPermissionForUser(admin, "leash.users.service.holds:delete")
	//   API Keys
	enforcer.AddPermissionForUser(admin, "leash.users.service.apikeys:list")
	enforcer.AddPermissionForUser(admin, "leash.users.service.apikeys:create")
	enforcer.AddPermissionForUser(admin, "leash.users.service.apikeys:get")
	enforcer.AddPermissionForUser(admin, "leash.users.service.apikeys:delete")
	enforcer.AddPermissionForUser(admin, "leash.users.service.apikeys:update")
	//   Notifications
	enforcer.AddPermissionForUser(volunteer, "leash.users.service.notifications:list")
	enforcer.AddPermissionForUser(volunteer, "leash.users.service.notifications:get")
	enforcer.AddPermissionForUser(admin, "leash.users.service.notifications:delete")
	enforcer.AddPermissionForUser(admin, "leash.users.service.notifications:create")

	// Training EPs
	enforcer.AddPermissionForUser(volunteer, "leash.trainings:get")
	enforcer.AddPermissionForUser(volunteer, "leash.trainings:delete")

	// Hold EPs
	enforcer.AddPermissionForUser(volunteer, "leash.holds:get")
	enforcer.AddPermissionForUser(volunteer, "leash.holds:delete")

	// API Key EPs
	enforcer.AddPermissionForUser(admin, "leash.apikeys:get")
	enforcer.AddPermissionForUser(admin, "leash.apikeys:delete")
	enforcer.AddPermissionForUser(admin, "leash.apikeys:update")

	// Notification EPs
	enforcer.AddPermissionForUser(volunteer, "leash.notifications:get")
	enforcer.AddPermissionForUser(volunteer, "leash.notifications:delete")

	// Sign In EPs
	enforcer.AddPermissionForUser(member, "leash:login")

	enforcer.SavePolicy()

	models.SetupEnforcer(enforcer)
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

	return nil
}

func SetupMiddlewares(app *fiber.App, db *gorm.DB, keys *leash_auth.Keys, externalAuth leash_auth.ExternalAuthenticator, enforcer *casbin.Enforcer) {
	// Allow all origins in development
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return os.Getenv("ENVIRONMENT") == "development"
		},
	}))

	app.Use(leash_auth.LocalsMiddleware(db, keys, externalAuth, enforcer))
}

func SetupRoutes(app *fiber.App) {
	api := app.Group("/api", leash_auth.SetPermissionPrefixMiddleware("leash"))

	leash_api.RegisterAPIEndpoints(api)

	auth := app.Group("/auth")

	leash_signin.RegisterAuthenticationEndpoints(auth)
}
