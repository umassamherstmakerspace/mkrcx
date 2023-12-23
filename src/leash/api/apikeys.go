package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"gorm.io/gorm"
)

func addUserUpdateEndpoints(user_ep fiber.Router, db *gorm.DB, keys leash_auth.Keys) {

}
