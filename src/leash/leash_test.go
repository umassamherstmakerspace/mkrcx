package main_test

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	leash_helpers "github.com/mkrcx/mkrcx/src/leash/helpers"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"github.com/valyala/fasthttp"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	ROLE_MEMBER = iota
	ROLE_VOLUNTEER
	ROLE_STAFF
	ROLE_ADMIN
)

var _ leash_auth.ExternalAuthenticator = (*DebugExternalAuth)(nil)

type DebugExternalAuth struct{}

func (d *DebugExternalAuth) GetAuthURL(state string) string {
	return "http://localhost:3000/auth/callback?code=admin@example.com&state=" + state
}

func (d *DebugExternalAuth) Callback(ctx context.Context, code string) (string, error) {
	return code, nil
}

type TestUser struct {
	User   models.User
	APIKey models.APIKey
}

type EndpointTester struct {
	t         *testing.T
	roleUsers []TestUser
	starUser  TestUser
	testUser  TestUser
	db        *gorm.DB
	enforcer  *leash_auth.EnforcerWrapper
}

func (e *EndpointTester) testEndpoint(endpoint string, method string, body []byte, auth string, expectedStatus int) (int, []byte) {
	// Test endpoint
	link := "http://localhost:3000" + endpoint
	agent := fiber.AcquireAgent()

	req := agent.Request()
	req.Header.SetMethod(method)
	req.SetRequestURI(link)
	req.Header.Set("Authorization", auth)
	req.Header.SetContentType(fiber.MIMEApplicationJSON)

	if body != nil {
		req.SetBody(body)
	} else {
		req.SetBody([]byte("{}"))
	}

	if err := agent.Parse(); err != nil {
		e.t.Fatal(err)
	}

	status, b, err := agent.Bytes()
	if err != nil {
		e.t.Fatal(err)
	}

	if status != expectedStatus {
		e.t.Fatalf("Expected status %d, got %d\n Response: %s", expectedStatus, status, string(b))
	}

	return status, b
}

type PreFunc func(user models.User)
type PostFunc func(user models.User, b []byte, status int)

func (e *EndpointTester) TestPermissions(endpoint string, method string, body []byte, permissions []string, expectedStatus int, preFunc PreFunc, postFunc PostFunc) {
	e.t.Log("Permission test for: ", endpoint)
	tmpUser := e.testUser.User

	maxPermission := int(math.Pow(2, float64(len(permissions))))
	for i := 0; i < maxPermission; i++ {
		testPermissions := []string{}
		for j := 0; j < len(permissions); j++ {
			if i&(1<<uint(j)) != 0 {
				testPermissions = append(testPermissions, permissions[j])
			}
		}

		e.enforcer.SetPermissionsForUser(e.testUser.User, testPermissions)
		err := e.enforcer.SavePolicy()
		if err != nil {
			e.t.Fatal(err)
		}

		status := fiber.StatusUnauthorized

		if len(testPermissions) == len(permissions) {
			status = expectedStatus
		}

		e.t.Log("Testing permissions: ", testPermissions)

		if preFunc != nil {
			preFunc(e.testUser.User)
		}

		status, b := e.testEndpoint(endpoint, method, body, "API-Key "+e.testUser.APIKey.Key, status)

		if postFunc != nil {
			postFunc(e.testUser.User, b, status)
		}
	}

	e.testUser.User = tmpUser
	e.db.Save(&e.testUser.User)
	e.enforcer.SetPermissionsForUser(e.testUser.User, e.testUser.User.Permissions)
}

func (e *EndpointTester) TestRoles(endpoint string, method string, body []byte, minimumRole int, expectedStatus int, preFunc PreFunc, postFunc PostFunc) {
	e.t.Log("Role test for: ", endpoint)

	for i := 0; i < len(e.roleUsers); i++ {
		testUser := e.roleUsers[i].User
		testAPIKey := e.roleUsers[i].APIKey

		tmpUser := e.roleUsers[i].User

		e.t.Log("Testing role: ", testUser.Role)

		status := fiber.StatusUnauthorized

		if i >= minimumRole {
			status = expectedStatus
		}

		if preFunc != nil {
			preFunc(testUser)
		}

		status, b := e.testEndpoint(endpoint, method, body, "API-Key "+testAPIKey.Key, status)

		if postFunc != nil {
			postFunc(testUser, b, status)
		}

		e.roleUsers[i].User = tmpUser
		e.db.Save(&e.roleUsers[i].User)
	}
}

type BodyTester func([]byte)

func (e *EndpointTester) TestResponse(endpoint string, method string, body []byte, bodyTester BodyTester, expectedStatus int, preFunc PreFunc, postFunc PostFunc) {
	e.t.Log("Response test for: ", endpoint)

	tmpUser := e.starUser.User

	if preFunc != nil {
		preFunc(e.starUser.User)
	}

	status, b := e.testEndpoint(endpoint, method, body, "API-Key "+e.starUser.APIKey.Key, expectedStatus)

	if postFunc != nil {
		postFunc(e.starUser.User, b, status)
	}

	e.starUser.User = tmpUser
	e.db.Save(&e.starUser.User)
}

func (e *EndpointTester) TestAll(testName string, endpoint string, method string, body []byte, permissions []string, minimumRole int, bodyTester BodyTester, expectedStatus int, preFunc PreFunc, postFunc PostFunc) {
	e.t.Logf("Running test: %s\n", testName)

	e.TestPermissions(endpoint, method, body, permissions, expectedStatus, preFunc, postFunc)
	e.TestRoles(endpoint, method, body, minimumRole, expectedStatus, preFunc, postFunc)
	e.TestResponse(endpoint, method, body, bodyTester, expectedStatus, preFunc, postFunc)
}

func setupEndpointTester(t *testing.T, db *gorm.DB, enforcer *casbin.Enforcer) (*EndpointTester, error) {
	enforcerWrapper := leash_auth.EnforcerWrapper{
		Enforcer: enforcer,
	}

	endpointTester := EndpointTester{
		t:         t,
		roleUsers: []TestUser{},
		starUser:  TestUser{},
		testUser:  TestUser{},
		db:        db,
		enforcer:  &enforcerWrapper,
	}

	endpointTester.starUser.User = models.User{
		Name:        "Star User",
		Email:       "star@example.com",
		Role:        "admin",
		Type:        "other",
		Permissions: []string{},
	}

	db.Create(&endpointTester.starUser.User)

	endpointTester.starUser.APIKey = models.APIKey{
		Key:         "star",
		UserID:      endpointTester.starUser.User.ID,
		FullAccess:  true,
		Permissions: []string{},
	}

	db.Create(&endpointTester.starUser.APIKey)

	endpointTester.testUser.User = models.User{
		Name:        "Test User",
		Email:       "test@example.com",
		Role:        "test",
		Type:        "other",
		Permissions: []string{},
	}

	db.Create(&endpointTester.testUser.User)

	endpointTester.testUser.APIKey = models.APIKey{
		Key:         "test",
		UserID:      endpointTester.testUser.User.ID,
		FullAccess:  true,
		Permissions: []string{},
	}

	db.Create(&endpointTester.testUser.APIKey)

	roles := []string{"member", "volunteer", "staff", "admin"}
	for i := 0; i < len(roles); i++ {
		roleUser := TestUser{}
		roleUser.User = models.User{
			Name:        roles[i] + " User",
			Email:       roles[i] + "@example.com",
			Role:        roles[i],
			Type:        "other",
			Permissions: []string{},
		}

		db.Create(&roleUser.User)

		roleUser.APIKey = models.APIKey{
			Key:         roles[i],
			UserID:      roleUser.User.ID,
			FullAccess:  true,
			Permissions: []string{},
		}

		db.Create(&roleUser.APIKey)

		endpointTester.roleUsers = append(endpointTester.roleUsers, roleUser)
	}

	return &endpointTester, nil
}

func TestLeash(t *testing.T) {
	// Initialize DB
	t.Log("Initializing DB...")
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	// db, err := gorm.Open(sqlite.Open("sqlite.db"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Migrating database schema...")
	err = leash_helpers.MigrateSchema(db)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Migrating database schema...")
	externalAuth := &DebugExternalAuth{}

	// JWT Key
	t.Log("Initializing JWT Keys...")
	set, err := leash_auth.GenerateJWTKeySet()
	if err != nil {
		t.Fatal(err)
	}

	keys, err := leash_auth.CreateKeys(set)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize RBAC
	t.Log("Initializing RBAC...")
	enforcer, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		t.Fatal(err)
	}

	leash_helpers.SetupCasbin(enforcer)

	// Setup testing DB
	t.Log("Setting up testing DB...")
	endpointTester, err := setupEndpointTester(t, db, enforcer)
	if err != nil {
		t.Fatal(err)
	}

	// Create App
	t.Log("Initializing Fiber...")

	app := fiber.New()

	t.Log("Setting up middleware...")
	leash_helpers.SetupMiddlewares(app, db, keys, externalAuth, enforcer)

	t.Log("Setting up routes...")
	leash_helpers.SetupRoutes(app)

	ready := make(chan bool)

	app.Hooks().OnListen(func() error {
		ready <- true
		return nil
	})

	t.Log("Starting server on port :3000")
	go app.Listen(":3000")

	<-ready

	encode := func(v interface{}) []byte {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}

		return b
	}

	statusEQ := func(statusCode int) BodyTester {
		status := fasthttp.StatusMessage(statusCode)
		return func(b []byte) {
			if status != string(b) {
				t.Fatalf("Expected %v, got %v", statusCode, string(b))
			}
		}
	}

	byteEQ := func(b1 []byte) BodyTester {
		return func(b2 []byte) {
			if string(b1) != string(b2) {
				t.Fatalf("Expected %v, got %v", string(b1), string(b2))
			}
		}
	}

	_ = byteEQ

	userEQ := func(user models.User) BodyTester {
		u1 := user
		return func(b []byte) {
			var u2 models.User
			err := json.Unmarshal(b, &u2)
			if err != nil {
				t.Fatal(err)
			}

			u1.UpdatedAt = u2.UpdatedAt
			u1.CreatedAt = u2.CreatedAt
			u1.ID = u2.ID
			u := encode(u1)

			if string(u) != string(b) {
				t.Fatalf("Expected %v, got %v", string(u), string(b))
			}
		}
	}

	trainingEQ := func(training models.Training) BodyTester {
		t1 := training
		return func(b []byte) {
			var t2 models.Training
			err := json.Unmarshal(b, &t2)
			if err != nil {
				t.Fatal(err)
			}

			t1.UpdatedAt = t2.UpdatedAt
			t1.CreatedAt = t2.CreatedAt
			t1.ID = t2.ID
			tr := encode(t1)

			if string(tr) != string(b) {
				t.Fatalf("Expected %v, got %v", string(tr), string(b))
			}
		}
	}

	holdEQ := func(hold models.Hold) BodyTester {
		h1 := hold
		return func(b []byte) {
			var h2 models.Hold
			err := json.Unmarshal(b, &h2)
			if err != nil {
				t.Fatal(err)
			}

			h1.UpdatedAt = h2.UpdatedAt
			h1.CreatedAt = h2.CreatedAt
			h1.ID = h2.ID
			h := encode(h1)

			if string(h) != string(b) {
				t.Fatalf("Expected %v, got %v", string(h), string(b))
			}
		}
	}

	apikeyEQ := func(apikey models.APIKey) BodyTester {
		a1 := apikey
		return func(b []byte) {
			var a2 models.APIKey
			err := json.Unmarshal(b, &a2)
			if err != nil {
				t.Fatal(err)
			}

			a1.UpdatedAt = a2.UpdatedAt
			a1.CreatedAt = a2.CreatedAt
			a := encode(a1)

			if string(a) != string(b) {
				t.Fatalf("Expected %v, got %v", string(a), string(b))
			}
		}
	}

	notificationEQ := func(notification models.Notification) BodyTester {
		n1 := notification
		return func(b []byte) {
			var n2 models.Notification
			err := json.Unmarshal(b, &n2)
			if err != nil {
				t.Fatal(err)
			}

			n1.UpdatedAt = n2.UpdatedAt
			n1.CreatedAt = n2.CreatedAt
			n1.ID = n2.ID
			n := encode(n1)

			if string(n) != string(b) {
				t.Fatalf("Expected %v, got %v", string(n), string(b))
			}
		}
	}

	listCountEQ := func(count int) BodyTester {
		return func(b []byte) {
			var list struct {
				Data  []interface{} `json:"data"`
				Total int           `json:"total"`
			}

			err := json.Unmarshal(b, &list)
			if err != nil {
				t.Fatal(err)
			}

			if len(list.Data) != count {
				t.Fatalf("Expected %d, got %d", count, len(list.Data))
			}
		}
	}

	var newUser models.User
	var newTraining models.Training
	var newHold models.Hold
	var newAPIKey models.APIKey
	var newNotification models.Notification
	t.Log("Testing Self Endpoints...")

	endpointTester.TestAll("Get Self",
		"/api/users/self", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK,
		nil,
		nil)

	endpointTester.TestAll("Get Self With Trainings",
		"/api/users/self?with_trainings=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.trainings:list"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK,
		nil,
		nil)

	endpointTester.TestAll("Get Self With Holds",
		"/api/users/self?with_holds=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.holds:list"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK,
		nil,
		nil)

	newUser = endpointTester.starUser.User
	newUser.APIKeys = append(newUser.APIKeys, endpointTester.starUser.APIKey)
	endpointTester.TestAll("Get Self With Api Keys",
		"/api/users/self?with_api_keys=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.apikeys:list"}, ROLE_MEMBER,
		userEQ(newUser),
		fiber.StatusOK,
		nil,
		nil)

	endpointTester.TestAll("Get Self With Updates",
		"/api/users/self?with_updates=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.updates:list"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK,
		nil,
		nil)

	endpointTester.TestAll("Get Self With Notifications",
		"/api/users/self?with_notifications=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.notifications:list"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK,
		nil,
		nil)

	endpointTester.TestAll("Get Self Updates Before Updates",
		"/api/users/self/updates", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.updates:list"}, ROLE_MEMBER,
		listCountEQ(0),
		fiber.StatusOK,
		nil,
		nil)

	endpointTester.TestAll("Update Self Empty",
		"/api/users/self", "PATCH",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:update"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK,
		nil,
		nil)

	newUser = endpointTester.starUser.User
	newUser.Name = "New Name"
	endpointTester.TestAll("Update Self Normal",
		"/api/users/self", "PATCH",
		[]byte("{\"name\":\"New Name\"}"),
		[]string{"leash.users:target_self", "leash.users.self:update"}, ROLE_MEMBER,
		userEQ(newUser),
		fiber.StatusOK,
		nil,
		nil)

	newUser = endpointTester.starUser.User
	newUser.Role = "member"
	endpointTester.TestAll("Update Self Role",
		"/api/users/self", "PATCH",
		[]byte("{\"role\":\"member\"}"),
		[]string{"leash.users:target_self", "leash.users.self:update", "leash.users.self:update_role"}, ROLE_ADMIN,
		userEQ(newUser),
		fiber.StatusOK,
		nil,
		nil)

	newUser = endpointTester.starUser.User
	card_id := "1234567890"
	newUser.CardID = &card_id
	endpointTester.TestAll("Update Self Card ID",
		"/api/users/self", "PATCH",
		[]byte("{\"card_id\":\"1234567890\"}"),
		[]string{"leash.users:target_self", "leash.users.self:update", "leash.users.self:update_card_id"}, ROLE_ADMIN,
		userEQ(newUser),
		fiber.StatusOK,
		nil,
		nil)

	endpointTester.TestAll("Get Self Updates After Updates",
		"/api/users/self/updates", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.updates:list"}, ROLE_MEMBER,
		listCountEQ(3),
		fiber.StatusOK,
		nil,
		nil)

	endpointTester.TestAll("Get Self Trainings Empty",
		"/api/users/self/trainings", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.trainings:list"}, ROLE_MEMBER,
		listCountEQ(0),
		fiber.StatusOK,
		nil,
		nil)

	newTraining = models.Training{
		TrainingType: "other",
		UserID:       endpointTester.starUser.User.ID,
		AddedBy:      endpointTester.starUser.User.ID,
	}

	endpointTester.TestAll("Create Self Training",
		"/api/users/self/trainings", "POST",
		[]byte("{\"training_type\":\"other\"}"),
		[]string{"leash.users:target_self", "leash.users.self.trainings:create"}, ROLE_VOLUNTEER,
		trainingEQ(newTraining),
		fiber.StatusOK,
		nil,
		func(user models.User, b []byte, status int) {
			if status == fiber.StatusOK {
				var training models.Training
				err := json.Unmarshal(b, &training)
				if err != nil {
					t.Fatal(err)
				}

				db.Unscoped().Delete(&training)
			}
		})

	endpointTester.TestAll("Get Self Trainings Not Empty",
		"/api/users/self/trainings", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.trainings:list"}, ROLE_MEMBER,
		listCountEQ(1),
		fiber.StatusOK,
		func(user models.User) {
			training := newTraining
			training.UserID = user.ID
			db.Create(&training)
		},
		func(user models.User, b []byte, status int) {
			db.Unscoped().Delete(&models.Training{}, newTraining.ID)
		})

	db.Create(&newHold)
	db.Unscoped().Delete(&newHold)

	endpointTester.TestAll("Get Self Single Training",
		"/api/users/self/trainings/other", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.trainings:get"}, ROLE_MEMBER,
		trainingEQ(newTraining),
		fiber.StatusOK,
		func(user models.User) {
			training := newTraining
			training.UserID = user.ID
			db.Create(&training)
		},
		func(user models.User, b []byte, status int) {
			db.Unscoped().Delete(&models.Training{}, newTraining.ID)
		})

	endpointTester.TestAll("Delete Self Training",
		"/api/users/self/trainings/other", "DELETE",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.trainings:delete"}, ROLE_VOLUNTEER,
		statusEQ(fiber.StatusNoContent),
		fiber.StatusNoContent,
		func(user models.User) {
			training := newTraining
			training.UserID = user.ID
			db.Create(&training)
		},
		func(user models.User, b []byte, status int) {
			db.Unscoped().Delete(&models.Training{}, newTraining.ID)
		})

	endpointTester.TestAll("Get Self Holds Empty",
		"/api/users/self/holds", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.holds:list"}, ROLE_MEMBER,
		listCountEQ(0),
		fiber.StatusOK,
		nil,
		nil)

	newHold = models.Hold{
		Reason:    "Test Hold",
		HoldType:  "other",
		UserID:    endpointTester.starUser.User.ID,
		AddedBy:   endpointTester.starUser.User.ID,
		Priority:  10,
		HoldStart: nil,
		HoldEnd:   nil,
	}

	endpointTester.TestAll("Create Self Hold",
		"/api/users/self/holds", "POST",
		[]byte("{\"reason\":\"Test Hold\",\"hold_type\":\"other\",\"priority\":10}"),
		[]string{"leash.users:target_self", "leash.users.self.holds:create"}, ROLE_VOLUNTEER,
		holdEQ(newHold),
		fiber.StatusOK,
		nil,
		func(user models.User, b []byte, status int) {
			if status == fiber.StatusOK {
				var hold models.Hold
				err := json.Unmarshal(b, &hold)
				if err != nil {
					t.Fatal(err)
				}

				db.Unscoped().Delete(&hold)
			}
		})

	endpointTester.TestAll("Get Self Holds Not Empty",
		"/api/users/self/holds", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.holds:list"}, ROLE_MEMBER,
		listCountEQ(1),
		fiber.StatusOK,
		func(user models.User) {
			hold := newHold
			hold.UserID = user.ID
			db.Create(&hold)
		},
		func(user models.User, b []byte, status int) {
			db.Unscoped().Delete(&models.Hold{}, newHold.ID)
		})

	db.Create(&newHold)
	db.Unscoped().Delete(&newHold)

	endpointTester.TestAll("Get Self Single Holds",
		"/api/users/self/holds/other", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.holds:get"}, ROLE_MEMBER,
		holdEQ(newHold),
		fiber.StatusOK,
		func(user models.User) {
			hold := newHold
			hold.UserID = user.ID
			db.Create(&hold)
		},
		func(user models.User, b []byte, status int) {
			db.Unscoped().Delete(&models.Hold{}, newHold.ID)
		})

	endpointTester.TestAll("Delete Self Hold",
		"/api/users/self/holds/other", "DELETE",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.holds:delete"}, ROLE_VOLUNTEER,
		statusEQ(fiber.StatusNoContent),
		fiber.StatusNoContent,
		func(user models.User) {
			hold := newHold
			hold.UserID = user.ID
			db.Create(&hold)
		},
		func(user models.User, b []byte, status int) {
			db.Unscoped().Delete(&models.Hold{}, newHold.ID)
		})

	endpointTester.TestAll("Get Self Api Keys Initial (1)",
		"/api/users/self/apikeys", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.apikeys:list"}, ROLE_MEMBER,
		listCountEQ(1),
		fiber.StatusOK,
		nil,
		nil)

	newAPIKey = models.APIKey{
		Key:         "newkey",
		UserID:      endpointTester.starUser.User.ID,
		Description: "New Key",
		FullAccess:  false,
		Permissions: []string{},
	}

	endpointTester.TestAll("Get Specific Self Api Key",
		"/api/users/self/apikeys/newkey", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.apikeys:get"}, ROLE_MEMBER,
		apikeyEQ(newAPIKey),
		fiber.StatusOK,
		func(user models.User) {
			apiKey := newAPIKey
			apiKey.UserID = user.ID
			db.Create(&apiKey)
		},
		func(user models.User, b []byte, status int) {
			apiKey := models.APIKey{}
			apiKey.Key = newAPIKey.Key
			db.Unscoped().Delete(&apiKey)
		})

	endpointTester.TestAll("Create Self Api Key",
		"/api/users/self/apikeys", "POST",
		[]byte("{\"full_access\":false,\"permissions\":[],\"description\":\"New Key\"}"),
		[]string{"leash.users:target_self", "leash.users.self.apikeys:create"}, ROLE_MEMBER,
		apikeyEQ(newAPIKey),
		fiber.StatusOK,
		nil,
		func(user models.User, b []byte, status int) {
			if status == fiber.StatusOK {
				var apiKey models.APIKey
				err := json.Unmarshal(b, &apiKey)
				if err != nil {
					t.Fatal(err)
				}

				db.Unscoped().Delete(&apiKey)
			}
		})

	_ = newNotification
	_ = notificationEQ
}
