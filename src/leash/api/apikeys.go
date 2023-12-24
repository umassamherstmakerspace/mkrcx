package leash_backend_api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

func userApiKeyMiddlware(c *fiber.Ctx, db *gorm.DB) error {
	user := c.Locals("target_user").(models.User)
	var apikey models.APIKey
	if err := db.Model(&user).Where("key = ?", c.Params("api_key")).Association("APIKeys").Find(&apikey); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "API Key not found")
	}
	c.Locals("apikey", apikey)

	permission_prefix := c.Locals("permission_prefix").(string)
	c.Locals("permission_prefix", permission_prefix+".apikeys")
	return c.Next()
}

func generalApiKeyMiddleware(c *fiber.Ctx, db *gorm.DB) error {
	var apikey models.APIKey
	if err := db.Where("key = ?", c.Params("api_key")).First(&apikey).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "API Key not found")
	}
	c.Locals("apikey", apikey)

	c.Locals("permission_prefix", "leash.apikeys")
	return c.Next()
}

func addCommonApiKeyEndpoints(apikey_ep fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	apikey_ep.Get("/", prefixGatedEndpointMiddleware("", "get", func(c *fiber.Ctx) error {
		apikey := c.Locals("apikey").(models.APIKey)
		return c.JSON(apikey)
	}))

	apikey_ep.Delete("/", prefixGatedEndpointMiddleware("", "delete", func(c *fiber.Ctx) error {
		apikey := c.Locals("apikey").(models.APIKey)

		db.Delete(&apikey)
		return c.SendStatus(fiber.StatusNoContent)
	}))

	apikey_ep.Patch("/", prefixGatedEndpointMiddleware("", "update", func(c *fiber.Ctx) error {
		type request struct {
			Description *string   `json:"description"`
			Permissions *[]string `json:"permissions"`
		}

		next := getBodyMiddleware(request{}, func(c *fiber.Ctx) error {
			apikey := c.Locals("apikey").(models.APIKey)
			req := c.Locals("body").(request)

			if req.Description != nil {
				apikey.Description = *req.Description
			}

			if req.Permissions != nil {
				apikey.Permissions = strings.Join(*req.Permissions, ",")
			}

			db.Save(&apikey)

			return c.JSON(apikey)
		})

		return next(c)
	}))
}

func addUserApiKeyEndpoints(user_ep fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	apikey_ep := user_ep.Group("/apikeys")

	apikey_ep.Get("/", prefixGatedEndpointMiddleware("apikeys", "list", func(c *fiber.Ctx) error {
		user := c.Locals("target_user").(models.User)
		var apikeys []models.APIKey
		db.Model(&user).Association("APIKeys").Find(&apikeys)
		return c.JSON(apikeys)
	}))

	apikey_ep.Post("/", prefixGatedEndpointMiddleware("apikeys", "create", func(c *fiber.Ctx) error {
		type request struct {
			Description string   `json:"description" validate:"required"`
			Permissions []string `json:"permissions" validate:"required"`
		}

		next := getBodyMiddleware(request{}, func(c *fiber.Ctx) error {
			user := c.Locals("target_user").(models.User)
			req := c.Locals("body").(request)

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

		return next(c)
	}))

	user_apikey_ep := apikey_ep.Group("/:api_key", func(c *fiber.Ctx) error {
		return userApiKeyMiddlware(c, db)
	})
	addCommonApiKeyEndpoints(user_apikey_ep, db, keys)
}

func registerApiKeyEndpoints(api fiber.Router, db *gorm.DB, keys leash_auth.Keys) {
	apikey_ep := api.Group("/apikeys")

	single_apikey_ep := apikey_ep.Group("/:api_key", func(c *fiber.Ctx) error {
		return generalApiKeyMiddleware(c, db)
	})

	addCommonApiKeyEndpoints(single_apikey_ep, db, keys)
}
