package leash_backend_api

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/disgoorg/log"
	"github.com/gofiber/fiber/v2"
	"github.com/lestrrat-go/jwx/v2/jwt"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

type userGetRequest struct {
	WithTrainings     *bool `query:"with_trainings" validate:"omitempty"`
	WithHolds         *bool `query:"with_holds" validate:"omitempty"`
	WithApiKeys       *bool `query:"with_api_keys" validate:"omitempty"`
	WithUpdates       *bool `query:"with_updates" validate:"omitempty"`
	WithNotifications *bool `query:"with_notifications" validate:"omitempty"`
}

// Preload preloads the user with the specified fields
func (req *userGetRequest) Preload(c *fiber.Ctx, db *gorm.DB, user *models.User) error {
	prefix := c.Locals("permission_prefix").(string)
	authenticator := leash_auth.GetAuthentication(c)

	auth := func(item string) bool {
		return authenticator.Authorize(prefix+"."+item+":list") == nil
	}

	if req.WithTrainings != nil && *req.WithTrainings {
		if auth("trainings") {
			user.Trainings = []models.Training{}
			db.Model(&user).Association("Trainings").Find(&user.Trainings)
		} else {
			return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to view trainings")
		}
	}

	if req.WithHolds != nil && *req.WithHolds {
		if auth("holds") {
			user.Holds = []models.Hold{}
			db.Model(&user).Association("Holds").Find(&user.Holds)
		} else {
			return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to view holds")
		}
	}

	if req.WithApiKeys != nil && *req.WithApiKeys {
		if auth("apikeys") {
			user.APIKeys = []models.APIKey{}
			db.Model(&user).Association("APIKeys").Find(&user.APIKeys)
		} else {
			return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to view api keys")
		}
	}

	if req.WithUpdates != nil && *req.WithUpdates {
		if auth("updates") {
			user.UserUpdates = []models.UserUpdate{}
			db.Model(&user).Association("UserUpdates").Find(&user.UserUpdates)
		} else {
			return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to view updates")
		}
	}

	if req.WithNotifications != nil && *req.WithNotifications {
		if auth("notifications") {
			user.Notifications = []models.Notification{}
			db.Model(&user).Association("Notifications").Find(&user.Notifications)
		} else {
			return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to view notifications")
		}
	}

	return nil
}

var userCreateCallbacks []func(UserEvent)
var userUpdateCallbacks []func(UserUpdateEvent)
var userDeleteCallbacks []func(UserEvent)

// searchEmail searches for a user by email or pending email
func searchEmail(db *gorm.DB, email string) (models.User, error) {
	var user models.User

	res := db.Limit(1).Where(&models.User{Email: email}).Or(&models.User{PendingEmail: &email}).Find(&user)

	if res.Error != nil || res.RowsAffected == 0 {
		return user, errors.New("user not found")
	}

	return user, nil
}

// selfMiddleware is a middleware that sets the target user to the current user
func selfMiddleware(c *fiber.Ctx) error {
	authentication := leash_auth.GetAuthentication(c)
	// Check if the user is authorized to perform the action
	if authentication.Authorize("leash.users:target_self") != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to target yourself")
	}

	// Get the user from the authentication
	apiUser := authentication.User

	c.Locals("target_user", apiUser)
	return c.Next()
}

// userMiddleware is a middleware that sets the target user to the user specified in the URL
func userMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	authentication := leash_auth.GetAuthentication(c)
	// Check if the user is authorized to perform the action
	if authentication.Authorize("leash.users:target_others") != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to target other users")
	}

	// Get the user ID from the URL
	user_id, err := strconv.Atoi(c.Params("user_id"))

	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}

	var user = models.User{
		ID: uint(user_id),
	}

	if res := db.Limit(1).Where(&user).Find(&user); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	c.Locals("target_user", user)
	return c.Next()
}

// noServiceMiddleware is a middleware that prevents targeting service accounts
func noServiceMiddleware(c *fiber.Ctx) error {
	user := c.Locals("target_user").(models.User)

	if user.Role == "service" {
		return fiber.NewError(fiber.StatusNotAcceptable, "This endpoint cannot target service accounts")
	}

	return c.Next()
}

// onlyServiceMiddleware is a middleware that prevents targeting non-service accounts
func onlyServiceMiddleware(c *fiber.Ctx) error {
	user := c.Locals("target_user").(models.User)

	if user.Role != "service" {
		return fiber.NewError(fiber.StatusNotAcceptable, "This endpoint can only target service accounts")
	}

	return c.Next()
}

// createBaseEndpoints creates the common endpoints for the base user endpoint
func createBaseEndpoints(users_ep fiber.Router) {
	// Create a new user endpoint
	type userCreateRequest struct {
		Email          string `json:"email" xml:"email" form:"email" validate:"required,email"`
		Name           string `json:"name" xml:"name" form:"name" validate:"required"`
		Role           string `json:"role" xml:"role" form:"role" validate:"required,oneof=member volunteer staff admin"`
		Type           string `json:"type" xml:"type" form:"type" validate:"required,oneof=undergrad grad faculty staff alumni other"`
		GraduationYear int    `json:"graduation_year" xml:"graduation_year" form:"graduation_year" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
		Major          string `json:"major" xml:"major" form:"major" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
	}
	users_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[userCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		req := c.Locals("body").(userCreateRequest)

		// Check if the user already exists
		_, err := searchEmail(db, req.Email)
		if err == nil {
			return fiber.NewError(fiber.StatusConflict, "User already exists")
		}

		// Create a new user in the database
		user := models.User{
			Email:          req.Email,
			Name:           req.Name,
			Role:           req.Role,
			Type:           req.Type,
			GraduationYear: req.GraduationYear,
			Major:          req.Major,
		}

		db.Create(&user)

		event := UserEvent{
			c:         c,
			Target:    user,
			Agent:     leash_auth.GetAuthentication(c).User,
			Timestamp: time.Now().Unix(),
		}

		for _, callback := range userCreateCallbacks {
			callback(event)
		}

		return c.JSON(user)
	})

	// Search for a user by name or email endpoint
	type userSearchQuery struct {
		listRequest
		userGetRequest
		Query       *string `query:"query" validate:"required"`
		ShowService *bool   `query:"show_service" validate:"omitempty"`
	}
	users_ep.Get("/search", leash_auth.PrefixAuthorizationMiddleware("search"), leash_auth.AuthorizationMiddleware("leash.users:target_others"), models.GetQueryMiddleware[userSearchQuery], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		req := c.Locals("query").(userSearchQuery)
		authenticator := leash_auth.GetAuthentication(c)

		var users []models.User

		showService := req.ShowService != nil && *req.ShowService

		authorizeList := func(item string) bool {
			return authenticator.Authorize("leash.users.others."+item+":list") == nil
		}

		con := db.Model(&models.User{})

		// Preload the user with the specified fields
		if req.WithTrainings != nil && *req.WithTrainings {
			if authorizeList("trainings") {
				con = con.Preload("Trainings")
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		if req.WithHolds != nil && *req.WithHolds {
			if authorizeList("holds") {
				con = con.Preload("Holds")
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		if req.WithApiKeys != nil && *req.WithApiKeys {
			if authorizeList("apikeys") {
				con = con.Preload("APIKeys")
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		if req.WithUpdates != nil && *req.WithUpdates {
			if authorizeList("updates") {
				con = con.Preload("UserUpdates")
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		if req.WithNotifications != nil && *req.WithNotifications {
			if authorizeList("notifications") {
				con = con.Preload("Notifications")
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		q := db.Where("name LIKE ?", "%"+*req.Query+"%").Or("email LIKE ?", "%"+*req.Query+"%").Or("pending_email LIKE ?", "%"+*req.Query+"%")

		// Allow searching for service accounts
		if showService {
			con = con.Where(q)
		} else {
			con = con.Where("role <> ?", "service").Where(q)
		}

		// Count the total number of users
		total := int64(0)
		con.Model(&models.User{}).Count(&total)

		// Paginate the results
		if req.Limit != nil {
			con = con.Limit(*req.Limit)
		} else {
			con = con.Limit(10)
		}

		if req.Offset != nil {
			con = con.Offset(*req.Offset)
		} else {
			con = con.Offset(0)
		}

		con.Find(&users)

		response := struct {
			Data  []models.User `json:"data"`
			Total int64         `json:"total"`
		}{
			Data:  users,
			Total: total,
		}

		return c.JSON(response)
	})
}

// createGetUserEndpoints creates the endpoints for getting users
func createGetUserEndpoints(get_ep fiber.Router) {
	// Get a user by email endpoint
	get_ep.Get("/email/:email", leash_auth.PrefixAuthorizationMiddleware("email"), models.GetQueryMiddleware[userGetRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		email := c.Params("email")
		user, err := searchEmail(db, email)

		// Check if the user exists
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Preload the user with the specified fields
		req := c.Locals("query").(userGetRequest)
		if err := req.Preload(c, db, &user); err != nil {
			return err
		}

		return c.JSON(user)
	})

	get_ep.Get("/card/:card", leash_auth.PrefixAuthorizationMiddleware("card"), models.GetQueryMiddleware[userGetRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		card_id := c.Params("card")

		// Check if the user exists
		var user = models.User{
			CardID: &card_id,
		}

		if res := db.Limit(1).Where(&user).Find(&user); res.Error != nil || res.RowsAffected == 0 {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Preload the user with the specified fields
		req := c.Locals("query").(userGetRequest)
		if err := req.Preload(c, db, &user); err != nil {
			return err
		}

		return c.JSON(user)
	})

	// Get a user by checkin token endpoint
	get_ep.Get("/checkin/:token", leash_auth.PrefixAuthorizationMiddleware("checkin"), models.GetQueryMiddleware[userGetRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		keys := leash_auth.GetKeys(c)
		token := c.Params("token")

		// Parse the token
		tok, err := keys.Parse(token, []string{"leash", "checkin"})
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid token")
		}

		// Get the user ID from the token
		val := tok.Subject()

		user_id, err := strconv.Atoi(val)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid token")
		}

		// Check if the user exists
		var user = models.User{
			ID: uint(user_id),
		}

		if res := db.Limit(1).Where(&user).Find(&user); res.Error != nil || res.RowsAffected == 0 {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Preload the user with the specified fields
		req := c.Locals("query").(userGetRequest)
		if err := req.Preload(c, db, &user); err != nil {
			return err
		}

		return c.JSON(user)
	})
}

// addUserUpdateEndpoints creates the endpoints for user updates
func addUserUpdateEndpoints(user_ep fiber.Router) {
	// Create a new user update endpoint group
	update_ep := user_ep.Group("/updates", leash_auth.ConcatPermissionPrefixMiddleware("updates"))

	// List user updates endpoint
	update_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), models.GetQueryMiddleware[listRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		req := c.Locals("query").(listRequest)

		// Count the total number of users
		total := db.Model(user).Association("UserUpdates").Count()

		// Paginate the results
		var updates []models.UserUpdate

		con := db
		if req.IncludeDeleted != nil && *req.IncludeDeleted {
			con = con.Unscoped()
		}

		con = con.Model(&updates).Where(models.UserUpdate{UserID: user.ID})
		if req.Limit != nil {
			con = con.Limit(*req.Limit)
		} else {
			con = con.Limit(10)
		}

		if req.Offset != nil {
			con = con.Offset(*req.Offset)
		} else {
			con = con.Offset(0)
		}

		con.Find(&updates)

		response := struct {
			Data  []models.UserUpdate `json:"data"`
			Total int64               `json:"total"`
		}{
			Data:  updates,
			Total: total,
		}

		return c.JSON(response)
	})
}

func getUserEndpoint(user_ep fiber.Router) {
	// Get the current user endpoint
	user_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), models.GetQueryMiddleware[userGetRequest], func(c *fiber.Ctx) error {
		req := c.Locals("query").(userGetRequest)
		user := c.Locals("target_user").(models.User)

		// Preload the user with the specified fields
		if err := req.Preload(c, leash_auth.GetDB(c), &user); err != nil {
			return err
		}
		return c.JSON(user)
	})
}

// updateUserEndpoints creates the endpoints for updating users
func updateUserEndpoint(user_ep fiber.Router) {
	// Update the current user endpoint
	type userUpdateRequest struct {
		Name           *string `json:"name" xml:"name" form:"name" validate:"omitempty"`
		Email          *string `json:"email" xml:"email" form:"email" validate:"omitempty,email"`
		CardId         *string `json:"card_id" xml:"card_id" form:"card_id" validate:"omitempty"`
		Role           *string `json:"role" xml:"role" form:"role" validate:"omitempty,oneof=member volunteer staff admin"`
		Type           *string `json:"type" xml:"type" form:"type" validate:"omitempty,oneof=undergrad grad faculty staff alumni other"`
		GraduationYear *int    `json:"graduation_year" xml:"graduation_year" form:"graduation_year" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
		Major          *string `json:"major" xml:"major" form:"major" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
	}
	user_ep.Patch("/", leash_auth.PrefixAuthorizationMiddleware("update"), noServiceMiddleware, models.GetBodyMiddleware[userUpdateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		req := c.Locals("body").(userUpdateRequest)
		user := c.Locals("target_user").(models.User)

		authenticator := leash_auth.GetAuthentication(c)
		permissionPrefix := c.Locals("permission_prefix").(string)

		// Base for the update event
		event := UserUpdateEvent{
			UserEvent: UserEvent{
				c:         c,
				Target:    user,
				Agent:     authenticator.User,
				Timestamp: time.Now().Unix(),
			},
			Changes: []UserChanges{},
		}

		// Helper function to check if a field has been modified and add it to the event
		modified := func(original string, new *string, field string) bool {
			if new == nil {
				return false
			}

			if original != *new {
				event.Changes = append(event.Changes, UserChanges{
					Old:   original,
					New:   *new,
					Field: field,
				})
				return true
			}

			return false
		}

		// Update fields
		if modified(user.Name, req.Name, "name") {
			user.Name = *req.Name
		}

		// Check if the email has been changed
		if req.Email != nil {
			if *req.Email != user.Email && (user.PendingEmail == nil || *req.Email != *user.PendingEmail) {
				_, err := searchEmail(db, *req.Email)
				if err == nil {
					// The user already exists
					return c.Status(fiber.StatusConflict).SendString("Email already in use")
				}

				old := ""

				if user.PendingEmail != nil {
					old = *user.PendingEmail
				}

				event.Changes = append(event.Changes, UserChanges{
					Old:   old,
					New:   *req.Email,
					Field: "pending_email",
				})

				user.PendingEmail = req.Email
			} else if user.PendingEmail != nil && *req.Email == user.Email {
				event.Changes = append(event.Changes, UserChanges{
					Old:   *user.PendingEmail,
					New:   "",
					Field: "pending_email",
				})

				user.PendingEmail = nil
			}
		}

		if modified(user.Type, req.Type, "type") {
			user.Type = *req.Type
		}

		var graduationYear *string
		if req.GraduationYear != nil {
			graduationYear = new(string)
			*graduationYear = fmt.Sprintf("%d", *req.GraduationYear)
		}

		if modified(fmt.Sprint(user.GraduationYear), graduationYear, "graduation_year") {
			user.GraduationYear = *req.GraduationYear
		}

		if modified(user.Major, req.Major, "major") {
			user.Major = *req.Major
		}

		if req.CardId != nil {
			if authenticator.Authorize(permissionPrefix+":update_card_id") == nil {
				changed := false
				old := ""
				new := ""

				if user.CardID == nil {
					if *req.CardId != "" {
						user.CardID = req.CardId
						changed = true
						new = *req.CardId
					}
				} else {
					if *req.CardId == "" {
						user.CardID = nil
						changed = true
						old = *user.CardID
					} else if *req.CardId != *user.CardID {
						old = *user.CardID
						user.CardID = req.CardId
						changed = true
						new = *req.CardId
					}
				}

				if changed {
					event.Changes = append(event.Changes, UserChanges{
						Old:   old,
						New:   new,
						Field: "card_id",
					})

					db.Save(&user)
				}
			} else {
				return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to update the card ID")
			}
		}

		if req.Role != nil {
			if authenticator.Authorize(permissionPrefix+":update_role") == nil {
				if *req.Role != user.Role {
					event.Changes = append(event.Changes, UserChanges{
						Old:   user.Role,
						New:   *req.Role,
						Field: "role",
					})
					user.Role = *req.Role
				}
			} else {
				return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to update the role")
			}
		}

		db.Save(&user)

		// Run the update callbacks
		for _, callback := range userUpdateCallbacks {
			callback(event)
		}

		return c.JSON(user)
	})
}

// deleteUserEndpoints creates the endpoints for deleting users
func deleteUserEndpoint(user_ep fiber.Router) {
	user_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		user := c.Locals("target_user").(models.User)

		event := UserEvent{
			c:         c,
			Target:    user,
			Agent:     leash_auth.GetAuthentication(c).User,
			Timestamp: time.Now().Unix(),
		}

		for _, callback := range userDeleteCallbacks {
			callback(event)
		}

		return c.SendStatus(fiber.StatusOK)
	})
}

// checkinUserEndpoint creates a JWT that lasts 2 minutes for checking in users
func checkinUserEndpoint(user_ep fiber.Router) {
	user_ep.Get("/checkin", leash_auth.PrefixAuthorizationMiddleware("checkin"), func(c *fiber.Ctx) error {
		user := c.Locals("target_user").(models.User)
		keys := leash_auth.GetKeys(c)

		tok, err := jwt.NewBuilder().
			Issuer(leash_auth.ISSUER).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(2 * time.Minute)).
			Subject(fmt.Sprintf("%d", user.ID)).
			Audience([]string{"leash", "checkin"}).
			Build()

		if err != nil {
			log.Error("Failed to build the checkin token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		signed, err := keys.Sign(tok)
		if err != nil {
			log.Error("Failed to sign the checkin token: %s\n", err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		token := struct {
			Token     string `json:"token"`
			ExpiresAt int64  `json:"expires_at"`
		}{
			Token:     string(signed),
			ExpiresAt: tok.Expiration().Unix(),
		}

		return c.JSON(token)
	})
}

// getPermissionsEndpoint returns all the permissions for a user including inherited permissions
func getPermissionsEndpoint(user_ep fiber.Router) {
	user_ep.Get("/permissions", leash_auth.PrefixAuthorizationMiddleware("permissions"), func(c *fiber.Ctx) error {
		user := c.Locals("target_user").(models.User)
		authenticator := leash_auth.GetAuthentication(c)

		user_permissions := authenticator.Enforcer.Enforcer.GetPermissionsForUser("user:" + fmt.Sprint(user.ID))
		role_permissions := authenticator.Enforcer.Enforcer.GetPermissionsForUser("role:" + user.Role)

		permissions := make([]string, len(user_permissions)+len(role_permissions))

		for i, perm := range user_permissions {
			permissions[i] = perm[1]
		}

		for i, perm := range role_permissions {
			permissions[i+len(user_permissions)] = perm[1]
		}

		return c.JSON(permissions)
	})
}

// serviceCreateEndpoint creates the service user creation endpoint
func serviceCreateEndpoint(service_ep fiber.Router) {
	// Create a new service user endpoint
	type serviceUserCreateRequest struct {
		Name        string   `json:"name" xml:"name" form:"name" validate:"required"`
		ServiceTag  string   `json:"service_tag" xml:"service_tag" form:"service_tag" validate:"required"`
		Permissions []string `json:"permissions" xml:"permissions" form:"permissions" validate:"required"`
	}
	service_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[serviceUserCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		req := c.Locals("body").(serviceUserCreateRequest)

		// Create a new user in the database
		user := models.User{
			Name:        req.Name,
			Role:        "service",
			Type:        "other",
			Email:       req.ServiceTag + "@mkrcx",
			Permissions: req.Permissions,
		}

		db.Create(&user)

		// Set the user's permissions in the RBAC
		enforcer := leash_auth.GetAuthentication(c).Enforcer

		enforcer.SetPermissionsForUser(user, req.Permissions)
		enforcer.SavePolicy()

		event := UserEvent{
			c:         c,
			Target:    user,
			Agent:     leash_auth.GetAuthentication(c).User,
			Timestamp: time.Now().Unix(),
		}

		for _, callback := range userCreateCallbacks {
			callback(event)
		}

		return c.JSON(user)
	})
}

// updateServiceEndpoint creates the service user editing endpoint
func updateServiceEndpoint(user_ep fiber.Router) {
	// Update the current service user endpoint
	type serviceUserUpdateRequest struct {
		Name        *string   `json:"name" xml:"name" form:"name" validate:"omitempty"`
		Permissions *[]string `json:"permissions" xml:"permissions" form:"permissions" validate:"omitempty"`
		ServiceTag  *string   `json:"service_tag" xml:"service_tag" form:"service_tag" validate:"omitempty"`
	}

	user_ep.Patch("/service", leash_auth.PrefixAuthorizationMiddleware("service_update"), onlyServiceMiddleware, models.GetBodyMiddleware[serviceUserUpdateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		req := c.Locals("body").(serviceUserUpdateRequest)
		user := c.Locals("target_user").(models.User)

		authenticator := leash_auth.GetAuthentication(c)

		// Base for the update event
		event := UserUpdateEvent{
			UserEvent: UserEvent{
				c:         c,
				Target:    user,
				Agent:     authenticator.User,
				Timestamp: time.Now().Unix(),
			},

			Changes: []UserChanges{},
		}

		if req.Name != nil && *req.Name != user.Name {
			event.Changes = append(event.Changes, UserChanges{
				Old:   user.Name,
				New:   *req.Name,
				Field: "name",
			})

			user.Name = *req.Name
		}

		if req.Permissions != nil {
			enforcer := leash_auth.GetAuthentication(c).Enforcer

			enforcer.SetPermissionsForUser(user, *req.Permissions)
			enforcer.SavePolicy()
			user.Permissions = *req.Permissions
		}

		if req.ServiceTag != nil {
			serviceTag := *req.ServiceTag + "@mkrcx"
			if serviceTag != user.Email {
				event.Changes = append(event.Changes, UserChanges{
					Old:   user.Email,
					New:   serviceTag,
					Field: "email",
				})

				user.Email = serviceTag
			}
		}

		db.Save(&user)

		// Run the update callbacks
		for _, callback := range userUpdateCallbacks {
			callback(event)
		}

		return c.JSON(user)
	})
}

// registerUserEndpoints registers all the User endpoints for Leash
func registerUserEndpoints(api fiber.Router) {
	users_ep := api.Group("/users", leash_auth.ConcatPermissionPrefixMiddleware("users"))

	userCreateCallbacks = []func(UserEvent){}
	userUpdateCallbacks = []func(UserUpdateEvent){}
	userDeleteCallbacks = []func(UserEvent){}

	// Register a callback to add user updates to the database
	OnUserUpdate(func(event UserUpdateEvent) {
		for _, change := range event.Changes {
			db := leash_auth.GetDB(event.GetCtx())
			update := models.UserUpdate{
				Field:    change.Field,
				NewValue: change.New,
				OldValue: change.Old,
				UserID:   event.Target.ID,
				EditedBy: event.Agent.ID,
			}

			db.Create(&update)
		}
	})

	createBaseEndpoints(users_ep)

	get_ep := users_ep.Group("/get", leash_auth.ConcatPermissionPrefixMiddleware("get"))
	createGetUserEndpoints(get_ep)

	service_ep := users_ep.Group("/service", leash_auth.ConcatPermissionPrefixMiddleware("service"))
	serviceCreateEndpoint(service_ep)

	self_ep := users_ep.Group("/self", leash_auth.ConcatPermissionPrefixMiddleware("self"), selfMiddleware)
	getUserEndpoint(self_ep)
	updateUserEndpoint(self_ep)
	updateServiceEndpoint(self_ep)
	checkinUserEndpoint(self_ep)
	getPermissionsEndpoint(self_ep)
	addUserUpdateEndpoints(self_ep)
	addUserTrainingEndpoints(self_ep)
	addUserHoldsEndpoints(self_ep)
	addUserApiKeyEndpoints(self_ep)
	addUserNotificationsEndpoints(self_ep)

	user_ep := users_ep.Group("/:user_id", leash_auth.ConcatPermissionPrefixMiddleware("others"), userMiddleware)
	getUserEndpoint(user_ep)
	updateUserEndpoint(user_ep)
	updateServiceEndpoint(user_ep)
	deleteUserEndpoint(user_ep)
	checkinUserEndpoint(user_ep)
	getPermissionsEndpoint(user_ep)
	addUserUpdateEndpoints(user_ep)
	addUserTrainingEndpoints(user_ep)
	addUserHoldsEndpoints(user_ep)
	addUserApiKeyEndpoints(user_ep)
	addUserNotificationsEndpoints(user_ep)
}

// OnUserCreate registers a callback to be called when a user is created
func OnUserCreate(callback func(UserEvent)) {
	userCreateCallbacks = append(userCreateCallbacks, callback)
}

// OnUserUpdate registers a callback to be called when a user is updated
func OnUserUpdate(callback func(UserUpdateEvent)) {
	userUpdateCallbacks = append(userUpdateCallbacks, callback)
}

// OnUserDelete registers a callback to be called when a user is deleted
func OnUserDelete(callback func(UserEvent)) {
	userDeleteCallbacks = append(userDeleteCallbacks, callback)
}

// UpdateEmail sets the pending email for a user as their email
func UpdatePendingEmail(user models.User, c *fiber.Ctx) (models.User, error) {
	db := leash_auth.GetDB(c)

	if user.PendingEmail == nil {
		return user, errors.New("no pending email")
	}

	// If a user has a pending email, update their email
	event := UserUpdateEvent{
		UserEvent: UserEvent{
			c:         c,
			Target:    user,
			Agent:     user,
			Timestamp: time.Now().Unix(),
		},
		Changes: []UserChanges{
			{
				Old:   user.Email,
				New:   *user.PendingEmail,
				Field: "email",
			},
			{
				Old:   *user.PendingEmail,
				New:   "",
				Field: "pending_email",
			},
		},
	}

	user.Email = *user.PendingEmail
	user.PendingEmail = nil
	db.Save(&user)

	// Run the update callbacks
	for _, callback := range userUpdateCallbacks {
		callback(event)
	}

	return user, nil
}
