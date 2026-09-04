package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
)

type listRequest struct {
	Limit          *int  `query:"limit" validate:"omitempty,min=1,max=100"`
	Offset         *int  `query:"offset" validate:"omitempty,min=0"`
	IncludeDeleted *bool `query:"include_deleted"`
}

// RegisterAPIEndpoints registers all the API endpoints for Leash
func RegisterAPIEndpoints(api fiber.Router, feedRuntime ...*FeedRuntime) {
	runtime := NewLocalFeedRuntime()
	if len(feedRuntime) > 0 && feedRuntime[0] != nil {
		runtime = feedRuntime[0]
	}

	// Browser WebSockets cannot attach an Authorization header to the upgrade.
	// Register only this route before the HTTP authentication middleware; the
	// endpoint authenticates its first frame before adding the socket to the hub.
	websocketFeedEndpoint(api.Group("/feeds"), runtime)
	registerDocusignConnectEndpoint(api)
	api.Use(leash_auth.AuthenticationMiddleware)

	registerUserEndpoints(api, runtime)
	registerTrainingEndpoints(api)
	registerHoldsEndpoints(api)
	registerApiKeyEndpoints(api)
	registerNotificationsEndpoints(api)
	registerFeedEndpoints(api, runtime)
	registerCheckinEndpoints(api, runtime)
	registerNoteEndpoints(api)
}
