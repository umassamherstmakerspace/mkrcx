package leash_backend_api

import "github.com/mkrcx/mkrcx/src/shared/models"

type UserEvent struct {
	Target    models.User `json:"target"`
	Agent     models.User `json:"agent"`
	Timestamp int64       `json:"time"`
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
