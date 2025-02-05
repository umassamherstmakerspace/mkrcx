package leash_backend_api

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

type connList struct {
	mu   sync.Mutex
	conn map[uuid.UUID]*websocket.Conn
}

func (c *connList) Add(connection *websocket.Conn) uuid.UUID {
	c.mu.Lock()
	defer c.mu.Unlock()
	u := uuid.New()
	c.conn[u] = connection
	return u
}

func (c *connList) Remove(u uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.conn, u)
}

func (c *connList) SendAll(messageType int, message []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	for _, c := range c.conn {
		err = c.WriteMessage(messageType, message)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *connList) DisconnectAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	for _, c := range c.conn {
		err = c.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// feedMiddleware is a middleware that sets the target feed to the user specified in the URL
func feedMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	authentication := leash_auth.GetAuthentication(c)
	// Check if the user is authorized to perform the action
	if authentication.Authorize("leash.feeds:target") != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to target feeds")
	}

	// Get the feed ID from the URL
	feed_id, err := strconv.Atoi(c.Params("feed_id"))

	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid feed ID")
	}

	var feed = models.Feed{
		ID: uint(feed_id),
	}

	if res := db.Limit(1).Where(&feed).Find(&feed); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Feed not found")
	}

	c.Locals("target_feed", feed)
	return c.Next()
}

// createCommonFeedEndpoints creates the common endpoints for the feed endpoint
func createCommonFeedEndpoints(feed_ep fiber.Router) {
	// Create a new user endpoint
	type feedCreateRequest struct {
		Name string `json:"name" xml:"name" form:"name" validate:"required"`
	}
	feed_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[feedCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		req := c.Locals("body").(feedCreateRequest)

		// Check if the feed already exists
		var existsing_feed = models.Feed{
			Name: req.Name,
		}

		if res := db.Limit(1).Where(&existsing_feed).Find(&existsing_feed); res.Error == nil && res.RowsAffected != 0 {
			return fiber.NewError(fiber.StatusConflict, "Feed already exists")
		}

		// Create a new user in the database
		feed := models.Feed{
			Name: req.Name,
		}

		db.Create(&feed)

		return c.JSON(feed)
	})

	// List feeds endpoint
	feed_ep.Get("/search", leash_auth.PrefixAuthorizationMiddleware("list"), leash_auth.AuthorizationMiddleware("leash.feeds:target"), models.GetQueryMiddleware[listRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		req := c.Locals("query").(listRequest)

		var feeds []models.Feed

		con := db.Model(&models.Feed{})

		// Count the total number of feeds
		total := int64(0)
		con.Model(&models.Feed{}).Count(&total)

		// Paginate the results
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

		con.Find(&feeds)

		response := struct {
			Data  []models.Feed `json:"data"`
			Total int64         `json:"total"`
		}{
			Data:  feeds,
			Total: total,
		}

		return c.JSON(response)
	})
}

type feedGetRequest struct {
	MessageCount  *int   `query:"messages" validate:"omitempty"`
	MessageBefore string `query:"with_notifications" validate:"omitempty"`
}

func getFeedEndpoint(feed_ep fiber.Router) {
	// Get the current feed endpoint
	feed_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), models.GetQueryMiddleware[feedGetRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		feed := c.Locals("target_feed").(models.Feed)

		feed.Messages = []models.FeedMessage{}
		db.Model(&feed).Association("Messages").Find(&feed.Messages)

		return c.JSON(feed)
	})
}

// deleteUserEndpoints creates the endpoints for deleting feeds
func deleteFeedEndpoint(feed_ep fiber.Router) {
	feed_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		feed := c.Locals("target_feed").(models.User)
		db := leash_auth.GetDB(c)

		db.Delete(&feed)
		db.Delete(&models.FeedMessage{}, "feed_id = ?", feed.ID)

		return c.SendStatus(fiber.StatusOK)
	})
}

// websocketFeedEndpoint creates the endpoint for the websocket
func websocketFeedEndpoint(feed_ep fiber.Router, authenticator leash_auth.Authentication) {
	feed_ep.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	feed_ep.Get("/ws", websocket.New(func(c *websocket.Conn) {
		user_permissions, err := authenticator.Enforcer.Enforcer.GetPermissionsForUser("user:" + fmt.Sprint(user.ID))
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		role_permissions, err := authenticator.Enforcer.Enforcer.GetPermissionsForUser("role:" + user.Role)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		permissions := make([]string, len(user_permissions)+len(role_permissions))

		for i, perm := range user_permissions {
			permissions[i] = perm[1]
		}

		for i, perm := range role_permissions {
			permissions[i+len(user_permissions)] = perm[1]
		}

		return c.JSON(permissions)
	}))
}

// registerFeedEndpoints registers all the Feed endpoints for Leash
func registerFeedEndpoints(api fiber.Router) {
	feeds_ep := api.Group("/feed", leash_auth.ConcatPermissionPrefixMiddleware("feeds"))

	createBaseEndpoints(feeds_ep)
	websocketFeedEndpoint(feeds_ep, leash_auth.GetAuthentication())

	feed_ep := feeds_ep.Group("/:feed_id")
	getFeedEndpoint(feed_ep)
	deleteFeedEndpoint(feed_ep)
	createCommonFeedEndpoints(feed_ep)
}
