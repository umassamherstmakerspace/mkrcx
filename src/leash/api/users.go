package leash_backend_api

import (
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

var userCreateCallbacks []func(UserEvent)
var userUpdateCallbacks []func(UserUpdateEvent)
var userDeleteCallbacks []func(UserEvent)

func selfMiddleware(c *fiber.Ctx) error {
	authentication := leash_auth.GetAuthentication(c)
	if authentication.Authorize("leash.users:target_self") != nil {
		return c.Status(401).SendString("Unauthorized")
	}

	apiUser := authentication.User

	c.Locals("target_user", apiUser)
	return c.Next()
}

func userMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	authentication := leash_auth.GetAuthentication(c)
	if authentication.Authorize("leash.users:target_others") != nil {
		return c.Status(401).SendString("Unauthorized")
	}

	user_id := c.Params("user_id")
	var user models.User
	err := db.First(&user, "id = ?", user_id).Error
	if err != nil {
		return c.Status(404).SendString("User not found")
	}

	c.Locals("target_user", user)
	return c.Next()
}

func createBaseEndpoints(users_ep fiber.Router) {
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
		{
			var user models.User
			res := db.Find(&user, "email = ? OR pending_email = ?", req.Email, req.Email)
			if res.RowsAffected > 0 {
				// The user already exists
				return c.Status(fiber.StatusConflict).SendString("User already exists")
			}
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

	type userSearchQuery struct {
		Query           *string `query:"query" validate:"required"`
		Limit           *int    `query:"limit" validate:"omitempty,min=1,max=100"`
		Offset          *int    `query:"offset" validate:"omitempty,min=0"`
		PreloadTraining *bool   `query:"preload_training" validate:"omitempty"`
		PreloadHolds    *bool   `query:"preload_holds" validate:"omitempty"`
	}
	users_ep.Get("/search", leash_auth.PrefixAuthorizationMiddleware("search"), models.GetBodyMiddleware[userSearchQuery], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		req := c.Locals("query").(userSearchQuery)
		authenticator := leash_auth.GetAuthentication(c)

		var users []models.User
		con := db.Where("name LIKE ?", "%"+*req.Query+"%").Or("email LIKE ?", "%"+*req.Query+"%")

		if req.PreloadTraining != nil && *req.PreloadTraining {
			if authenticator.Authorize("leash.users.others.trainings:list") != nil {
				con = con.Preload("Trainings")
			}
		}

		if req.PreloadHolds != nil && *req.PreloadHolds {
			if authenticator.Authorize("leash.users.others.holds:list") != nil {
				con = con.Preload("Holds")
			}
		}

		total := int64(0)
		con.Model(&models.User{}).Count(&total)

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

func createGetUserEndpoints(get_ep fiber.Router) {
	get_ep.Get("/email/:email", leash_auth.AuthorizationMiddleware("email"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		email := c.Params("email")
		var user models.User
		err := db.First(&user, "email = ? OR pending_email = ?", email, email).Error
		if err != nil {
			return c.Status(404).SendString("User not found")
		}

		return c.JSON(user)
	})

	get_ep.Get("/get/card/:card", leash_auth.AuthorizationMiddleware("card"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		card := c.Params("card")
		var user models.User
		err := db.First(&user, "card_id = ?", card).Error
		if err != nil {
			return c.Status(404).SendString("User not found")
		}

		return c.JSON(user)
	})
}

func addUserUpdateEndpoints(user_ep fiber.Router) {
	update_ep := user_ep.Group("/updates", leash_auth.ConcatPermissionPrefixMiddleware("updates"))

	update_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		var updates []models.UserUpdate
		db.Model(&user).Association("UserUpdates").Find(&updates)
		return c.JSON(updates)
	})
}

func commonUserEndpoints(user_ep fiber.Router) {
	user_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("read"), func(c *fiber.Ctx) error {
		return c.JSON(c.Locals("target_user"))
	})

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

		event := UserUpdateEvent{
			UserEvent: UserEvent{
				c:         c,
				Target:    user,
				Agent:     authenticator.User,
				Timestamp: time.Now().Unix(),
			},
			Changes: []UserChanges{},
		}

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

		if modified(user.Name, req.Name, "name") {
			user.Name = *req.Name
		}

		if req.Email != nil && *req.Email != user.Email && *req.Email != user.PendingEmail {
			var tmpUser models.User
			res := db.Find(&tmpUser, "email = ? OR pending_email = ?", *req.Email, *req.Email)
			if res.RowsAffected > 0 {
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
				return c.SendStatus(403)
			}
		}

		if modified(user.Role, req.Role, "role") {
			if authenticator.Authorize(permissionPrefix+":update_role") != nil {
				user.Role = *req.Role
				authenticator.Enforcer.SetUserRole(user, *req.Role)
			} else {
				return c.SendStatus(403)
			}
		}

		db.Save(&user)

		for _, callback := range userUpdateCallbacks {
			callback(event)
		}

		return c.JSON(user)
	})

	addUserUpdateEndpoints(user_ep)
	addUserTrainingEndpoints(user_ep)
	addUserHoldsEndpoints(user_ep)
	addUserApiKeyEndpoints(user_ep)
}

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

func registerUserEndpoints(api fiber.Router) {
	users_ep := api.Group("/users", leash_auth.ConcatPermissionPrefixMiddleware("users"))

	userCreateCallbacks = []func(UserEvent){}
	userUpdateCallbacks = []func(UserUpdateEvent){}
	userDeleteCallbacks = []func(UserEvent){}

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

func OnUserCreate(callback func(UserEvent)) {
	userCreateCallbacks = append(userCreateCallbacks, callback)
}

func OnUserUpdate(callback func(UserUpdateEvent)) {
	userUpdateCallbacks = append(userUpdateCallbacks, callback)
}

func OnUserDelete(callback func(UserEvent)) {
	userDeleteCallbacks = append(userDeleteCallbacks, callback)
}

func UpdatePendingEmail(user models.User, c *fiber.Ctx) (models.User, error) {
	db := leash_auth.GetDB(c)

	if user.PendingEmail == "" {
		return user, errors.New("no pending email")
	}

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

	for _, callback := range userUpdateCallbacks {
		callback(event)
	}

	return user, nil
}
