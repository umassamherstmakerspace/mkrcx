package leash_backend_api

import (
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

// userTrainingMiddleware is a middleware that fetches the training from a user and stores it in the context
func userTrainingMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	user := c.Locals("target_user").(models.User)
	authentication := leash_auth.GetAuthentication(c)
	permissionPrefix := c.Locals("permission_prefix").(string)

	// Check if the user is authorized to perform the action
	if authentication.Authorize(permissionPrefix+":target") != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to read this user's trainings")
	}

	training_name, err := url.QueryUnescape(c.Params("training_name"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid training type")
	}

	var training = models.Training{
		UserID: user.ID,
		Name:   training_name,
	}

	if res := db.Limit(1).Where(&training).Find(&training); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Training not found")
	}

	c.Locals("training", training)

	return c.Next()
}

// generalTrainingMiddleware is a middleware that fetches the training by ID and stores it in the context
func generalTrainingMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	authentication := leash_auth.GetAuthentication(c)

	// Check if the user is authorized to perform the action
	if authentication.Authorize("leash.trainings:target") != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "You are not authorized to read trainings")
	}

	training_id, err := strconv.Atoi(c.Params("training_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid training ID")
	}

	var training = models.Training{
		ID: uint(training_id),
	}

	if res := db.Limit(1).Where(&training).Find(&training); res.Error != nil || res.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "Training not found")
	}

	c.Locals("training", training)

	return c.Next()
}

// addCommonTrainingEndpoints adds the common endpoints for training
func addCommonTrainingEndpoints(training_ep fiber.Router) {
	// Get current training endpoint
	training_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), func(c *fiber.Ctx) error {
		training := c.Locals("training").(models.Training)
		return c.JSON(training)
	})

	// Delete current training endpoint
	training_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		training := c.Locals("training").(models.Training)
		training.RemovedBy = leash_auth.GetAuthentication(c).User.ID

		db.Save(&training)

		db.Delete(&training)
		return c.SendStatus(fiber.StatusOK)
	})
}

// addUserTrainingEndpoints adds the endpoints for user training
func addUserTrainingEndpoints(user_ep fiber.Router) {
	training_ep := user_ep.Group("/trainings", leash_auth.ConcatPermissionPrefixMiddleware("trainings"))

	// List trainings endpoint
	training_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), models.GetQueryMiddleware[listRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		req := c.Locals("query").(listRequest)

		// Count the total number of users
		total := db.Model(user).Association("Trainings").Count()

		// Paginate the results
		var trainings []models.Training

		con := db
		if req.IncludeDeleted != nil && *req.IncludeDeleted {
			con = con.Unscoped()
		}

		con = con.Model(&trainings).Where(models.Training{UserID: user.ID})
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

		con.Find(&trainings)

		response := struct {
			Data  []models.Training `json:"data"`
			Total int64             `json:"total"`
		}{
			Data:  trainings,
			Total: total,
		}

		return c.JSON(response)
	})

	// Create training endpoint
	type trainingCreateRequest struct {
		Name  string `json:"name" xml:"name" form:"name" validate:"required"`
		Level string `json:"level" xml:"level" form:"level" validate:"required,oneof=in_progress supervised unsupervised can_train"`
	}
	training_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[trainingCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		authenticator := leash_auth.GetAuthentication(c)
		req := c.Locals("body").(trainingCreateRequest)

		// Check if training already exists for user
		var existingTraining = models.Training{
			UserID: user.ID,
			Name:   req.Name,
		}
		if res := db.Limit(1).Where(&existingTraining).Find(&existingTraining); res.Error == nil && res.RowsAffected != 0 {
			return fiber.NewError(fiber.StatusConflict, "User already has this training")
		}

		training := models.Training{
			Name:    req.Name,
			Level:   req.Level,
			AddedBy: authenticator.User.ID,
		}

		db.Model(&user).Association("Trainings").Append(&training)

		return c.JSON(training)
	})

	user_training_ep := training_ep.Group("/:training_name", userTrainingMiddleware)

	addCommonTrainingEndpoints(user_training_ep)
}

// registerTrainingEndpoints registers the training endpoints
func registerTrainingEndpoints(api fiber.Router) {
	trainings_ep := api.Group("/trainings", leash_auth.ConcatPermissionPrefixMiddleware("trainings"))

	single_training_ep := trainings_ep.Group("/:training_id", generalTrainingMiddleware)

	addCommonTrainingEndpoints(single_training_ep)
}
