package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

func getBodyMiddleware[V interface{}](structType V, next fiber.Handler) fiber.Handler {

	return func(c *fiber.Ctx) error {
		var req V

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		errors := models.ValidateStruct(req)
		if errors != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errors)
		}

		c.Locals("body", req)
		return next(c)
	}
}

func getQueryMiddleware[V interface{}](structType V, next fiber.Handler) fiber.Handler {

	return func(c *fiber.Ctx) error {
		var req V

		if err := c.QueryParser(&req); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		errors := models.ValidateStruct(req)
		if errors != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errors)
		}

		c.Locals("query", req)
		return next(c)
	}
}

func RegisterAPIEndpoints(api fiber.Router) {
	api.Use(leash_auth.AuthenticationMiddleware)

	users_ep := api.Group("/users")

	registerUserEndpoints(users_ep)
	registerHoldsEndpoints(users_ep)
	registerTrainingEndpoints(users_ep)
	registerApiKeyEndpoints(users_ep)
}

func prefixGatedEndpointMiddleware(permissionSuffix string, permissionAction string, next fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		permissionObject := c.Locals("permission_prefix").(string) + "." + permissionSuffix
		authorize := leash_auth.AuthorizationMiddleware(permissionObject, permissionAction, next)
		return authorize(c)
	}
}
