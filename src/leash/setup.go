package main

import (
	"github.com/casbin/casbin/v2"
)

func setupCasbin(enforcer *casbin.Enforcer) {
	//member volunteer staff admin

	member := "role:member"
	volunteer := "role:volunteer"
	staff := "role:staff"
	admin := "role:admin"

	enforcer.DeleteRole(member)
	enforcer.DeleteRole(volunteer)
	enforcer.DeleteRole(staff)
	enforcer.DeleteRole(admin)

	enforcer.AddRoleForUser(admin, staff)
	enforcer.AddRoleForUser(staff, volunteer)
	enforcer.AddRoleForUser(volunteer, member)

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
}
