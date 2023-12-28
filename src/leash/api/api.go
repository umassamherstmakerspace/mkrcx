package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
)

func RegisterAPIEndpoints(api fiber.Router) {
	api.Use(leash_auth.AuthenticationMiddleware)

	registerUserEndpoints(api)
	registerHoldsEndpoints(api)
	registerTrainingEndpoints(api)
	registerApiKeyEndpoints(api)
}
