package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	models "github.com/spectrum-control/spectrum/src/shared/models"
)

type ctxUserKey struct{}

const host = ":8000"

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&models.APIKey{})
	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Training{})

	// Debug
	makeMakerspaceSystemUser(db)

	r := mux.NewRouter()

	// Create a new user
	r.HandleFunc("/users", apiKeyAuthMiddleware(db, func(w http.ResponseWriter, r *http.Request) {
		// Get api user from the request context
		apiUser := r.Context().Value(ctxUserKey{}).(models.User)

		// Make sure API user is system user
		if apiUser.Email != "makerspace@umass.edu" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Unauthorized")
			return
		}

		// Get the user's email and name from the request body
		email := r.FormValue("email")
		name := r.FormValue("name")

		// Check if the user already exists
		{
			var user models.User
			res := db.First(&user, "email = ?", email)
			if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
				// The user already exists
				w.WriteHeader(http.StatusConflict)
				fmt.Fprintf(w, "User already exists")
				return
			}
		}

		// Create a new user in the database
		user := models.User{
			Email:   email,
			Name:    name,
			ID:      uuid.NewString(),
			Enabled: false,
		}
		db.Create(&user)

		// Write a success message to the response
		fmt.Fprintf(w, "User created successfully")
	})).Methods(http.MethodPost)

	// Add completed training to a user
	r.HandleFunc("/training", apiKeyAuthMiddleware(db, func(w http.ResponseWriter, r *http.Request) {
		// Get api user from the request context
		apiUser := r.Context().Value(ctxUserKey{}).(models.User)

		if !apiUser.Admin {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Unauthorized")
			return
		}

		// Get the user's email and training type from the request body
		email := r.FormValue("email")
		trainingType := r.FormValue("training_type")

		// Check if the user exists
		var user models.User
		res := db.First(&user, "email = ?", email)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "User not found")
			return
		}

		// Create a new training in the database
		training := models.Training{
			User:         user,
			UserID:       user.ID,
			TrainingType: trainingType,
			AddedBy:      apiUser.ID,
		}

		db.Create(&training)

		// If the user has completed the trainings "orientation" and "docusign", enable the user
		userTrainingEnable(db, user)

		// Write a success message to the response
		fmt.Fprintf(w, "Training added successfully")
	})).Methods(http.MethodPost)

	// Used for debugging
	// List all users
	r.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		var users []models.User
		db.Find(&users)

		// Write the users to the response as json
		msg, err := json.Marshal(users)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error marshalling users")
			return
		}
		fmt.Fprintf(w, "%s", msg)
	}).Methods(http.MethodGet)

	log.Printf("Starting server on port %s\n", host)
	http.ListenAndServe(host, r)
}

func userTrainingEnable(db *gorm.DB, user models.User) {
	var trainings []models.Training
	db.Find(&trainings, "user_id = ?", user.ID)

	orientationCompleted := false
	docusignCompleted := false
	for _, training := range trainings {
		if training.TrainingType == "orientation" {
			orientationCompleted = true
		}
		if training.TrainingType == "docusign" {
			docusignCompleted = true
		}
	}

	if orientationCompleted && docusignCompleted {
		user.Enabled = true
		db.Save(&user)
	}
}

func makeMakerspaceSystemUser(db *gorm.DB) {
	var systemUser models.User
	res := db.First(&systemUser, "email = ?", "makerspace@umass.edu")
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		// Create a system user
		user := models.User{
			Email:   "makerspace@umass.edu",
			Name:    "System",
			ID:      uuid.NewString(),
			Admin:   true,
			Enabled: true,
		}

		// Create a default API key
		apiKey := models.APIKey{
			Key:         uuid.NewString(),
			User:        user,
			UserID:      user.ID,
			Description: "Default API key",
		}
		db.Create(&user)
		db.Create(&apiKey)

		log.Printf("API key: %s\n", apiKey.Key)
		return
	}
}

func apiKeyAuthMiddleware(db *gorm.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the API key from the request header
		apiKey := r.Header.Get("API-Key")

		var apiKeyRecord models.APIKey
		res := db.First(&apiKeyRecord, "key = ?", apiKey)

		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The API key is not valid
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Invalid API key")
			return
		}

		var user models.User
		res = db.First(&user, "id = ?", apiKeyRecord.UserID)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			// The user does not exist
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "User not found")
			return
		}

		// If the API key is valid and the user exists, add the user to the request context
		ctx := context.WithValue(r.Context(), ctxUserKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
