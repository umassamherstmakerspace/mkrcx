package leash_backend_api

import (
	"fmt"

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

	c.Locals("target_user", apiUser)
	c.Locals("self", true)
	return c.Next()
}

func userMiddleware(c *fiber.Ctx, db *gorm.DB) error {
	authentication := leash_auth.GetAuthentication(c)
	if authentication.Authorize("leash.target.others") != nil {
		return c.Status(401).SendString("Unauthorized")
	}

	user_id := c.Params("user_id")
	var user models.User
	db.First(&user, "id = ?", user_id)

	if user.ID == 0 {
		return c.Status(404).SendString("Not found")
	}

	c.Locals("target_user", user)
	c.Locals("self", false)
	return c.Next()
}

func registerUserEndpoints(api fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	self_ep := api.Group("/self", selfMiddleware)

	self_ep.Get("/", func(c *fiber.Ctx) error {
		fmt.Println("SELF")
		fmt.Println(c.Locals("user_id"))
		return c.SendString("Hello, " + c.Locals("target_user").(models.User).Email)
	})

	user_ep := api.Group("/:user_id", func(c *fiber.Ctx) error {
		return userMiddleware(c, db)
	})

	user_ep.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, " + c.Locals("target_user").(models.User).Email)
	})
}
