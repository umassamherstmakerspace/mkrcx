package leash_backend_api

import (
	"encoding/json"
	"strconv"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

type websocketConn struct {
	ws   *websocket.Conn
	auth leash_auth.Authentication
}

type connList struct {
	mu   sync.Mutex
	conn map[uuid.UUID]websocketConn
}

func (c *connList) Add(connection *websocket.Conn, auth leash_auth.Authentication) uuid.UUID {
	c.mu.Lock()
	defer c.mu.Unlock()
	u := uuid.New()
	c.conn[u] = websocketConn{ws: connection, auth: auth}
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
		err = c.auth.Authorize("leash.feeds:ws")
		if err == nil {
			err = c.ws.WriteMessage(messageType, message)
			if err != nil {
				return err
			}
		} else {
			c.ws.Close()
		}
	}

	return nil
}

func (c *connList) DisconnectAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	for _, c := range c.conn {
		err = c.ws.Close()
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

// createBaseFeedEndpoints creates the base endpoints for the feed endpoint
func createBaseFeedEndpoints(feed_ep fiber.Router) {
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
	feed_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), leash_auth.AuthorizationMiddleware("leash.feeds:target"), models.GetQueryMiddleware[listRequest], func(c *fiber.Ctx) error {
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

// createCommonFeedEndpoints creates the common endpoints for the feed endpoint
func createCommonFeedEndpoints(feed_ep fiber.Router, websocketConnections *connList) {
	// Get the current feed endpoint
	type feedGetRequest struct {
		MessageCount  *int   `query:"messages" validate:"omitempty"`
		MessageBefore string `query:"with_notifications" validate:"omitempty"`
	}
	feed_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), models.GetQueryMiddleware[feedGetRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		feed := c.Locals("target_feed").(models.Feed)

		feed.Messages = []models.FeedMessage{}
		db.Model(&feed).Association("Messages").Find(&feed.Messages)

		return c.JSON(feed)
	})

	// Get the current feed endpoint
	type feedPostRequest struct {
		LogLevel             uint    `json:"level" xml:"level" form:"level" validate:"required,numeric"`
		Title                string  `json:"title" xml:"title" form:"title" validate:"required"`
		Message              string  `json:"message" xml:"message" form:"message" validate:"required"`
		UserID               *uint   `json:"user" xml:"user" form:"user" validate:"omitempty,min=1,numeric"`
		PendingUserSpecifier *string `json:"user_specifier" xml:"user_specifier" form:"user_specifier" validate:"omitempty"`
		PendingUserData      *string `json:"user_data" xml:"user_data" form:"user_data" validate:"omitempty"`
	}

	feed_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("get"), models.GetBodyMiddleware[feedPostRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		feed := c.Locals("target_feed").(models.Feed)
		req := c.Locals("body").(feedPostRequest)
		authenticator := leash_auth.GetAuthentication(c)

		message := models.FeedMessage{
			FeedId:   feed.ID,
			AddedBy:  authenticator.User.ID,
			LogLevel: req.LogLevel,
			Title:    req.Title,
			Message:  req.Message,
		}

		if req.UserID != nil {
			message.UserID = *req.UserID
		}

		if req.PendingUserData != nil && req.PendingUserSpecifier != nil {
			message.PendingUserData = *req.PendingUserData
			message.PendingUserSpecifier = *req.PendingUserSpecifier
		}

		data, err := json.Marshal(feed)
		if err != nil {
			return fiber.ErrInternalServerError
		}

		db.Create(&message)

		websocketConnections.SendAll(websocket.TextMessage, data)

		return c.JSON(feed)
	})

	feed_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		feed := c.Locals("target_feed").(models.User)
		db := leash_auth.GetDB(c)

		db.Delete(&feed)
		db.Delete(&models.FeedMessage{}, "feed_id = ?", feed.ID)

		return c.SendStatus(fiber.StatusOK)
	})
}

// websocketFeedEndpoint creates the endpoint for the websocket
func websocketFeedEndpoint(feed_ep fiber.Router, websocketConnections *connList) {
	feed_ep.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	feed_ep.Get("/ws", func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		keys := leash_auth.GetKeys(c)
		e := leash_auth.GetEnforcer(c)

		return websocket.New(func(conn *websocket.Conn) {
			defer conn.Close()
			var u uuid.UUID
			authenticated := false
			for {
				mt, msg, err := conn.ReadMessage()
				if err != nil {
					break
				}

				if authenticated {
					// Use message recieved after authenticated
				} else {
					if mt == websocket.TextMessage {
						authentication, err := leash_auth.AuthenticateHeader(string(msg), db, keys, e)
						if err != nil || authentication.Authorize("leash.feeds:ws") != nil {
							conn.WriteMessage(websocket.TextMessage, []byte("Fail to authenticate"))
							break
						}

						websocketConnections.Add(conn, authentication)
						authenticated = true
					}
				}
			}

			if authenticated {
				websocketConnections.Remove(u)
			}
		})(c)
	})
}

// registerFeedEndpoints registers all the Feed endpoints for Leash
func registerFeedEndpoints(api fiber.Router) {
	feeds_ep := api.Group("/feeds", leash_auth.ConcatPermissionPrefixMiddleware("feeds"))

	websocketConnections := connList{
		conn: make(map[uuid.UUID]websocketConn),
	}

	createBaseFeedEndpoints(feeds_ep)
	websocketFeedEndpoint(feeds_ep, &websocketConnections)

	feed_ep := feeds_ep.Group("/:feed_id", feedMiddleware)
	createCommonFeedEndpoints(feed_ep, &websocketConnections)
}
