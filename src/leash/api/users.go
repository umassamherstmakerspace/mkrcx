package leash_backend_api

import (
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
	user_ep.Get("/", userGatedEndpointMiddleware("read", func(c *fiber.Ctx) error {
		return c.JSON(c.Locals("target_user"))
	}))
}

func registerUserEndpoints(api fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
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

	self_ep := api.Group("/self", selfMiddleware)
	commonUserEndpoints(self_ep, db, keys)

	user_ep := api.Group("/:user_id", func(c *fiber.Ctx) error {
		return userMiddleware(c, db)
	})
	commonUserEndpoints(user_ep, db, keys)
}
