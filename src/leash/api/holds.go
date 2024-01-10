package leash_backend_api

import (
	"net/url"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

// userHoldMiddlware is a middleware that fetches the hold from a user and stores it in the context
func userHoldMiddlware(c *fiber.Ctx) error {
	return leash_auth.AfterAuthenticationMiddleware(func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)

		hold_type, err := url.QueryUnescape(c.Params("hold_type"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid hold type")
		}

		var hold = models.Hold{
			UserID:   user.ID,
			HoldType: hold_type,
		}
		if res := db.Limit(1).Where(&hold).Find(&hold); res.Error != nil || res.RowsAffected == 0 {
			return fiber.NewError(fiber.StatusNotFound, "Hold not found")
		}
		c.Locals("hold", hold)

		return nil
	})(c)
}

// generalHoldMiddleware is a middleware that fetches the hold by ID and stores it in the context
func generalHoldMiddleware(c *fiber.Ctx) error {
	return leash_auth.AfterAuthenticationMiddleware(func(c *fiber.Ctx) error {
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

		return nil
	})(c)
}

// addCommonHoldEndpoints adds the common endpoints for holds
func addCommonHoldEndpoints(hold_ep fiber.Router) {
	// Get current hold endpoint
	hold_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), func(c *fiber.Ctx) error {
		hold := c.Locals("hold").(models.Hold)
		return c.JSON(hold)
	})

	// Delete current hold endpoint
	hold_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		hold := c.Locals("hold").(models.Hold)
		hold.RemovedBy = leash_auth.GetAuthentication(c).User.ID

		db.Save(&hold)

		db.Delete(&hold)
		return c.SendStatus(fiber.StatusNoContent)
	})
}

// addUserHoldsEndpoints adds the endpoints for holds for a user
func addUserHoldsEndpoints(user_ep fiber.Router) {
	hold_ep := user_ep.Group("/holds", leash_auth.ConcatPermissionPrefixMiddleware("holds"))

	// List holds endpoint
	hold_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), models.GetQueryMiddleware[listRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		req := c.Locals("query").(listRequest)

		// Count the total number of users
		total := db.Model(user).Association("Holds").Count()

		// Paginate the results
		var holds []models.Hold

		con := db
		if req.IncludeDeleted != nil && *req.IncludeDeleted {
			con = con.Unscoped()
		}

		con = con.Model(&holds).Where(models.Hold{UserID: user.ID})
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

		con.Find(&holds)

		response := struct {
			Data  []models.Hold `json:"data"`
			Total int64         `json:"total"`
		}{
			Data:  holds,
			Total: total,
		}

		return c.JSON(response)
	})

	// Create hold endpoint
	type holdCreateRequest struct {
		HoldType  string `json:"hold_type" xml:"hold_type" form:"hold_type" validate:"required"`
		Reason    string `json:"reason" xml:"reason" form:"reason" validate:"required"`
		HoldStart *int64 `json:"hold_start" xml:"hold_start" form:"hold_start" validate:"omitempty,numeric"`
		HoldEnd   *int64 `json:"hold_end" xml:"hold_end" form:"hold_end" validate:"omitempty,numeric"`
		Priority  *int   `json:"priority" xml:"priority" form:"priority" validate:"required,numeric"`
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
			AddedBy:  leash_auth.GetAuthentication(c).User.ID,
			Priority: *body.Priority,
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

// registerHoldsEndpoints registers the endpoints for holds
func registerHoldsEndpoints(api fiber.Router) {
	holds_ep := api.Group("/holds", leash_auth.ConcatPermissionPrefixMiddleware("holds"))

	single_hold_ep := holds_ep.Group("/:hold_id", generalHoldMiddleware)

	addCommonHoldEndpoints(single_hold_ep)
}
