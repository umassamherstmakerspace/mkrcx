package leash_backend_api

import (
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
)

// Add more data to this type if needed
type client struct {
	isClosing bool
	mu        sync.Mutex
	auth      leash_auth.Authentication
}

type feedConnection struct {
	websocket *websocket.Conn
	auth      leash_auth.Authentication
}

var clients = make(map[*websocket.Conn]*client) // Note: although large maps with pointer-like types (e.g. strings) as keys are slow, using pointers themselves as keys is acceptable and fast
var register = make(chan *feedConnection)
var broadcast = make(chan string)
var unregister = make(chan *websocket.Conn)

func runHub() {
	for {
		select {
		case connection := <-register:
			clients[connection.websocket] = &client{
				auth: connection.auth,
			}
			log.Println("connection registered")

		case message := <-broadcast:
			log.Println("message received:", message)
			// Send the message to all clients
			for connection, c := range clients {
				go func(connection *websocket.Conn, c *client) { // send to each client in parallel so we don't block on a slow client
					c.mu.Lock()
					defer c.mu.Unlock()
					if c.isClosing {
						return
					}
					if err := connection.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
						c.isClosing = true
						log.Println("write error:", err)

						connection.WriteMessage(websocket.CloseMessage, []byte{})
						connection.Close()
						unregister <- connection
					}
				}(connection, c)
			}

		case connection := <-unregister:
			// Remove the client from the hub
			delete(clients, connection)

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

	go runHub()

	feed_ep.Get("/ws", websocket.New(func(c *websocket.Conn) {
		auth := c.Locals(leash_auth.CtxAuthKey).(leash_auth.Authentication)

		// When the function returns, unregister the client and close the connection
		defer func() {
			unregister <- c
			c.Close()
		}()

		// Register the client
		register <- &feedConnection{
			websocket: c,
			auth:      auth,
		}

		for {
			messageType, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Println("read error:", err)
				}

				return // Calls the deferred function, i.e. closes the connection on error
			}

			if messageType == websocket.TextMessage {
				// Broadcast the received message
				broadcast <- string(message)
			} else {
				log.Println("websocket message received of type", messageType)
			}
		}
	}))
}
