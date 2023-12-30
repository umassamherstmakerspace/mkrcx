package leash_backend_api

import (
	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
)

func userTrainingMiddlware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	user := c.Locals("target_user").(models.User)
	var training models.Training
	if err := db.Model(&user).Where("training_type = ?", c.Params("training_type")).Association("Trainings").Find(&training); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Training not found")
	}
	c.Locals("training", training)

	return c.Next()
}

func generalTrainingMiddleware(c *fiber.Ctx) error {
	db := leash_auth.GetDB(c)
	var training models.Training
	if err := db.Where("id = ?", c.Params("training_id")).First(&training).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Training not found")
	}
	c.Locals("training", training)

	return c.Next()
}

func addCommonTrainingEndpoints(training_ep fiber.Router) {
	training_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("get"), func(c *fiber.Ctx) error {
		training := c.Locals("training").(models.Training)
		return c.JSON(training)
	})

	training_ep.Delete("/", leash_auth.PrefixAuthorizationMiddleware("delete"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		training := c.Locals("training").(models.Training)
		training.RemovedBy = leash_auth.GetAuthentication(c).User.ID

		db.Save(&training)

		db.Delete(&training)
		return c.SendStatus(fiber.StatusNoContent)
	})
}

func addUserTrainingEndpoints(user_ep fiber.Router) {
	training_ep := user_ep.Group("/trainings", leash_auth.PrefixAuthorizationMiddleware("trainings"))

	training_ep.Get("/", leash_auth.PrefixAuthorizationMiddleware("list"), func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		var trainings []models.Training
		db.Model(&user).Association("Trainings").Find(&trainings)
		return c.JSON(trainings)
	})

	type trainingCreateRequest struct {
		TrainingType string `json:"training_type" xml:"training_type" form:"training_type" validate:"required"`
	}
	training_ep.Post("/", leash_auth.PrefixAuthorizationMiddleware("create"), models.GetBodyMiddleware[trainingCreateRequest], func(c *fiber.Ctx) error {
		db := leash_auth.GetDB(c)
		user := c.Locals("target_user").(models.User)
		authenticator := leash_auth.GetAuthentication(c)
		req := c.Locals("body").(trainingCreateRequest)

		// Check if training already exists for user
		var existingTraining models.Training
		db.Model(&user).Where("training_type = ?", req.TrainingType).Association("Trainings").Find(&existingTraining)

		if existingTraining.ID != 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Training already exists for user")
		}

		training := models.Training{
			TrainingType: req.TrainingType,
			AddedBy:      authenticator.User.ID,
		}

		db.Model(&user).Association("Trainings").Append(&training)

		return c.SendStatus(fiber.StatusCreated)
	})

	user_training_ep := training_ep.Group("/:training_type", userTrainingMiddlware)

	addCommonTrainingEndpoints(user_training_ep)
}

func registerTrainingEndpoints(api fiber.Router) {
	trainings_ep := api.Group("/trainings", leash_auth.PrefixAuthorizationMiddleware("trainings"))

	single_training_ep := trainings_ep.Group("/:training_id", generalTrainingMiddleware)

	addCommonTrainingEndpoints(single_training_ep)
}
