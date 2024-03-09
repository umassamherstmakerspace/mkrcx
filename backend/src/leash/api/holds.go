package leash_backend_api

import (
	"net/url"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

// userHoldMiddleware is a middleware that fetches the hold from a user and stores it in the context
func userHoldMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	user := c.Locals("target_user").(models.User)
	authentication := leash_auth.GetAuthentication(c)
	permissionPrefix := c.Locals("permission_prefix").(string)

	// Check if the user is authorized to perform the action
	if authentication.Authorize(permissionPrefix+":target") != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to read this user's holds")
	}

	hold_name, err := url.QueryUnescape(c.Params("hold_name"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid hold type")
	}

	var hold = models.Hold{
		UserID: user.ID,
		Name:   hold_name,
	}
	if res := db.Limit(1).Where(&hold).Find(&hold); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Hold not found")
	}
	c.Locals("hold", hold)

	return c.Next()
}

// generalHoldMiddleware is a middleware that fetches the hold by ID and stores it in the context
func generalHoldMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	authentication := leash_auth.GetAuthentication(c)

	// Check if the user is authorized to perform the action
	if authentication.Authorize("leash.holds:target") != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to read holds")
	}

	hold_id, err := strconv.Atoi(c.Params("hold_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid hold ID")
	}

	var hold = models.Hold{
		ID: uint(hold_id),
	}

	if res := db.Limit(1).Where(&hold).Find(&hold); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Hold not found")
	}
	c.Locals("hold", hold)

	return c.Next()
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
		return c.SendStatus(fiber.StatusOK)
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
		Name     string `json:"name" xml:"name" form:"name" validate:"required"`
		Reason   string `json:"reason" xml:"reason" form:"reason" validate:"required"`
		Start    *int64 `json:"start" xml:"start" form:"start" validate:"omitempty,numeric"`
		End      *int64 `json:"end" xml:"end" form:"end" validate:"omitempty,numeric"`
		Priority *int   `json:"priority" xml:"priority" form:"priority" validate:"required,numeric"`
	}
	hold_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[holdCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		body := c.Locals("body").(holdCreateRequest)

		// Check if the user already has a hold of this type
		var existingHold = models.Hold{
			UserID: user.ID,
			Name:   body.Name,
		}
		if res := db.Limit(1).Where(&existingHold).Find(&existingHold); res.Error == nil && res.RowsAffected != 0 {
			return fiber.NewError(fiber.StatusConflict, "User already has a hold of this type")
		}

		hold := models.Hold{
			Name:     body.Name,
			Reason:   body.Reason,
			UserID:   user.ID,
			AddedBy:  leash_auth.GetAuthentication(c).User.ID,
			Priority: *body.Priority,
		}

		if body.Start != nil {
			start := time.Unix(*body.Start, 0)
			hold.Start = &start
		}

		if body.End != nil {
			end := time.Unix(*body.End, 0)
			if end.Before(time.Now()) {
				return fiber.NewError(fiber.StatusBadRequest, "Hold end time cannot be in the past")
			}

			hold.End = &end
		}

		if hold.Start != nil && hold.End != nil && hold.Start.After(*hold.End) {
			return fiber.NewError(fiber.StatusBadRequest, "Hold start time cannot be after hold end time")
		}

		db.Save(&hold)

		return c.JSON(hold)
	})

	user_hold_ep := hold_ep.Group("/:hold_name", userHoldMiddleware)

	addCommonHoldEndpoints(user_hold_ep)
}

// registerHoldsEndpoints registers the endpoints for holds
func registerHoldsEndpoints(api fiber.Router) {
	holds_ep := api.Group("/holds", leash_auth.ConcatPermissionPrefixMiddleware("holds"))

	single_hold_ep := holds_ep.Group("/:hold_id", generalHoldMiddleware)

	addCommonHoldEndpoints(single_hold_ep)
}
