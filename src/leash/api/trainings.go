package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"gorm.io/gorm"
)

func addUserTrainingEndpoints(user_ep fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	training_ep := user_ep.Group("/trainings")

	training_ep.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("get all trainings")
	})
}

func registerTrainingEndpoints(api fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	training_ep := api.Group("/trainings")

	training_ep.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("get all trainings")
	})
}
