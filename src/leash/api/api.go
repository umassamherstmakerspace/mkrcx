package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
)

func RegisterAPIEndpoints(api fiber.Router) {
	api.Get("/users/@me", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})
}
