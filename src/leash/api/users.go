package leash_backend_api

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

var userCreateCallbacks []func(UserEvent)
var userUpdateCallbacks []func(UserUpdateEvent)
var userDeleteCallbacks []func(UserEvent)

func selfMiddleware(c *fiber.Ctx) error {
	authentication := leash_auth.GetAuthentication(c)
	if authentication.Authorize("leash.target.self") != nil {
		return c.Status(401).SendString("Unauthorized")
	}

	apiUser := authentication.User

	c.Locals("source_user", apiUser)
	c.Locals("target_user", apiUser)
	c.Locals("self", true)
	c.Locals("permission_prefix", "leash.users.self")
	return c.Next()
}

func createEndpoint(api fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	api.Post("/", leash_auth.AuthorizationMiddleware("leash.users.create", func(c *fiber.Ctx) error {
		type request struct {
			Email    string `json:"email" xml:"email" form:"email" validate:"required,email"`
			Name     string `json:"name" xml:"name" form:"name" validate:"required"`
			Role     string `json:"role" xml:"role" form:"role" validate:"required,oneof=member volunteer staff admin"`
			Type     string `json:"type" xml:"type" form:"type" validate:"required,oneof=undergrad grad faculty staff alumni other"`
			GradYear int    `json:"grad_year" xml:"grad_year" form:"grad_year" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
			Major    string `json:"major" xml:"major" form:"major" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
		}

		next := getBodyMiddleware(request{}, func(c *fiber.Ctx) error {
			req := c.Locals("body").(request)

			// Check if the user already exists
			{
				var user models.User
				res := db.Find(&user, "email = ?", req.Email)
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

			event := UserEvent{
				Target:    user,
				Agent:     leash_auth.GetAuthentication(c).User,
				Timestamp: time.Now().Unix(),
			}

			for _, callback := range userCreateCallbacks {
				callback(event)
			}

			// Write a success message to the response
			return c.SendString("User created successfully")
		})

		return next(c)
	}))

	api.Get("/search", leash_auth.AuthorizationMiddleware("leash.users.search", func(c *fiber.Ctx) error {
		type request struct {
			Query  *string `query:"query" validate:"required"`
			Limit  *int    `query:"limit" validate:"omitempty,min=1,max=100"`
			Offset *int    `query:"offset" validate:"omitempty,min=0"`
		}

		next := getQueryMiddleware(request{}, func(c *fiber.Ctx) error {
			req := c.Locals("query").(request)

			var users []models.User
			con := db.Where("name LIKE ?", "%"+*req.Query+"%").Or("email LIKE ?", "%"+*req.Query+"%")

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

		return next(c)
	}))

	api.Get("/get/email/:email", leash_auth.AuthorizationMiddleware("leash.users.get.email", func(c *fiber.Ctx) error {
		email := c.Params("email")
		var user models.User
		err := db.First(&user, "email = ?", email).Error
		if err != nil {
			return c.Status(404).SendString("User not found")
		}

		return c.JSON(user)
	}))

	api.Get("/get/card/:card", leash_auth.AuthorizationMiddleware("leash.users.get.card", func(c *fiber.Ctx) error {
		card := c.Params("card")
		var user models.User
		err := db.First(&user, "card_id = ?", card).Error
		if err != nil {
			return c.Status(404).SendString("User not found")
		}

		return c.JSON(user)
	}))
}

func userMiddleware(c *fiber.Ctx, db *gorm.DB) error {
	authentication := leash_auth.GetAuthentication(c)
	if authentication.Authorize("leash.target.others") != nil {
		return c.Status(401).SendString("Unauthorized")
	}

	user_id := c.Params("user_id")
	var user models.User
	err := db.First(&user, "id = ?", user_id).Error
	if err != nil {
		return c.Status(404).SendString("User not found")
	}

	c.Locals("source_user", authentication.User)
	c.Locals("target_user", user)
	c.Locals("self", false)
	c.Locals("permission_prefix", "leash.users.other")
	return c.Next()
}

func userGatedEndpointMiddleware(permissionSuffix string, next fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permission := c.Locals("permission_prefix").(string) + "." + permissionSuffix
		authorize := leash_auth.AuthorizationMiddleware(permission, next)
		return authorize(c)
	}
}

func commonUserEndpoints(user_ep fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	user_ep.Get("/", userGatedEndpointMiddleware("read", func(c *fiber.Ctx) error {
		return c.JSON(c.Locals("target_user"))
	}))

	user_ep.Put("/", userGatedEndpointMiddleware("edit", func(c *fiber.Ctx) error {
		type request struct {
			Name     *string `json:"name" xml:"name" form:"name" validate:"omitempty"`
			Email    *string `json:"new_email" xml:"new_email" form:"new_email" validate:"omitempty,email"`
			Role     *string `json:"role" xml:"role" form:"role" validate:"omitempty,oneof=member volunteer staff admin"`
			Type     *string `json:"type" xml:"type" form:"type" validate:"omitempty,oneof=undergrad grad faculty staff alumni other"`
			GradYear *int    `json:"grad_year" xml:"grad_year" form:"grad_year" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
			Major    *string `json:"major" xml:"major" form:"major" validate:"required_if=Type undergrad,required_if=Type grad,required_if=Type alumni"`
		}

		next := getBodyMiddleware(request{}, func(c *fiber.Ctx) error {
			req := c.Locals("body").(request)
			user := c.Locals("target_user").(models.User)

			event := UserUpdateEvent{
				UserEvent: UserEvent{
					Target:    user,
					Agent:     leash_auth.GetAuthentication(c).User,
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

			if modified(user.Email, req.Email, "email") {
				user.Email = *req.Email
			}

			if modified(user.Role, req.Role, "role") {
				user.Role = *req.Role
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

			db.Save(&user)

			for _, callback := range userUpdateCallbacks {
				callback(event)
			}

			return c.SendString("User updated successfully")
		})

		return next(c)
	}))
}

func otherUserEndpoints(user_ep fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	user_ep.Delete("/", userGatedEndpointMiddleware("delete", func(c *fiber.Ctx) error {
		user := c.Locals("target_user").(models.User)

		event := UserEvent{
			Target:    user,
			Agent:     leash_auth.GetAuthentication(c).User,
			Timestamp: time.Now().Unix(),
		}

		for _, callback := range userDeleteCallbacks {
			callback(event)
		}

		return c.SendString("User deleted successfully")
	}))
}

func registerUserEndpoints(api fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	userCreateCallbacks = []func(UserEvent){}
	userUpdateCallbacks = []func(UserUpdateEvent){}
	userDeleteCallbacks = []func(UserEvent){}

	OnUserUpdate(func(event UserUpdateEvent) {
		for _, change := range event.Changes {

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

	createEndpoint(api, db, keys)

	self_ep := api.Group("/self", selfMiddleware)
	commonUserEndpoints(self_ep, db, keys)

	user_ep := api.Group("/:user_id", func(c *fiber.Ctx) error {
		return userMiddleware(c, db)
	})
	commonUserEndpoints(user_ep, db, keys)
	otherUserEndpoints(user_ep, db, keys)
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
