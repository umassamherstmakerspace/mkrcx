package leash_backend_api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

// userApiKeyMiddleware is a middleware that fetches the api key from a user and stores it in the context
func userApiKeyMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	user := c.Locals("target_user").(models.User)
	authentication := leash_auth.GetAuthentication(c)
	permissionPrefix := c.Locals("permission_prefix").(string)

	// Check if the user is authorized to perform the action
	if authentication.Authorize(permissionPrefix+":target") != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to read this user's api keys")
	}

	var apikey = models.APIKey{
		UserID: user.ID,
		Key:    c.Params("api_key"),
	}

	if res := db.Limit(1).Where(&apikey).Find(&apikey); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "API Key not found")
	}

	c.Locals("apikey", apikey)

	return c.Next()
}

// generalApiKeyMiddleware is a middleware that fetches the api key by ID and stores it in the context
func generalApiKeyMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	authentication := leash_auth.GetAuthentication(c)

	// Check if the user is authorized to perform the action
	if authentication.Authorize("leash.apikeys:target") != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to read api keys")
	}

	var apikey = models.APIKey{
		Key: c.Params("api_key"),
	}

	if res := db.Limit(1).Where(&apikey).Find(&apikey); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "API Key not found")
	}

	c.Locals("apikey", apikey)

	return c.Next()
}

// addCommonApiKeyEndpoints adds the common endpoints for api keys
func addCommonApiKeyEndpoints(apikey_ep fiber.Router) {
	// Get current api key endpoint
	apikey_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), func(c *fiber.Ctx) error {
		apikey := c.Locals("apikey").(models.APIKey)
		return c.JSON(apikey)
	})

	// Delete current api key endpoint
	apikey_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		apikey := c.Locals("apikey").(models.APIKey)
		enforcer := leash_auth.GetEnforcer(c)
		enforcer.DeletePermissionsForUser(fmt.Sprintf("apikey:%s", apikey.Key))

		db.Delete(&apikey)
		return c.SendStatus(fiber.StatusOK)
	})

	// Update current api key endpoint
	type apikeyUpdateRequest struct {
		Description *string   `json:"description" xml:"description" form:"description"`
		Permissions *[]string `json:"permissions" xml:"permissions" form:"permissions"`
		FullAccess  *bool     `json:"full_access" xml:"full_access" form:"full_access"`
	}

	apikey_ep.Patch("/", leash_auth.PrefixAuthorizationMiddleware("update"), models.GetBodyMiddleware[apikeyUpdateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		apikey := c.Locals("apikey").(models.APIKey)
		req := c.Locals("body").(apikeyUpdateRequest)

		if req.Description != nil {
			apikey.Description = *req.Description
		}

		enforcer := leash_auth.GetAuthentication(c).Enforcer

		if req.Permissions != nil {
			enforcer.SetPermissionsForAPIKey(apikey, *req.Permissions)
		}

		if req.FullAccess != nil {
			apikey.FullAccess = *req.FullAccess
		}

		enforcer.SavePolicy()

		db.Save(&apikey)

		return c.JSON(apikey)
	})
}

// addUserApiKeyEndpoints adds the endpoints for api keys for a user
func addUserApiKeyEndpoints(user_ep fiber.Router) {
	apikey_ep := user_ep.Group("/apikeys", leash_auth.ConcatPermissionPrefixMiddleware("apikeys"))

	// List api keys endpoint
	apikey_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), models.GetQueryMiddleware[listRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		req := c.Locals("query").(listRequest)

		// Count the total number of users
		total := db.Model(user).Association("APIKeys").Count()

		// Paginate the results
		var apikeys []models.APIKey

		con := db
		if req.IncludeDeleted != nil && *req.IncludeDeleted {
			con = con.Unscoped()
		}

		con = con.Model(&apikeys).Where(models.APIKey{UserID: user.ID})
		if req.Limit != nil {
			con = con.Limit(*req.Limit)
		} else {
			con = con.Limit(10)
		}

		if req.Offset != nil {
			con = con.Offset(*req.Offset)
		} else {
			con = con.Offset(0)
		}

		con.Find(&apikeys)

		response := struct {
			Data  []models.APIKey `json:"data"`
			Total int64           `json:"total"`
		}{
			Data:  apikeys,
			Total: total,
		}

		return c.JSON(response)
	})

	// Create api key endpoint
	type apikeyCreateRequest struct {
		Description *string   `json:"description" xml:"description" form:"description" validate:"omitempty"`
		Permissions *[]string `json:"permissions" xml:"permissions" form:"permissions" validate:"required"`
		FullAccess  *bool     `json:"full_access" xml:"full_access" form:"full_access" validate:"required"`
	}
	apikey_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[apikeyCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		req := c.Locals("body").(apikeyCreateRequest)

		key := uuid.New()

		if req.Description == nil {
			req.Description = new(string)
		}

		apikey := models.APIKey{
			Description: *req.Description,
			UserID:      user.ID,
			FullAccess:  *req.FullAccess,
			Key:         key.String(),
		}

		db.Create(&apikey)

		enforcer := leash_auth.GetAuthentication(c).Enforcer
		enforcer.SetPermissionsForAPIKey(apikey, *req.Permissions)
		enforcer.SavePolicy()

		return c.JSON(apikey)
	})

	user_apikey_ep := apikey_ep.Group("/:api_key", userApiKeyMiddleware)
	addCommonApiKeyEndpoints(user_apikey_ep)
}

// registerApiKeyEndpoints registers the api key endpoints
func registerApiKeyEndpoints(api fiber.Router) {
	apikey_ep := api.Group("/apikeys", leash_auth.ConcatPermissionPrefixMiddleware("apikeys"))

	single_apikey_ep := apikey_ep.Group("/:api_key", generalApiKeyMiddleware)

	addCommonApiKeyEndpoints(single_apikey_ep)
}
