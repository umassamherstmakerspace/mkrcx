package leash_backend_api

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const (
	feedSocketAuthTimeout  = 5 * time.Second
	feedSocketWriteTimeout = 5 * time.Second
	feedSocketQueueSize    = 32
	feedSocketReadLimit    = 4096
	defaultFeedItemLimit   = 25
)

type feedSocket struct {
	id             uuid.UUID
	feedID         uint
	conn           *websocket.Conn
	authorization  string
	feedPermission string
	db             *gorm.DB
	keys           *leash_auth.Keys
	enforcer       leash_auth.EnforcerWrapper
	outbound       chan []byte
	done           chan struct{}
	closeOnce      sync.Once
}

func (s *feedSocket) close() {
	s.closeOnce.Do(func() {
		close(s.done)
		_ = s.conn.Close()
	})
}

func (s *feedSocket) authorize() error {
	authentication, err := leash_auth.AuthenticateHeader(
		s.authorization,
		s.db,
		s.keys,
		s.enforcer.Enforcer,
	)
	if err != nil {
		return err
	}
	if authentication.Authorize("leash.feeds:read") == nil {
		return nil
	}
	return authentication.Authorize(s.feedPermission)
}

func (s *feedSocket) writeLoop(remove func(uuid.UUID)) {
	defer remove(s.id)
	for {
		select {
		case <-s.done:
			return
		case payload := <-s.outbound:
			if s.authorize() != nil {
				_ = s.conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "feed access revoked"),
					time.Now().Add(feedSocketWriteTimeout),
				)
				s.close()
				return
			}
			_ = s.conn.SetWriteDeadline(time.Now().Add(feedSocketWriteTimeout))
			if err := s.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				s.close()
				return
			}
		}
	}
}

type feedHub struct {
	mu      sync.RWMutex
	sockets map[uint]map[uuid.UUID]*feedSocket
}

func newFeedHub() *feedHub {
	return &feedHub{sockets: make(map[uint]map[uuid.UUID]*feedSocket)}
}

func (h *feedHub) add(socket *feedSocket) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sockets[socket.feedID] == nil {
		h.sockets[socket.feedID] = make(map[uuid.UUID]*feedSocket)
	}
	h.sockets[socket.feedID][socket.id] = socket
}

func (h *feedHub) remove(id uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for feedID, sockets := range h.sockets {
		if socket, ok := sockets[id]; ok {
			delete(sockets, id)
			socket.close()
			if len(sockets) == 0 {
				delete(h.sockets, feedID)
			}
			return
		}
	}
}

func (h *feedHub) broadcast(item models.FeedMessage) error {
	payload, err := json.Marshal(fiber.Map{
		"type":    "feed_item.created",
		"feed_id": item.FeedID,
		"item":    item,
	})
	if err != nil {
		return err
	}

	h.mu.RLock()
	sockets := make([]*feedSocket, 0, len(h.sockets[item.FeedID]))
	for _, socket := range h.sockets[item.FeedID] {
		sockets = append(sockets, socket)
	}
	h.mu.RUnlock()

	// All sockets in a process normally share one enforcer. Refresh each unique
	// policy once per broadcast so revocations are observed without multiplying
	// database work by the number of connected displays.
	policyResults := make(map[*casbin.SyncedEnforcer]error)
	for _, socket := range sockets {
		enforcer := socket.enforcer.Enforcer
		if enforcer == nil {
			continue
		}
		if _, loaded := policyResults[enforcer]; !loaded {
			policyResults[enforcer] = enforcer.LoadPolicy()
		}
	}

	for _, socket := range sockets {
		if err := policyResults[socket.enforcer.Enforcer]; err != nil {
			socket.close()
			continue
		}
		select {
		case socket.outbound <- payload:
		default:
			// A cursor catch-up is authoritative, so a slow client can be safely evicted.
			socket.close()
		}
	}
	return nil
}

// FeedRuntime owns the process-local WebSocket hub and an optional cross-pod publisher.
// When publisher is nil, published items are delivered directly to local sockets.
type FeedRuntime struct {
	hub       *feedHub
	publisher func(models.FeedMessage) error
	close     func()
}

func NewLocalFeedRuntime() *FeedRuntime {
	return &FeedRuntime{hub: newFeedHub()}
}

func (r *FeedRuntime) publish(item models.FeedMessage) error {
	if r.publisher != nil {
		return r.publisher(item)
	}
	return r.hub.broadcast(item)
}

func (r *FeedRuntime) Close() {
	if r != nil && r.close != nil {
		r.close()
	}
}

func feedPermission(feed models.Feed, action string) string {
	return fmt.Sprintf("leash.feeds.%s:%s", feed.Name, action)
}

func authorizeFeed(authentication leash_auth.Authentication, feed models.Feed, action string) error {
	if authentication.Authorize("leash.feeds:"+action) == nil {
		return nil
	}
	return authentication.Authorize(feedPermission(feed, action))
}

func loadFeed(c *fiber.Ctx) (models.Feed, error) {
	feedID, err := strconv.ParseUint(c.Params("feed_id"), 10, 64)
	if err != nil || feedID == 0 {
		return models.Feed{}, fiber.NewError(fiber.StatusBadRequest, "invalid feed ID")
	}
	var feed models.Feed
	result := leash_auth.GetDB(c).First(&feed, uint(feedID))
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return models.Feed{}, fiber.NewError(fiber.StatusNotFound, "feed not found")
	}
	if result.Error != nil {
		return models.Feed{}, fiber.ErrInternalServerError
	}
	return feed, nil
}

func feedMiddleware(action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		feed, err := loadFeed(c)
		if err != nil {
			return err
		}
		if authorizeFeed(leash_auth.GetAuthentication(c), feed, action) != nil {
			return fiber.ErrForbidden
		}
		c.Locals("target_feed", feed)
		return c.Next()
	}
}

type feedCreateRequest struct {
	Name string `json:"name" validate:"required,max=64"`
}

var feedNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type feedItemRequest struct {
	LogLevel               uint       `json:"level" validate:"max=5"`
	Title                  string     `json:"title" validate:"required,notblank,max=160"`
	Message                string     `json:"message" validate:"required,notblank,max=2000"`
	UserID                 *uint      `json:"user" validate:"omitempty,min=1"`
	PendingUserSpecifier   *string    `json:"user_specifier" validate:"omitempty,max=64"`
	PendingCardFingerprint *string    `json:"-"`
	CheckinDecision        string     `json:"-"`
	OccurredAt             *time.Time `json:"-"`
}

type feedItemListRequest struct {
	AfterID  *uint `query:"after_id" validate:"omitempty,min=1"`
	BeforeID *uint `query:"before_id" validate:"omitempty,min=1"`
	Limit    *int  `query:"limit" validate:"omitempty,min=1,max=100"`
}

func sameFeedItemContent(first, second models.FeedMessage) bool {
	return first.AddedBy == second.AddedBy &&
		first.LogLevel == second.LogLevel &&
		first.UserID == second.UserID &&
		first.Title == second.Title &&
		first.Message == second.Message &&
		first.PendingUserSpecifier == second.PendingUserSpecifier &&
		first.CheckinDecision == second.CheckinDecision &&
		first.IdempotencyScope == second.IdempotencyScope &&
		(second.CreatedAt.IsZero() || first.CreatedAt.Equal(second.CreatedAt))
}

func feedIdempotencyScope(authentication leash_auth.Authentication) string {
	if authentication.IsAPIKey() {
		key := authentication.Data.(models.APIKey).Key
		digest := sha256.Sum256([]byte(key))
		return fmt.Sprintf("apikey:%x", digest)
	}
	return fmt.Sprintf("user:%d", authentication.User.ID)
}

func createFeed(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	req := c.Locals("body").(feedCreateRequest)
	req.Name = strings.TrimSpace(req.Name)
	if !feedNamePattern.MatchString(req.Name) {
		return fiber.NewError(fiber.StatusBadRequest, "feed name must be lower-kebab-case")
	}

	var count int64
	if err := db.Unscoped().Model(&models.Feed{}).Where("name = ?", req.Name).Count(&count).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	if count != 0 {
		return fiber.NewError(fiber.StatusConflict, "feed already exists")
	}

	feed := models.Feed{Name: req.Name}
	if err := db.Create(&feed).Error; err != nil {
		if db.Unscoped().Where("name = ?", req.Name).First(&models.Feed{}).Error == nil {
			return fiber.NewError(fiber.StatusConflict, "feed already exists")
		}
		return fiber.ErrInternalServerError
	}
	return c.Status(fiber.StatusCreated).JSON(feed)
}

func listFeeds(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	authentication := leash_auth.GetAuthentication(c)
	req := c.Locals("query").(listRequest)
	limit, offset := 10, 0
	if req.Limit != nil {
		limit = *req.Limit
	}
	if req.Offset != nil {
		offset = *req.Offset
	}

	query := db.Model(&models.Feed{}).Order("id ASC")
	if req.IncludeDeleted != nil && *req.IncludeDeleted {
		query = query.Unscoped()
	}
	var all []models.Feed
	if err := query.Find(&all).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	readable := make([]models.Feed, 0, len(all))
	for _, feed := range all {
		if authorizeFeed(authentication, feed, "read") == nil {
			readable = append(readable, feed)
		}
	}
	total := len(readable)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return c.JSON(fiber.Map{"data": readable[offset:end], "total": total})
}

func getFeed(c *fiber.Ctx) error {
	return c.JSON(c.Locals("target_feed").(models.Feed))
}

func listFeedItems(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	feed := c.Locals("target_feed").(models.Feed)
	req := c.Locals("query").(feedItemListRequest)
	items, err := queryFeedItems(db, feed, req)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": items})
}

func queryFeedItems(db *gorm.DB, feed models.Feed, req feedItemListRequest) ([]models.FeedMessage, error) {
	if req.AfterID != nil && req.BeforeID != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "after_id and before_id cannot be combined")
	}
	limit := defaultFeedItemLimit
	if req.Limit != nil {
		limit = *req.Limit
	}

	query := db.Where("feed_id = ?", feed.ID).Limit(limit)
	if feed.Name == checkinFeedName {
		query = query.Where("created_at >= ?", time.Now().UTC().Add(-checkinFeedRetention))
	}
	if req.AfterID != nil {
		query = query.Where("id > ?", *req.AfterID).Order("id ASC")
	} else {
		if req.BeforeID != nil {
			query = query.Where("id < ?", *req.BeforeID)
		}
		query = query.Order("id DESC")
	}
	var items []models.FeedMessage
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	if feed.Name == checkinFeedName {
		if err := hydrateFeedUserDisplayNames(db, items); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func hydrateFeedUserDisplayNames(db *gorm.DB, items []models.FeedMessage) error {
	ids := make([]uint, 0, len(items))
	seen := make(map[uint]struct{}, len(items))
	for index := range items {
		if items[index].UserID == 0 {
			continue
		}
		if _, exists := seen[items[index].UserID]; exists {
			continue
		}
		seen[items[index].UserID] = struct{}{}
		ids = append(ids, items[index].UserID)
	}
	if len(ids) == 0 {
		return nil
	}

	var users []struct {
		ID    uint
		Name  string
		Email string
	}
	if err := db.Model(&models.User{}).Select("id", "name", "email").Where("deleted_at IS NULL").Where("id IN ?", ids).Scan(&users).Error; err != nil {
		return err
	}
	names := make(map[uint]string, len(users))
	emails := make(map[uint]string, len(users))
	for _, user := range users {
		names[user.ID] = user.Name
		emails[user.ID] = user.Email
	}
	for index := range items {
		items[index].UserDisplayName = names[items[index].UserID]
		items[index].UserEmail = emails[items[index].UserID]
	}
	return nil
}

func hydrateFeedUserDisplayName(db *gorm.DB, item *models.FeedMessage) error {
	items := []models.FeedMessage{*item}
	if err := hydrateFeedUserDisplayNames(db, items); err != nil {
		return err
	}
	*item = items[0]
	return nil
}

func appendFeedItem(runtime *FeedRuntime) fiber.Handler {
	return func(c *fiber.Ctx) error {
		feed := c.Locals("target_feed").(models.Feed)
		req := c.Locals("body").(feedItemRequest)
		return persistFeedItem(c, runtime, feed, req, feedItemResponseFull)
	}
}

type feedItemResponseKind uint8

const (
	feedItemResponseFull feedItemResponseKind = iota
	feedItemResponseOpaque
	feedItemResponseCheckinDecision
)

func feedItemResponse(c *fiber.Ctx, item models.FeedMessage, status int, kind feedItemResponseKind) error {
	switch kind {
	case feedItemResponseOpaque:
		return c.Status(fiber.StatusNoContent).Send(nil)
	case feedItemResponseCheckinDecision:
		return sendCheckinDecision(c, item, status)
	default:
		return c.Status(status).JSON(item)
	}
}

func persistFeedItem(c *fiber.Ctx, runtime *FeedRuntime, feed models.Feed, req feedItemRequest, responseKind feedItemResponseKind) error {
	db := leash_auth.GetDB(c)
	authentication := leash_auth.GetAuthentication(c)

	if req.UserID != nil {
		var count int64
		if err := db.Model(&models.User{}).Where("id = ?", *req.UserID).Count(&count).Error; err != nil {
			return fiber.ErrInternalServerError
		}
		if count == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "user not found")
		}
	}

	item := models.FeedMessage{
		FeedID:           feed.ID,
		AddedBy:          authentication.User.ID,
		LogLevel:         req.LogLevel,
		Title:            strings.TrimSpace(req.Title),
		Message:          strings.TrimSpace(req.Message),
		CheckinDecision:  req.CheckinDecision,
		IdempotencyScope: feedIdempotencyScope(authentication),
	}
	if req.OccurredAt != nil {
		item.CreatedAt = req.OccurredAt.UTC()
	}
	if req.UserID != nil {
		item.UserID = *req.UserID
	}
	if req.PendingUserSpecifier != nil {
		item.PendingUserSpecifier = strings.TrimSpace(*req.PendingUserSpecifier)
	}
	item.PendingCardFingerprint = req.PendingCardFingerprint

	if idempotencyKey := strings.TrimSpace(c.Get("Idempotency-Key")); idempotencyKey != "" {
		if len(idempotencyKey) > 128 {
			return fiber.NewError(fiber.StatusBadRequest, "Idempotency-Key is too long")
		}
		item.IdempotencyKey = &idempotencyKey
		var existing models.FeedMessage
		result := db.Where(
			"feed_id = ? AND idempotency_scope = ? AND idempotency_key = ?",
			feed.ID, item.IdempotencyScope, idempotencyKey,
		).First(&existing)
		if result.Error == nil {
			if !sameFeedItemContent(existing, item) {
				return fiber.NewError(fiber.StatusConflict, "Idempotency-Key was already used for different content")
			}
			if feed.Name == checkinFeedName {
				if err := hydrateFeedUserDisplayName(db, &existing); err != nil {
					return fiber.ErrInternalServerError
				}
			}
			return feedItemResponse(c, existing, fiber.StatusOK, responseKind)
		}
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fiber.ErrInternalServerError
		}
	}

	if err := db.Create(&item).Error; err != nil {
		if item.IdempotencyKey != nil {
			var existing models.FeedMessage
			if db.Where(
				"feed_id = ? AND idempotency_scope = ? AND idempotency_key = ?",
				feed.ID, item.IdempotencyScope, *item.IdempotencyKey,
			).First(&existing).Error == nil {
				if !sameFeedItemContent(existing, item) {
					return fiber.NewError(fiber.StatusConflict, "Idempotency-Key was already used for different content")
				}
				if feed.Name == checkinFeedName {
					if err := hydrateFeedUserDisplayName(db, &existing); err != nil {
						return fiber.ErrInternalServerError
					}
				}
				return feedItemResponse(c, existing, fiber.StatusOK, responseKind)
			}
		}
		return fiber.ErrInternalServerError
	}
	if feed.Name == checkinFeedName {
		if err := hydrateFeedUserDisplayName(db, &item); err != nil {
			return fiber.ErrInternalServerError
		}
	}
	// The row is authoritative. A failed live signal is repaired by cursor catch-up.
	if err := runtime.publish(item); err != nil {
		log.Printf("feed item %d committed but live delivery failed: %v", item.ID, err)
	}
	return feedItemResponse(c, item, fiber.StatusCreated, responseKind)
}

func archiveFeed(c *fiber.Ctx) error {
	feed := c.Locals("target_feed").(models.Feed)
	if err := leash_auth.GetDB(c).Delete(&feed).Error; err != nil {
		return fiber.ErrInternalServerError
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func websocketFeedEndpoint(feeds fiber.Router, runtime *FeedRuntime) {
	feeds.Use("/:feed_id/ws", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		feedID, err := strconv.ParseUint(c.Params("feed_id"), 10, 64)
		if err != nil || feedID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "invalid feed ID")
		}
		c.Locals("ws_feed_id", uint(feedID))
		return c.Next()
	})

	feeds.Get("/:feed_id/ws", func(c *fiber.Ctx) error {
		feedID := c.Locals("ws_feed_id").(uint)
		db := leash_auth.GetDB(c)
		keys := leash_auth.GetKeys(c)
		enforcer := leash_auth.EnforcerWrapper{Enforcer: leash_auth.GetEnforcer(c)}

		return websocket.New(func(conn *websocket.Conn) {
			defer conn.Close()
			conn.SetReadLimit(feedSocketReadLimit)
			_ = conn.SetReadDeadline(time.Now().Add(feedSocketAuthTimeout))
			messageType, message, err := conn.ReadMessage()
			if err != nil || messageType != websocket.TextMessage {
				return
			}

			authorization := string(message)
			if err := enforcer.Enforcer.LoadPolicy(); err != nil {
				_ = conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "authorization unavailable"),
					time.Now().Add(feedSocketWriteTimeout),
				)
				return
			}
			authentication, err := leash_auth.AuthenticateHeader(authorization, db, keys, enforcer.Enforcer)
			if err != nil {
				_ = conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "failed to authenticate"),
					time.Now().Add(feedSocketWriteTimeout),
				)
				return
			}

			// Do not reveal feed existence before the first-frame credential is valid.
			var feed models.Feed
			result := db.First(&feed, feedID)
			if result.Error != nil || authorizeFeed(authentication, feed, "read") != nil {
				_ = conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "failed to authenticate"),
					time.Now().Add(feedSocketWriteTimeout),
				)
				return
			}

			socket := &feedSocket{
				id: uuid.New(), feedID: feed.ID, conn: conn, authorization: authorization,
				feedPermission: feedPermission(feed, "read"),
				db:             db, keys: keys, enforcer: enforcer,
				outbound: make(chan []byte, feedSocketQueueSize), done: make(chan struct{}),
			}
			runtime.hub.add(socket)
			ready, err := json.Marshal(fiber.Map{"type": "feed.ready", "feed_id": feed.ID})
			if err != nil {
				runtime.hub.remove(socket.id)
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(feedSocketWriteTimeout))
			if err := conn.WriteMessage(websocket.TextMessage, ready); err != nil {
				runtime.hub.remove(socket.id)
				return
			}
			go socket.writeLoop(runtime.hub.remove)
			defer runtime.hub.remove(socket.id)

			_ = conn.SetReadDeadline(time.Time{})
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		})(c)
	})
}

func registerFeedEndpoints(api fiber.Router, runtime *FeedRuntime) {
	feeds := api.Group("/feeds", leash_auth.ConcatPermissionPrefixMiddleware("feeds"))
	feeds.Use(func(c *fiber.Ctx) error {
		if err := leash_auth.GetEnforcer(c).LoadPolicy(); err != nil {
			log.Printf("unable to refresh feed authorization policy: %v", err)
			return fiber.ErrInternalServerError
		}
		return c.Next()
	})

	feeds.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[feedCreateRequest], createFeed)
	feeds.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), models.GetQueryMiddleware[listRequest], listFeeds)

	feeds.Get("/:feed_id", feedMiddleware("read"), getFeed)
	feeds.Delete("/:feed_id", leash_auth.PrefixAuthorizationMiddleware("delete"), feedMiddleware("manage"), archiveFeed)
	feeds.Get("/:feed_id/items", feedMiddleware("read"), models.GetQueryMiddleware[feedItemListRequest], listFeedItems)
	feeds.Post("/:feed_id/items", feedMiddleware("append"), models.GetBodyMiddleware[feedItemRequest], appendFeedItem(runtime))
}
