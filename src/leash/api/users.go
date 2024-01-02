package leash_backend_api

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

type userGetRequest struct {
	WithTrainings *bool `query:"with_trainings" validate:"omitempty"`
	WithHolds     *bool `query:"with_holds" validate:"omitempty"`
	WithApiKeys   *bool `query:"with_api_keys" validate:"omitempty"`
	WithUpdates   *bool `query:"with_updates" validate:"omitempty"`
}

// Preload preloads the user with the specified fields
func (req *userGetRequest) Preload(db *gorm.DB, user *models.User) {
	if req.WithTrainings != nil && *req.WithTrainings {
		user.Trainings = []models.Training{}
		db.Model(&user).Association("Trainings").Find(&user.Trainings)
	}

	if req.WithHolds != nil && *req.WithHolds {
		user.Holds = []models.Hold{}
		db.Model(&user).Association("Holds").Find(&user.Holds)
	}

	if req.WithApiKeys != nil && *req.WithApiKeys {
		user.APIKeys = []models.APIKey{}
		db.Model(&user).Association("APIKeys").Find(&user.APIKeys)
	}

	if req.WithUpdates != nil && *req.WithUpdates {
		user.UserUpdates = []models.UserUpdate{}
		db.Model(&user).Association("UserUpdates").Find(&user.UserUpdates)
	}
}

var userCreateCallbacks []func(UserEvent)
var userUpdateCallbacks []func(UserUpdateEvent)
var userDeleteCallbacks []func(UserEvent)

// searchEmail searches for a user by email or pending email
func searchEmail(db *gorm.DB, email string) (models.User, error) {
	var user models.User

	res := db.Limit(1).Where(&models.User{Email: email}).Or(&models.User{PendingEmail: email}).Find(&user)

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
		return c.SendStatus(fiber.StatusUnauthorized)
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
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	// Get the user ID from the URL
	user_id, err := strconv.Atoi(c.Params("user_id"))

	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user ID")
	}

	var user = models.User{}
	user.ID = uint(user_id)

	if res := db.Limit(1).Where(&user).Find(&user); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}

	c.Locals("target_user", user)
	return c.Next()
}

// createBaseEndpoints creates the common endpoints for the base user endpoint
func createBaseEndpoints(users_ep fiber.Router) {
	// Create a new user endpoint
	type userCreateRequest struct {
		Email    string `json:"email" xml:"email" form:"email" validate:"required,email"`
		Name     string `json:"name" xml:"name" form:"name" validate:"required"`
		Role     string `json:"role" xml:"role" form:"role" validate:"required,oneof=member volunteer staff admin"`
		Type     string `json:"type" xml:"type" form:"type" validate:"required,oneof=undergrad grad faculty staff alumni other"`
		GradYear int    `json:"grad_year" xml:"grad_year" form:"grad_year" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
		Major    string `json:"major" xml:"major" form:"major" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
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
			GraduationYear: req.GradYear,
			Major:          req.Major,
			Enabled:        false,
		}

		db.Create(&user)

		// Set the user's role in the RBAC
		leash_auth.GetAuthentication(c).Enforcer.SetUserRole(user, req.Role)

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
	users_ep.Get("/search", leash_auth.PrefixAuthorizationMiddleware("search"), models.GetQueryMiddleware[userSearchQuery], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		req := c.Locals("query").(userSearchQuery)
		authenticator := leash_auth.GetAuthentication(c)

		var users []models.User

		con := db.Model(&models.User{})

		// Preload the user with the specified fields
		if req.WithTrainings != nil && *req.WithTrainings {
			if authenticator.Authorize("leash.users.others.trainings:list") == nil {
				con = con.Preload("Trainings")
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		if req.WithHolds != nil && *req.WithHolds {
			if authenticator.Authorize("leash.users.others.holds:list") == nil {
				con = con.Preload("Holds")
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		if req.WithApiKeys != nil && *req.WithApiKeys {
			if authenticator.Authorize("leash.users.others.apikeys:list") == nil {
				con = con.Preload("APIKeys")
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		if req.WithUpdates != nil && *req.WithUpdates {
			if authenticator.Authorize("leash.users.others.updates:list") == nil {
				con = con.Preload("UserUpdates")
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		q := db.Where("name LIKE ?", "%"+*req.Query+"%").Or("email LIKE ?", "%"+*req.Query+"%").Or("pending_email LIKE ?", "%"+*req.Query+"%")

		// Allow searching for service accounts
		if req.ShowService == nil || !*req.ShowService {
			con = con.Where("role <> ?", "service").Where(q)
		} else {
			con = con.Where(q)
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
			Users []models.User `json:"users"`
			Total int64         `json:"total"`
		}{
			Users: users,
			Total: total,
		}

		return c.JSON(response)
	})
}

// createGetUserEndpoints creates the endpoints for getting users
func createGetUserEndpoints(get_ep fiber.Router) {
	// Get a user by email endpoint
	get_ep.Get("/email/:email", leash_auth.AuthorizationMiddleware("email"), models.GetQueryMiddleware[userGetRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		email := c.Params("email")
		user, err := searchEmail(db, email)

		// Check if the user exists
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Preload the user with the specified fields
		req := c.Locals("query").(userGetRequest)
		req.Preload(db, &user)

		return c.JSON(user)
	})

	get_ep.Get("/get/card/:card", leash_auth.AuthorizationMiddleware("card"), models.GetQueryMiddleware[userGetRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		card := c.Params("card")

		card_id, err := strconv.ParseUint(card, 10, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid card ID")
		}

		// Check if the user exists
		var user = models.User{
			CardID: card_id,
		}

		if res := db.Limit(1).Where(&user).Find(&user); res.Error != nil || res.RowsAffected == 0 {
			return fiber.NewError(fiber.StatusNotFound, "User not found")
		}

		// Preload the user with the specified fields
		req := c.Locals("query").(userGetRequest)
		req.Preload(db, &user)

		return c.JSON(user)
	})
}

// addUserUpdateEndpoints creates the endpoints for user updates
func addUserUpdateEndpoints(user_ep fiber.Router) {
	// Create a new user update endpoint group
	update_ep := user_ep.Group("/updates", leash_auth.ConcatPermissionPrefixMiddleware("updates"))

	// List user updates endpoint
	update_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		req := c.Locals("query").(listRequest)

		// Count the total number of users
		total := db.Model(user).Association("UserUpdates").Count()

		// Paginate the results
		var updates []models.UserUpdate
		con := db.Model(&updates).Where(models.UserUpdate{UserID: user.ID})
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
			Updates []models.UserUpdate `json:"updates"`
			Total   int64               `json:"total"`
		}{
			Updates: updates,
			Total:   total,
		}

		return c.JSON(response)
	})
}

// commonUserEndpoints creates the endpoints for both self and others
func commonUserEndpoints(user_ep fiber.Router) {
	// Get the current user endpoint
	user_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("read"), models.GetQueryMiddleware[userGetRequest], func(c *fiber.Ctx) error {
		req := c.Locals("query").(userGetRequest)
		user := c.Locals("target_user").(models.User)

		// Preload the user with the specified fields
		req.Preload(leash_auth.GetDB(c), &user)
		return c.JSON(user)
	})

	// Update the current user endpoint
	type userUpdateRequest struct {
		Name     *string `json:"name" xml:"name" form:"name" validate:"omitempty"`
		Email    *string `json:"email" xml:"email" form:"email" validate:"omitempty,email"`
		CardId   *uint64 `json:"card_id" xml:"card_id" form:"card_id" validate:"omitempty"`
		Role     *string `json:"role" xml:"role" form:"role" validate:"omitempty,oneof=member volunteer staff admin"`
		Type     *string `json:"type" xml:"type" form:"type" validate:"omitempty,oneof=undergrad grad faculty staff alumni other"`
		GradYear *int    `json:"grad_year" xml:"grad_year" form:"grad_year" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
		Major    *string `json:"major" xml:"major" form:"major" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
	}
	user_ep.Patch("/", leash_auth.PrefixAuthorizationMiddleware("update"), models.GetBodyMiddleware[userUpdateRequest], func(c *fiber.Ctx) error {
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
		if req.Email != nil && *req.Email != user.Email && *req.Email != user.PendingEmail {
			_, err := searchEmail(db, *req.Email)
			if err == nil {
				// The user already exists
				return c.Status(fiber.StatusConflict).SendString("Email already in use")
			}

			event.Changes = append(event.Changes, UserChanges{
				Old:   user.PendingEmail,
				New:   *req.Email,
				Field: "pending_email",
			})
		}

		var cardId *string
		if req.CardId != nil {
			cardId = new(string)
			*cardId = fmt.Sprintf("%d", *req.CardId)
		}

		if modified(user.Type, req.Type, "type") {
			user.Type = *req.Type
		}

		var gradYear *string
		if req.GradYear != nil {
			gradYear = new(string)
			*gradYear = fmt.Sprintf("%d", *req.GradYear)
		}

		if modified(fmt.Sprint(user.GraduationYear), gradYear, "grad_year") {
			user.GraduationYear = *req.GradYear
		}

		if modified(user.Major, req.Major, "major") {
			user.Major = *req.Major
		}

		if modified(fmt.Sprint(user.CardID), cardId, "card_id") {
			if authenticator.Authorize(permissionPrefix+":update_card_id") != nil {
				user.CardID = *req.CardId
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		if modified(user.Role, req.Role, "role") {
			if authenticator.Authorize(permissionPrefix+":update_role") != nil {
				user.Role = *req.Role
				authenticator.Enforcer.SetUserRole(user, *req.Role)
			} else {
				return c.SendStatus(fiber.StatusUnauthorized)
			}
		}

		db.Save(&user)

		// Run the update callbacks
		for _, callback := range userUpdateCallbacks {
			callback(event)
		}

		return c.JSON(user)
	})

	// Add sub-endpoints
	addUserUpdateEndpoints(user_ep)
	addUserTrainingEndpoints(user_ep)
	addUserHoldsEndpoints(user_ep)
	addUserApiKeyEndpoints(user_ep)
}

// otherUserEndpoints creates the endpoints for other users
func otherUserEndpoints(user_ep fiber.Router) {
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

		return c.SendStatus(fiber.StatusNoContent)
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

	self_ep := users_ep.Group("/self", leash_auth.ConcatPermissionPrefixMiddleware("self"), selfMiddleware)
	commonUserEndpoints(self_ep)

	user_ep := users_ep.Group("/:user_id", leash_auth.ConcatPermissionPrefixMiddleware("others"), userMiddleware)
	commonUserEndpoints(user_ep)
	otherUserEndpoints(user_ep)
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

	if user.PendingEmail == "" {
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
				New:   user.PendingEmail,
				Field: "email",
			},
			{
				Old:   user.PendingEmail,
				New:   "",
				Field: "pending_email",
			},
		},
	}

	user.Email = user.PendingEmail
	user.PendingEmail = ""
	db.Save(&user)

	// Run the update callbacks
	for _, callback := range userUpdateCallbacks {
		callback(event)
	}

	return user, nil
}
