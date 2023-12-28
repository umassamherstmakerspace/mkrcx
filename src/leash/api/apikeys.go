package leash_backend_api

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

func userApiKeyMiddlware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	user := c.Locals("target_user").(models.User)
	var apikey models.APIKey
	if err := db.Model(&user).Where("key = ?", c.Params("api_key")).Association("APIKeys").Find(&apikey); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "API Key not found")
	}
	c.Locals("apikey", apikey)
	return c.Next()
}

func generalApiKeyMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	var apikey models.APIKey
	if err := db.Where("key = ?", c.Params("api_key")).First(&apikey).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "API Key not found")
	}
	c.Locals("apikey", apikey)

	return c.Next()
}

func addCommonApiKeyEndpoints(apikey_ep fiber.Router) {
	apikey_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), func(c *fiber.Ctx) error {
		apikey := c.Locals("apikey").(models.APIKey)
		return c.JSON(apikey)
	})

	apikey_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		apikey := c.Locals("apikey").(models.APIKey)
		enforcer := leash_auth.GetEnforcer(c)
		enforcer.DeletePermissionsForUser(fmt.Sprintf("apikey:%s", apikey.Key))

		db.Delete(&apikey)
		return c.SendStatus(fiber.StatusNoContent)
	})

	type apikeyUpdateRequest struct {
		Description *string   `json:"description" xml:"description" form:"description"`
		Permissions *[]string `json:"permissions" xml:"permissions" form:"permissions"`
	}

	apikey_ep.Patch("/", leash_auth.PrefixAuthorizationMiddleware("update"), models.GetBodyMiddleware[apikeyUpdateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		apikey := c.Locals("apikey").(models.APIKey)
		req := c.Locals("body").(apikeyUpdateRequest)

		if req.Description != nil {
			apikey.Description = *req.Description
		}

		if req.Permissions != nil {
			apikey.Permissions = strings.Join(*req.Permissions, ",")

			enforcer := leash_auth.GetEnforcer(c)
			apiSubject := fmt.Sprintf("apikey:%s", apikey.Key)
			enforcer.DeletePermissionsForUser(apiSubject)
			for _, permission := range *req.Permissions {
				enforcer.AddPermissionForUser(apiSubject, permission)
			}
		}

		db.Save(&apikey)

		return c.JSON(apikey)
	})
}

func addUserApiKeyEndpoints(user_ep fiber.Router) {
	apikey_ep := user_ep.Group("/apikeys", leash_auth.ConcatPermissionPrefixMiddleware("apikeys"))

	apikey_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		var apikeys []models.APIKey
		db.Model(&user).Association("APIKeys").Find(&apikeys)
		return c.JSON(apikeys)
	})

	type apikeyCreateRequest struct {
		Description string   `json:"description" validate:"required"`
		Permissions []string `json:"permissions" validate:"required"`
	}
	apikey_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[apikeyCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		req := c.Locals("body").(apikeyCreateRequest)

		key := uuid.New()

		apikey := models.APIKey{
			Description: req.Description,
			UserID:      user.ID,
			Permissions: strings.Join(req.Permissions, ","),
			Key:         key.String(),
		}

		db.Create(&apikey)

		return c.JSON(apikey)
	})

	user_apikey_ep := apikey_ep.Group("/:api_key", userApiKeyMiddlware)
	addCommonApiKeyEndpoints(user_apikey_ep)
}

func registerApiKeyEndpoints(api fiber.Router) {
	apikey_ep := api.Group("/apikeys", leash_auth.ConcatPermissionPrefixMiddleware("apikeys"))

	single_apikey_ep := apikey_ep.Group("/:api_key", generalApiKeyMiddleware)

	addCommonApiKeyEndpoints(single_apikey_ep)
}
