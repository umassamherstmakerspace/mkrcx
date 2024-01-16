package leash_backend_api

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

// userNotificationMiddleware is a middleware that fetches the notification from a user and stores it in the context
func userNotificationMiddleware(c *fiber.Ctx) error {
	return leash_auth.AfterAuthenticationMiddleware(func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		notification_id, err := strconv.Atoi(c.Params("notification_id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid notification ID")
		}

		var notification = models.Notification{
			UserID: user.ID,
			ID:     uint(notification_id),
		}

		if res := db.Limit(1).Where(&notification).Find(&notification); res.Error != nil || res.RowsAffected == 0 {
			return fiber.NewError(fiber.StatusNotFound, "Notification not found")
		}
		c.Locals("notification", notification)

		return nil
	})(c)
}

// generalNotificationMiddleware is a middleware that fetches the notification by ID and stores it in the context
func generalNotificationMiddleware(c *fiber.Ctx) error {
	return leash_auth.AfterAuthenticationMiddleware(func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)

		notification_id, err := strconv.Atoi(c.Params("notification_id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid notification ID")
		}

		var notification = models.Notification{
			ID: uint(notification_id),
		}

		if res := db.Limit(1).Where(&notification).Find(&notification); res.Error != nil || res.RowsAffected == 0 {
			return fiber.NewError(fiber.StatusNotFound, "Notification not found")
		}
		c.Locals("notification", notification)

		return nil
	})(c)
}

// addCommonNotificationEndpoints adds the common endpoints for notifications
func addCommonNotificationEndpoints(notification_ep fiber.Router) {
	// Get current notification endpoint
	notification_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), func(c *fiber.Ctx) error {
		notification := c.Locals("notification").(models.Notification)
		return c.JSON(notification)
	})

	// Delete current notification endpoint
	notification_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		notification := c.Locals("notification").(models.Notification)

		notification.RemovedBy = leash_auth.GetAuthentication(c).User.ID
		db.Save(&notification)

		db.Delete(&notification)

		return c.SendStatus(fiber.StatusOK)
	})
}

// addUserNotificationsEndpoints adds the endpoints for notifications for a user
func addUserNotificationsEndpoints(user_ep fiber.Router) {
	notification_ep := user_ep.Group("/notifications", leash_auth.ConcatPermissionPrefixMiddleware("notifications"))

	// List notifications endpoint
	notification_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), models.GetQueryMiddleware[listRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		req := c.Locals("query").(listRequest)

		// Count the total number of users
		total := db.Model(user).Association("Notifications").Count()

		// Paginate the results
		var notifications []models.Notification

		con := db
		if req.IncludeDeleted != nil && *req.IncludeDeleted {
			con = con.Unscoped()
		}

		con = con.Model(&notifications).Where(models.Notification{UserID: user.ID})
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

		con.Find(&notifications)

		response := struct {
			Data  []models.Notification `json:"data"`
			Total int64                 `json:"total"`
		}{
			Data:  notifications,
			Total: total,
		}

		return c.JSON(response)
	})

	// Create notification endpoint
	type notificationCreateRequest struct {
		Title   string  `json:"title" xml:"title" form:"title" validate:"required"`
		Message string  `json:"message" xml:"message" form:"message" validate:"required"`
		Link    *string `json:"link" xml:"link" form:"link" validate:"omitempty,url"`
		Group   *string `json:"group" xml:"group" form:"group" validate:"omitempty"`
	}
	notification_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[notificationCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		body := c.Locals("body").(notificationCreateRequest)

		if body.Link == nil {
			body.Link = new(string)
		}

		if body.Group == nil {
			body.Group = new(string)
		}

		notification := models.Notification{
			UserID:  user.ID,
			AddedBy: leash_auth.GetAuthentication(c).User.ID,
			Title:   body.Title,
			Message: body.Message,
			Link:    *body.Link,
			Group:   *body.Group,
		}

		db.Save(&notification)

		return c.JSON(notification)
	})

	user_notification_ep := notification_ep.Group("/:notification_id", userNotificationMiddleware)

	addCommonNotificationEndpoints(user_notification_ep)
}

// registerNotificationsEndpoints registers the endpoints for notifications
func registerNotificationsEndpoints(api fiber.Router) {
	notification_ep := api.Group("/notifications", leash_auth.ConcatPermissionPrefixMiddleware("notifications"))

	single_notification_ep := notification_ep.Group("/:notification_id", generalNotificationMiddleware)

	addCommonNotificationEndpoints(single_notification_ep)
}
