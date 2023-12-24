package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
)

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
