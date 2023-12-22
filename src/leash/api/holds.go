package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"gorm.io/gorm"
)

func addUserHoldsEndpoints(user_ep fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	hold_ep := user_ep.Group("/holds")

	hold_ep.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("get all holds")
	})
}

func registerHoldsEndpoints(api fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	holds_ep := api.Group("/holds")

	holds_ep.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("get all holds")
	})
}
