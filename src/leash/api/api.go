package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"gorm.io/gorm"
)

func RegisterAPIEndpoints(api fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	api.Use(leash_auth.AuthenticationMiddleware(db, keys, func(c *fiber.Ctx) error {
		return c.Next()
	}))

	users_ep := api.Group("/users")

	registerUserEndpoints(users_ep, db, keys)
}
