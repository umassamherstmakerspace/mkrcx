package leash_backend_api

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

func userHoldMiddlware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	user := c.Locals("target_user").(models.User)
	var hold = models.Hold{
		UserID:   user.ID,
		HoldType: c.Params("hold_type"),
	}
	if res := db.Limit(1).Where(&hold).Find(&hold); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Hold not found")
	}
	c.Locals("hold", hold)

	return c.Next()
}

func generalHoldMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)

	hold_id, err := strconv.Atoi(c.Params("hold_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid hold ID")
	}

	var hold models.Hold
	hold.ID = uint(hold_id)

	if res := db.Limit(1).Where(&hold).Find(&hold); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Hold not found")
	}
	c.Locals("hold", hold)

	return c.Next()
}

func addCommonHoldEndpoints(hold_ep fiber.Router) {
	hold_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), func(c *fiber.Ctx) error {
		hold := c.Locals("hold").(models.Hold)
		return c.JSON(hold)
	})

	hold_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		hold := c.Locals("hold").(models.Hold)
		hold.RemovedBy = leash_auth.GetAuthentication(c).User.ID

		db.Save(&hold)

		db.Delete(&hold)
		return c.SendStatus(fiber.StatusNoContent)
	})
}

func addUserHoldsEndpoints(user_ep fiber.Router) {
	hold_ep := user_ep.Group("/holds", leash_auth.PrefixAuthorizationMiddleware("holds"))

	hold_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		var holds []models.Hold
		db.Model(&user).Association("Holds").Find(&holds)
		return c.JSON(holds)
	})

	type holdCreateRequest struct {
		HoldType  string `json:"hold_type" xml:"hold_type" form:"hold_type" validate:"required"`
		Reason    string `json:"reason" xml:"reason" form:"reason" validate:"required"`
		HoldStart *int64 `json:"hold_start" xml:"hold_start" form:"hold_start" validate:"numeric"`
		HoldEnd   *int64 `json:"hold_end" xml:"hold_end" form:"hold_end" validate:"numeric"`
	}
	hold_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[holdCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		body := c.Locals("body").(holdCreateRequest)

		// Check if the user already has a hold of this type
		var existingHold = models.Hold{
			UserID:   user.ID,
			HoldType: body.HoldType,
		}
		if res := db.Limit(1).Where(&existingHold).Find(&existingHold); res.Error == nil && res.RowsAffected != 0 {
			return fiber.NewError(fiber.StatusConflict, "User already has a hold of this type")
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
	})

	user_hold_ep := hold_ep.Group("/:hold_type", userHoldMiddlware)

	addCommonHoldEndpoints(user_hold_ep)
}

func registerHoldsEndpoints(api fiber.Router) {
	holds_ep := api.Group("/holds", leash_auth.PrefixAuthorizationMiddleware("holds"))

	single_hold_ep := holds_ep.Group("/:hold_id", generalHoldMiddleware)

	addCommonHoldEndpoints(single_hold_ep)
}
