package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

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
}

func registerUserEndpoints(api fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	self_ep := api.Group("/self", selfMiddleware)
	commonUserEndpoints(self_ep, db, keys)

	user_ep := api.Group("/:user_id", func(c *fiber.Ctx) error {
		return userMiddleware(c, db)
	})
	commonUserEndpoints(user_ep, db, keys)
}
