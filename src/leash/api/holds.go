package leash_backend_api

import (
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

func userHoldMiddlware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	user := c.Locals("target_user").(models.User)
	var hold models.Hold
	if err := db.Model(&user).Where("hold_type = ?", c.Params("hold_type")).Association("Holds").Find(&hold); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Hold not found")
	}
	c.Locals("hold", hold)

	permission_prefix := c.Locals("permission_prefix").(string)
	c.Locals("permission_prefix", permission_prefix+".holds")
	return c.Next()
}

func generalHoldMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	var hold models.Hold
	if err := db.Where("id = ?", c.Params("hold_id")).First(&hold).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Hold not found")
	}
	c.Locals("hold", hold)

	c.Locals("permission_prefix", "leash.holds")
	return c.Next()
}

func addCommonHoldEndpoints(hold_ep fiber.Router) {
	hold_ep.Get("/", prefixGatedEndpointMiddleware("", "get", func(c *fiber.Ctx) error {
		hold := c.Locals("hold").(models.Hold)
		return c.JSON(hold)
	}))

	hold_ep.Delete("/", prefixGatedEndpointMiddleware("", "delete", func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		hold := c.Locals("hold").(models.Hold)
		hold.RemovedBy = leash_auth.GetAuthentication(c).User.ID

		db.Save(&hold)

		db.Delete(&hold)
		return c.SendStatus(fiber.StatusNoContent)
	}))
}

func addUserHoldsEndpoints(user_ep fiber.Router) {
	hold_ep := user_ep.Group("/holds")

	hold_ep.Get("/", prefixGatedEndpointMiddleware("holds", "list", func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		var holds []models.Hold
		db.Model(&user).Association("Holds").Find(&holds)
		return c.JSON(holds)
	}))

	hold_ep.Post("/", prefixGatedEndpointMiddleware("holds", "create", func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		type request struct {
			HoldType  string `json:"hold_type" validate:"required"`
			Reason    string `json:"reason" validate:"required"`
			HoldStart *int64 `json:"hold_start" validate:"numeric"`
			HoldEnd   *int64 `json:"hold_end" validate:"numeric"`
		}

		return models.GetBodyMiddleware(request{}, func(c *fiber.Ctx) error {
			user := c.Locals("target_user").(models.User)
			body := c.Locals("body").(request)

			// Check if the user already has a hold of this type
			var existingHold models.Hold
			if err := db.Model(&user).Where("hold_type = ?", body.HoldType).Association("Holds").Find(&existingHold); err == nil {
				return fiber.NewError(fiber.StatusBadRequest, "User already has a hold of this type")
			}

			hold := models.Hold{
				HoldType: body.HoldType,
				Reason:   body.Reason,
				UserID:   user.ID,
			}

			if body.HoldStart != nil {
				holdStart := time.Unix(*body.HoldStart, 0)
				hold.HoldStart = &holdStart
			}

			if body.HoldEnd != nil {
				holdEnd := time.Unix(*body.HoldEnd, 0)
				hold.HoldEnd = &holdEnd
			}

			db.Save(&hold)

			return c.JSON(hold)
		})(c)
	}))

	user_hold_ep := hold_ep.Group("/:hold_type", userHoldMiddlware)

	addCommonHoldEndpoints(user_hold_ep)
}

func registerHoldsEndpoints(api fiber.Router) {
	holds_ep := api.Group("/holds")

	single_hold_ep := holds_ep.Group("/:hold_id", generalHoldMiddleware)

	addCommonHoldEndpoints(single_hold_ep)
}
