package leash_backend_api

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

// Add more data to this type if needed
type feedWSClient struct {
	isClosing bool
	mu        sync.Mutex
	auth      leash_auth.Authentication
	feed      models.Feed
}

type feedWSConnection struct {
	feed      models.Feed
	websocket *websocket.Conn
	auth      leash_auth.Authentication
}

var feedWSClients = make(map[*websocket.Conn]*feedWSClient)
var feedWSRegister = make(chan *feedWSConnection)
var feedWSBroadcast = make(chan models.FeedItem)
var feedWSUnregister = make(chan *websocket.Conn)

func runFeedWebSocketHandler() {
	for {
		select {
		case connection := <-feedWSRegister:
			feedWSClients[connection.websocket] = &feedWSClient{
				auth: connection.auth,
				feed: connection.feed,
			}
			log.Println("connection registered")

		case message := <-feedWSBroadcast:
			messageJSON, err := json.Marshal(message)
			if err != nil {
				log.Println("json error:", err)
				continue
			}

			// Send the message to all clients
			for connection, c := range feedWSClients {
				go func(connection *websocket.Conn, c *feedWSClient) { // send to each client in parallel so we don't block on a slow client
					c.mu.Lock()
					defer c.mu.Unlock()
					if c.isClosing {
						return
					}

					// Check if the client is looking at the feed
					if c.feed.ID != message.FeedID {
						return
					}

					// Check if the client has the permission to see the message
					if err := c.auth.Authorize("feeds:read"); err != nil {
						c.isClosing = true

						connection.WriteMessage(websocket.CloseMessage, []byte{})
						connection.Close()
						feedWSUnregister <- connection

						return
					}

					if err := connection.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
						c.isClosing = true
						log.Println("write error:", err)

						connection.WriteMessage(websocket.CloseMessage, []byte{})
						connection.Close()
						feedWSUnregister <- connection
					}
				}(connection, c)
			}

		case connection := <-feedWSUnregister:
			// Remove the client from the hub
			delete(feedWSClients, connection)

			log.Println("connection unregistered")
		}
	}
}

// registerFeedEndpoints registers the feed endpoints
func registerFeedEndpoints(api fiber.Router) {
	feed_ep := api.Group("/feeds", leash_auth.ConcatPermissionPrefixMiddleware("feeds"))

	// feed_ep.Use(func(c *fiber.Ctx) error {
	// 	if websocket.IsWebSocketUpgrade(c) { // Returns true if the client requested upgrade to the WebSocket protocol
	// 		return c.Next()
	// 	}
	// 	return c.SendStatus(fiber.StatusUpgradeRequired)
	// })

	go runFeedWebSocketHandler()

	feed_ep.Get("/ws", websocket.New(func(c *websocket.Conn) {
		auth := c.Locals(leash_auth.CtxAuthKey).(leash_auth.Authentication)

		// When the function returns, unregister the client and close the connection
		defer func() {
			feedWSUnregister <- c
			c.Close()
		}()

		// Register the client
		feedWSRegister <- &feedWSConnection{
			websocket: c,
			auth:      auth,
		}

		for {
			messageType, _, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Println("read error:", err)
				}

				return // Calls the deferred function, i.e. closes the connection on error
			}

			if messageType == websocket.CloseMessage {
				return // Calls the deferred function, i.e. closes the connection
			}
		}
	}))
}
