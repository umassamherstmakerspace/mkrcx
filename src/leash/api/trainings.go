package leash_backend_api

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

// userTrainingMiddlware is a middleware that fetches the training from a user and stores it in the context
func userTrainingMiddlware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	user := c.Locals("target_user").(models.User)
	var training = models.Training{
		UserID:       user.ID,
		TrainingType: c.Params("training_type"),
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

	training_id, err := strconv.Atoi(c.Params("training_id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid training ID")
	}

	var training = models.Training{}
	training.ID = uint(training_id)

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
		return c.SendStatus(fiber.StatusNoContent)
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
		con := db.Model(&trainings).Where(models.Training{UserID: user.ID})
		if req.LoadAll != nil && *req.LoadAll {
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
		}

		con.Find(&trainings)

		response := struct {
			Trainings []models.Training `json:"trainings"`
			Total     int64             `json:"total"`
		}{
			Trainings: trainings,
			Total:     total,
		}

		return c.JSON(response)
	})

	// Create training endpoint
	type trainingCreateRequest struct {
		TrainingType string `json:"training_type" xml:"training_type" form:"training_type" validate:"required"`
	}
	training_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[trainingCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		authenticator := leash_auth.GetAuthentication(c)
		req := c.Locals("body").(trainingCreateRequest)

		// Check if training already exists for user
		var existingTraining = models.Training{
			UserID:       user.ID,
			TrainingType: req.TrainingType,
		}
		if res := db.Limit(1).Where(&existingTraining).Find(&existingTraining); res.Error == nil && res.RowsAffected != 0 {
			return fiber.NewError(fiber.StatusConflict, "User already has this training")
		}

		training := models.Training{
			TrainingType: req.TrainingType,
			AddedBy:      authenticator.User.ID,
		}

		db.Model(&user).Association("Trainings").Append(&training)

		return c.JSON(training)
	})

	user_training_ep := training_ep.Group("/:training_type", userTrainingMiddlware)

	addCommonTrainingEndpoints(user_training_ep)
}

// registerTrainingEndpoints registers the training endpoints
func registerTrainingEndpoints(api fiber.Router) {
	trainings_ep := api.Group("/trainings", leash_auth.ConcatPermissionPrefixMiddleware("trainings"))

	single_training_ep := trainings_ep.Group("/:training_id", generalTrainingMiddleware)

	addCommonTrainingEndpoints(single_training_ep)
}
