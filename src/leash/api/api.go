package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
)

type listRequest struct {
	Limit  *int `query:"limit" validate:"omitempty,min=1,max=100"`
	Offset *int `query:"offset" validate:"omitempty,min=0"`
}

// RegisterAPIEndpoints registers all the API endpoints for Leash
func RegisterAPIEndpoints(api fiber.Router) {
	api.Use(leash_auth.AuthenticationMiddleware)

	registerUserEndpoints(api)
	registerTrainingEndpoints(api)
	registerHoldsEndpoints(api)
	registerApiKeyEndpoints(api)
}
