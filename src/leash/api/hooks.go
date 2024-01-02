package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

type UserEvent struct {
	c         *fiber.Ctx
	Target    models.User `json:"target"`
	Agent     models.User `json:"agent"`
	Timestamp int64       `json:"time"`
}

// GetCtx returns the context of the event
func (e *UserEvent) GetCtx() *fiber.Ctx {
	return e.c
}

type UserChanges struct {
	Old   string `json:"old"`
	New   string `json:"new"`
	Field string `json:"field"`
}

type UserUpdateEvent struct {
	UserEvent
	Changes []UserChanges `json:"changes"`
}
