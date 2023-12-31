package leash_helpers

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

func SetupCasbin(enforcer *casbin.Enforcer) {
	member := "leash:member"
	volunteer := "leash:volunteer"
	staff := "leash:staff"
	admin := "leash:admin"

	enforcer.DeleteRole(member)
	enforcer.DeleteRole(volunteer)
	enforcer.DeleteRole(staff)
	enforcer.DeleteRole(admin)

	fmt.Println("Setting up casbin")

	enforcer.AddRoleForUser(admin, staff)
	enforcer.AddRoleForUser(staff, volunteer)
	enforcer.AddRoleForUser(volunteer, member)

	fmt.Println("Setting up permissions")

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

	// Sign In EPs
	enforcer.AddPermissionForUser(member, "leash:login")

	enforcer.SavePolicy()
}

func MigrateUserPermissions(db *gorm.DB, enforcer *casbin.Enforcer) {
	fmt.Println("Migrating user permissions")
	var users []models.User
	db.Find(&users)

	for _, user := range users {
		user_id := fmt.Sprintf("user:%d", user.ID)

		role := "role:member"
		if user.Role == "volunteer" {
			role = "role:volunteer"
		} else if user.Role == "staff" {
			role = "role:staff"
		} else if user.Role == "admin" {
			role = "role:admin"
		}

		val, err := enforcer.HasRoleForUser(user_id, role)
		if err != nil {
			panic(err)
		}

		if val {
			continue
		}

		enforcer.DeleteRolesForUser(user_id)
		enforcer.AddRoleForUser(user_id, role)
	}

	enforcer.SavePolicy()
}

func MigrateAPIKeyPermissions(db *gorm.DB, enforcer *casbin.Enforcer) {
	fmt.Println("Migrating apikey permissions")
	var apikeys []models.APIKey
	db.Find(&apikeys)

	for _, apikey := range apikeys {
		apikey_id := fmt.Sprintf("apikey:%s", apikey.Key)

		enforcer.DeletePermissionsForUser(apikey_id)

		permissions := strings.Split(apikey.Permissions, ",")
		for _, permission := range permissions {
			if permission != "" {
				enforcer.AddPermissionForUser(apikey_id, permission)
			}
		}
	}

	enforcer.SavePolicy()
}
