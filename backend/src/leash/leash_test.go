package main_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
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

const MAX_ROLE = ROLE_ADMIN

func roleFromNumber(roleNum int) string {
	switch roleNum {
	case ROLE_MEMBER:
		return "member"
	case ROLE_VOLUNTEER:
		return "volunteer"
	case ROLE_STAFF:
		return "staff"
	case ROLE_ADMIN:
		return "admin"
	default:
		return "INVALID"
	}
}

var _ leash_auth.ExternalAuthenticator = (*DebugExternalAuth)(nil)

type DebugExternalAuth struct{}

func (d *DebugExternalAuth) GetAuthURL(state string) string {
	return "http://localhost:3000/auth/callback?code=admin@example.com&state=" + state
}

func (d *DebugExternalAuth) Callback(ctx context.Context, code string) (string, error) {
	return code, nil
}

func purgeUser(db *gorm.DB, user models.User) {
	db.Unscoped().Delete(&models.UserUpdate{}, &models.UserUpdate{UserID: user.ID})
	db.Unscoped().Delete(&models.Training{}, &models.Training{UserID: user.ID})
	db.Unscoped().Delete(&models.Hold{}, &models.Hold{UserID: user.ID})
	db.Unscoped().Delete(&models.APIKey{}, &models.APIKey{UserID: user.ID})
	db.Unscoped().Delete(&models.Notification{}, &models.Notification{UserID: user.ID})
}

type TestUser struct {
	User   models.User
	APIKey models.APIKey
}

type QueryArgs map[string]string

type EndpointTester struct {
	t           *testing.T
	db          *gorm.DB
	enforcer    *leash_auth.EnforcerWrapper
	URL         string
	Method      string
	testPrefix  string
	TestName    string
	Body        []byte
	SetupUser   func(testPrefix string, user models.User) error
	CleanUpUser func(testPrefix string, user models.User) error
}

func (e *EndpointTester) testingID() string {
	data := e.URL + e.Method + string(e.Body) + e.TestName + e.testPrefix
	return uuid.NewSHA1(uuid.Nil, []byte(data)).String()
}

func (e *EndpointTester) TestEndpoint(t *testing.T, auth string) (int, []byte) {
	// Test endpoint
	agent := fiber.AcquireAgent()

	req := agent.Request()
	req.Header.SetMethod(e.Method)
	req.SetRequestURI(e.URL)
	req.Header.Set("Authorization", auth)
	req.Header.SetContentType(fiber.MIMEApplicationJSON)

	if e.Body != nil {
		req.SetBody(e.Body)
	} else {
		req.SetBody([]byte("{}"))
	}

	if err := agent.Parse(); err != nil {
		t.Fatal(err)
	}

	status, b, err := agent.Bytes()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Got status code: ", status)
	t.Log("Got body: ", string(b))

	return status, b
}

func (e *EndpointTester) RequiresPermissions(permissions []string) *EndpointTester {
	e.t.Run("Permission Test", func(t *testing.T) {
		t.Log("Running Permission Test for: ", e.TestName)
		t.Log("Testing permissions: ", permissions)

		userUUID := e.testingID() + ".permissions"

		oldUser := models.User{
			Email: userUUID + "@mkr.cx",
		}

		if res := e.db.Limit(1).Where(&oldUser).Find(&oldUser); res.Error != nil || res.RowsAffected > 0 {
			purgeUser(e.db, oldUser)
			e.db.Unscoped().Delete(&models.User{}, &models.User{ID: oldUser.ID})
		}

		user := models.User{
			Name:  "Permission Test User",
			Email: userUUID + "@mkr.cx",
			Role:  "test",
			Type:  "other",
		}

		e.db.Create(&user)
		apiKey := models.APIKey{
			Key:         userUUID,
			UserID:      user.ID,
			FullAccess:  true,
			Permissions: []string{},
		}
		e.db.Create(&apiKey)

		if e.SetupUser != nil {
			err := e.SetupUser(e.testingID(), user)
			if err != nil {
				t.Fatal(err)
			}
		}

		maxPermission := int(math.Pow(2, float64(len(permissions))))
		for i := 0; i < maxPermission; i++ {
			testPermissions := []string{}
			for j := 0; j < len(permissions); j++ {
				if i&(1<<uint(j)) != 0 {
					testPermissions = append(testPermissions, permissions[j])
				}
			}

			e.enforcer.SetPermissionsForUser(user, testPermissions)
			err := e.enforcer.SavePolicy()
			if err != nil {
				e.t.Fatal(err)
			}

			e.t.Log("Testing permissions: ", testPermissions)

			status, _ := e.TestEndpoint(t, "API-Key "+apiKey.Key)

			if len(testPermissions) == len(permissions) {
				if status == fiber.StatusUnauthorized {
					t.Fatalf("Endpoint was not authorized with all permissions: %v", testPermissions)
				}
			} else {
				if status != fiber.StatusUnauthorized {
					t.Fatalf("Endpoint was authorized with only permissions: %v, code: %v", testPermissions, status)
				}
			}

		}

		e.db.Save(&user)
		e.enforcer.SetPermissionsForUser(user, []string{})

		if e.CleanUpUser != nil {
			err := e.CleanUpUser(e.testingID(), user)
			if err != nil {
				t.Fatal(err)
			}
		}
	})

	return e
}

func (e *EndpointTester) MinimumRole(minimumRole int) *EndpointTester {
	e.t.Run("Role Test", func(t *testing.T) {
		t.Log("Running Role Test for: ", e.TestName)
		t.Log("Testing minimum role: ", roleFromNumber(minimumRole))

		for i := 0; i <= MAX_ROLE; i++ {
			userUUID := e.testingID() + ".role." + roleFromNumber(i)

			oldUser := models.User{
				Email: userUUID + "@mkr.cx",
			}

			if res := e.db.Limit(1).Where(&oldUser).Find(&oldUser); res.Error != nil || res.RowsAffected > 0 {
				purgeUser(e.db, oldUser)
				e.db.Unscoped().Delete(&models.User{}, &models.User{ID: oldUser.ID})
			}

			user := models.User{
				Name:  "Role Test User: " + roleFromNumber(i),
				Email: userUUID + "@mkr.cx",
				Role:  roleFromNumber(i),
				Type:  "other",
			}

			e.db.Create(&user)

			apiKey := models.APIKey{
				Key:         userUUID,
				UserID:      user.ID,
				FullAccess:  true,
				Permissions: []string{},
			}
			e.db.Create(&apiKey)

			if e.SetupUser != nil {
				err := e.SetupUser(e.testingID(), user)
				if err != nil {
					t.Fatal(err)
				}
			}

			e.t.Log("Testing role: ", roleFromNumber(i))

			status, _ := e.TestEndpoint(t, "API-Key "+apiKey.Key)

			if i >= minimumRole {
				if status == fiber.StatusUnauthorized {
					t.Fatalf("Endpoint was not authorized with role: %v, when it should have been", roleFromNumber(i))
				}
			} else {
				if status != fiber.StatusUnauthorized {
					t.Fatalf("Endpoint was authorized with role: %v, when it should not have been", roleFromNumber(i))
				}
			}

			e.db.Save(&user)
			e.enforcer.SetPermissionsForUser(user, []string{})

			if e.CleanUpUser != nil {
				err := e.CleanUpUser(e.testingID(), user)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	})

	return e
}

type ResponseTester struct {
	Name string
	Test func(*testing.T, string, int, []byte)
}

func (e *EndpointTester) GivesResponse(responseTesters ...ResponseTester) *EndpointTester {
	e.t.Run("Response Test", func(t *testing.T) {
		t.Log("Running Response Test for: ", e.TestName)

		testerNames := ""
		for i, tester := range responseTesters {
			if i != 0 {
				testerNames = testerNames + ", "
			}
			testerNames = testerNames + tester.Name
		}

		t.Logf("Testing responses: [%v]", testerNames)

		userUUID := e.testingID() + ".response"

		oldUser := models.User{
			Email: userUUID + "@mkr.cx",
		}

		if res := e.db.Limit(1).Where(&oldUser).Find(&oldUser); res.Error != nil || res.RowsAffected > 0 {
			purgeUser(e.db, oldUser)
			e.db.Unscoped().Delete(&models.User{}, &models.User{ID: oldUser.ID})
		}

		user := models.User{
			Name:  "Response Test User",
			Email: userUUID + "@mkr.cx",
			Role:  "admin",
			Type:  "other",
		}

		e.db.Create(&user)

		apiKey := models.APIKey{
			Key:         userUUID,
			UserID:      user.ID,
			FullAccess:  true,
			Permissions: []string{},
		}

		e.db.Create(&apiKey)

		if e.SetupUser != nil {
			err := e.SetupUser(e.testingID(), user)
			if err != nil {
				t.Fatal(err)
			}
		}

		status, b := e.TestEndpoint(t, "API-Key "+apiKey.Key)

		for _, tester := range responseTesters {
			tester.Test(t, e.testingID(), status, b)
		}

		e.db.Save(&user)
		e.enforcer.SetPermissionsForUser(user, []string{})

		if e.CleanUpUser != nil {
			err := e.CleanUpUser(e.testingID(), user)
			if err != nil {
				t.Fatal(err)
			}
		}
	})

	return e
}

func (e *EndpointTester) GivesResponseNoAuth(responseTesters ...ResponseTester) *EndpointTester {
	e.t.Run("Response Test", func(t *testing.T) {
		t.Log("Running Response Test Without Authentication for: ", e.TestName)

		testerNames := ""
		for i, tester := range responseTesters {
			if i != 0 {
				testerNames = testerNames + ", "
			}
			testerNames = testerNames + tester.Name
		}

		t.Logf("Testing responses: [%v]", testerNames)

		status, b := e.TestEndpoint(t, "")

		for _, tester := range responseTesters {
			tester.Test(t, e.testingID(), status, b)
		}
	})

	return e
}

type EndpointBuilder struct {
	t           *testing.T
	db          *gorm.DB
	enforcer    *leash_auth.EnforcerWrapper
	testPrefix  string
	endpoint    string
	method      string
	query       QueryArgs
	body        []byte
	userSetup   func(testPrefix string, user models.User) error
	userCleanup func(testPrefix string, user models.User) error
}

func (b *EndpointBuilder) Test(testName string, endpointTesterFunction func(t *EndpointTester)) {
	b.t.Run(testName, func(t *testing.T) {
		t.Log("Running Test: ", testName)
		t.Log("Testing Endpoint: ", b.endpoint)
		t.Log("Testing Method: ", b.method)
		t.Log("Testing Query: ", b.query)
		t.Log("Testing Body: ", string(b.body))

		url := url.URL{
			Scheme: "http",
			Host:   "localhost:3000",
			Path:   b.endpoint,
		}

		q := url.Query()

		for k, v := range b.query {
			q.Set(k, v)
		}

		url.RawQuery = q.Encode()

		t.Log("Testing URL: ", url.String())

		tester := &EndpointTester{
			t:           t,
			db:          b.db,
			enforcer:    b.enforcer,
			URL:         url.String(),
			Method:      b.method,
			testPrefix:  b.testPrefix,
			TestName:    testName,
			Body:        b.body,
			SetupUser:   b.userSetup,
			CleanUpUser: b.userCleanup,
		}

		endpointTesterFunction(tester)
	})
}

func (b *EndpointBuilder) WithQuery(query QueryArgs) *EndpointBuilder {
	b.query = query
	return b
}

func (b *EndpointBuilder) WithBody(body []byte) *EndpointBuilder {
	b.body = body
	return b
}

func (b *EndpointBuilder) SetupUser(userSetup func(testPrefix string, user models.User) error) *EndpointBuilder {
	b.userSetup = userSetup
	return b
}

func (b *EndpointBuilder) CleanupUser(userCleanup func(testPrefix string, user models.User) error) *EndpointBuilder {
	b.userCleanup = userCleanup
	return b
}

type Tester struct {
	t          *testing.T
	db         *gorm.DB
	enforcer   *leash_auth.EnforcerWrapper
	testPrefix string
}

func (test *Tester) Test(testName string, testFunction func(test *Tester)) *Tester {
	test.t.Run(testName, func(t *testing.T) {
		t.Log("Running Test: ", testName)
		subTest := &Tester{
			t:          t,
			db:         test.db,
			enforcer:   test.enforcer,
			testPrefix: test.testPrefix + "." + testName,
		}

		testFunction(subTest)
	})

	return test
}

func (test *Tester) Endpoint(endpoint string, method string) *EndpointBuilder {
	return &EndpointBuilder{
		t:          test.t,
		db:         test.db,
		enforcer:   test.enforcer,
		endpoint:   endpoint,
		method:     method,
		testPrefix: test.testPrefix,
	}
}

func setupTester(t *testing.T, db *gorm.DB, enforcer *casbin.Enforcer) *Tester {

	enforcerWrapper := leash_auth.EnforcerWrapper{
		Enforcer: enforcer,
	}

	return &Tester{
		t:          t,
		db:         db,
		enforcer:   &enforcerWrapper,
		testPrefix: "",
	}
}

func TestLeash(t *testing.T) {
	// Initialize DB
	t.Log("Initializing DB...")
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Migrating database schema...")
	err = leash_helpers.MigrateSchema(db)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Setting up auth...")
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

	hmacKey := make([]byte, 64)
	rand.Read(hmacKey)

	// Initialize RBAC
	t.Log("Initializing RBAC...")
	enforcer, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		t.Fatal(err)
	}

	leash_helpers.SetupCasbin(enforcer)

	// Setup tester
	t.Log("Setting tester...")
	tester := setupTester(t, db, enforcer)

	// Create App
	t.Log("Initializing Fiber...")

	app := fiber.New()

	t.Log("Setting up middleware...")
	leash_helpers.SetupMiddlewares(app, db, keys, hmacKey, externalAuth, enforcer)

	t.Log("Setting up routes...")
	leash_helpers.SetupRoutes(app)

	ready := make(chan bool)

	app.Hooks().OnListen(func(_ fiber.ListenData) error {
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

	statusCode := func(statusCode int) ResponseTester {
		return ResponseTester{
			Name: fmt.Sprintf("Status Code %d", statusCode),
			Test: func(t *testing.T, _ string, status int, _ []byte) {
				if statusCode != status {
					t.Fatalf("Expected status code %d, got %d", statusCode, status)
				} else {
					t.Logf("Got status code %d", statusCode)
				}
			},
		}
	}

	defaultStatusResponse := ResponseTester{
		Name: "FastHTTP Default Status Message",
		Test: func(t *testing.T, _ string, status int, b []byte) {
			message := fasthttp.StatusMessage(status)
			if message != string(b) {
				t.Fatalf("Expected %v, got %v", message, string(b))
			} else {
				t.Logf("Got status message %v", message)
			}
		},
	}

	listLengthEQ := func(length int) ResponseTester {
		return ResponseTester{
			Name: fmt.Sprintf("List Length %d", length),
			Test: func(t *testing.T, _ string, _ int, b []byte) {
				var list struct {
					Data  []interface{} `json:"data"`
					Total int           `json:"total"`
				}

				err := json.Unmarshal(b, &list)
				if err != nil {
					t.Fatal(err)
				}

				if len(list.Data) != length {
					t.Fatalf("Expected %d, got %d", length, len(list.Data))
				} else {
					t.Logf("Got list length %d", length)
				}
			},
		}
	}

	userEQ := func(user models.User) ResponseTester {
		testUser := user
		return ResponseTester{
			Name: "User Response Tester",
			Test: func(t *testing.T, testPrefix string, _ int, b []byte) {
				var responseUser models.User
				err := json.Unmarshal(b, &responseUser)
				if err != nil {
					t.Fatal(err)
				}

				if testUser.Permissions == nil {
					testUser.Permissions = []string{}
				}

				testUser.UpdatedAt = responseUser.UpdatedAt
				testUser.CreatedAt = responseUser.CreatedAt
				testUser.ID = responseUser.ID

				if !strings.HasSuffix(responseUser.Email, testUser.Email) {
					t.Fatalf("Expected email to be like: %v, got %v", testUser.Email, responseUser.Email)
				}

				testUser.Email = responseUser.Email
				expected := string(encode(testUser))

				if expected != string(b) {
					t.Fatalf("Expected %v, got %v", expected, string(b))
				} else {
					t.Logf("Got user %v", expected)
				}
			},
		}
	}

	trainingEQ := func(training models.Training) ResponseTester {
		testTraining := training
		return ResponseTester{
			Name: "Training Response Tester",
			Test: func(t *testing.T, testPrefix string, _ int, b []byte) {
				var responseTraining models.Training
				err := json.Unmarshal(b, &responseTraining)
				if err != nil {
					t.Fatal(err)
				}

				testTraining.UpdatedAt = responseTraining.UpdatedAt
				testTraining.CreatedAt = responseTraining.CreatedAt
				testTraining.ID = responseTraining.ID
				testTraining.UserID = responseTraining.UserID
				testTraining.AddedBy = responseTraining.AddedBy
				expected := string(encode(testTraining))

				if expected != string(b) {
					t.Fatalf("Expected %v, got %v", expected, string(b))
				} else {
					t.Logf("Got training %v", expected)
				}
			},
		}
	}

	holdEQ := func(hold models.Hold) ResponseTester {
		testHold := hold
		return ResponseTester{
			Name: "Hold Response Tester",
			Test: func(t *testing.T, testPrefix string, _ int, b []byte) {
				var responseHold models.Hold
				err := json.Unmarshal(b, &responseHold)
				if err != nil {
					t.Fatal(err)
				}

				testHold.UpdatedAt = responseHold.UpdatedAt
				testHold.CreatedAt = responseHold.CreatedAt
				testHold.ID = responseHold.ID
				testHold.UserID = responseHold.UserID
				testHold.AddedBy = responseHold.AddedBy
				expected := string(encode(testHold))

				if expected != string(b) {
					t.Fatalf("Expected %v, got %v", expected, string(b))
				} else {
					t.Logf("Got hold %v", expected)
				}
			},
		}
	}

	apiKeyEQ := func(apikey models.APIKey) ResponseTester {
		testAPIKey := apikey
		return ResponseTester{
			Name: "API Key Response Tester",
			Test: func(t *testing.T, testPrefix string, _ int, b []byte) {
				var responseAPIKey models.APIKey
				err := json.Unmarshal(b, &responseAPIKey)
				if err != nil {
					t.Fatal(err)
				}

				testAPIKey.UpdatedAt = responseAPIKey.UpdatedAt
				testAPIKey.CreatedAt = responseAPIKey.CreatedAt
				testAPIKey.UserID = responseAPIKey.UserID
				testAPIKey.Key = responseAPIKey.Key
				expected := string(encode(testAPIKey))

				if expected != string(b) {
					t.Fatalf("Expected %v, got %v", expected, string(b))
				} else {
					t.Logf("Got api key %v", expected)
				}
			},
		}
	}

	notificationEQ := func(notification models.Notification) ResponseTester {
		testNotification := notification
		return ResponseTester{
			Name: "Notification Response Tester",
			Test: func(t *testing.T, testPrefix string, _ int, b []byte) {
				var responseNotification models.Notification
				err := json.Unmarshal(b, &responseNotification)
				if err != nil {
					t.Fatal(err)
				}

				testNotification.UpdatedAt = responseNotification.UpdatedAt
				testNotification.CreatedAt = responseNotification.CreatedAt
				testNotification.ID = responseNotification.ID
				testNotification.UserID = responseNotification.UserID
				testNotification.AddedBy = responseNotification.AddedBy
				expected := string(encode(testNotification))

				if expected != string(b) {
					t.Fatalf("Expected %v, got %v", expected, string(b))
				} else {
					t.Logf("Got notification %v", expected)
				}
			},
		}
	}

	tester.Test("Base User Endpoints", func(test *Tester) {
		searchUser := models.User{
			Name:  "Search User",
			Email: "star.testing.search@test.mkr.cx",
			Role:  "admin",
			Type:  "other",
		}

		db.FirstOrCreate(&searchUser, &searchUser)

		test.Endpoint("/api/users/search", fiber.MethodGet).
			WithQuery(QueryArgs{"query": "star.testing.search"}).
			Test("Search Users", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:search", "leash.users:target_others"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		test.Endpoint("/api/users/search", fiber.MethodGet).
			WithQuery(QueryArgs{"query": "star.testing.search", "with_trainings": "true"}).
			Test("Search Users With Trainings", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:search", "leash.users:target_others", "leash.users.others.trainings:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		test.Endpoint("/api/users/search", fiber.MethodGet).
			WithQuery(QueryArgs{"query": "star.testing.search", "with_holds": "true"}).
			Test("Search Users With Holds", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:search", "leash.users:target_others", "leash.users.others.holds:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		test.Endpoint("/api/users/search", fiber.MethodGet).
			WithQuery(QueryArgs{"query": "star.testing.search", "with_api_keys": "true"}).
			Test("Search Users With Api Keys", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:search", "leash.users:target_others", "leash.users.others.apikeys:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		test.Endpoint("/api/users/search", fiber.MethodGet).
			WithQuery(QueryArgs{"query": "star.testing.search", "with_updates": "true"}).
			Test("Search Users With Updates", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:search", "leash.users:target_others", "leash.users.others.updates:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		test.Endpoint("/api/users/search", fiber.MethodGet).
			WithQuery(QueryArgs{"query": "star.testing.search", "with_notifications": "true"}).
			Test("Search Users With Notifications", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:search", "leash.users:target_others", "leash.users.others.notifications:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		testingUser := models.User{
			Name:           "New User",
			Email:          "new@testing.mkr.cx",
			Role:           "member",
			Type:           "other",
			GraduationYear: 0,
			Major:          "",
		}

		db.FirstOrCreate(&testingUser, &testingUser)
		db.Unscoped().Delete(&testingUser)

		test.Endpoint("/api/users", fiber.MethodPost).
			CleanupUser(func(_ string, _ models.User) error {
				return db.Unscoped().Delete(&models.User{}, &models.User{Email: "new@testing.mkr.cx"}).Error
			}).
			WithBody(encode(map[string]interface{}{
				"name":  "New User",
				"email": "new@testing.mkr.cx",
				"role":  "member",
				"type":  "other",
			})).
			Test("Create User", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:create"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(testingUser),
					)
			})
	})

	tester.Test("User Get Endpoints", func(test *Tester) {
		responseUser := models.User{
			Name:  "Get Test User",
			Email: "get@testing.mkr.cx",
			Role:  "admin",
			Type:  "other",
		}

		cardID := "1234567890test"
		responseUser.CardID = &cardID

		db.Unscoped().Delete(&models.User{}, &models.User{Email: responseUser.Email})
		db.Unscoped().Delete(&models.User{}, &models.User{CardID: responseUser.CardID})
		db.Create(&responseUser)

		test.Endpoint("/api/users/get/email/"+responseUser.Email, fiber.MethodGet).
			Test("Get User By Email", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:email"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/email/"+responseUser.Email, fiber.MethodGet).
			WithQuery(QueryArgs{"with_trainings": "true"}).
			Test("Get User By Email With Trainings", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:email", "leash.users.get.trainings:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/email/"+responseUser.Email, fiber.MethodGet).
			WithQuery(QueryArgs{"with_holds": "true"}).
			Test("Get User By Email With Holds", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:email", "leash.users.get.holds:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/email/"+responseUser.Email, fiber.MethodGet).
			WithQuery(QueryArgs{"with_api_keys": "true"}).
			Test("Get User By Email With Api Keys", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:email", "leash.users.get.apikeys:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/email/"+responseUser.Email, fiber.MethodGet).
			WithQuery(QueryArgs{"with_updates": "true"}).
			Test("Get User By Email With Updates", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:email", "leash.users.get.updates:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/email/"+responseUser.Email, fiber.MethodGet).
			WithQuery(QueryArgs{"with_notifications": "true"}).
			Test("Get User By Email With Notifications", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:email", "leash.users.get.notifications:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/card/"+*responseUser.CardID, fiber.MethodGet).
			Test("Get User By Card ID", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:card"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/card/"+*responseUser.CardID, fiber.MethodGet).
			WithQuery(QueryArgs{"with_trainings": "true"}).
			Test("Get User By Card ID With Trainings", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:card", "leash.users.get.trainings:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/card/"+*responseUser.CardID, fiber.MethodGet).
			WithQuery(QueryArgs{"with_holds": "true"}).
			Test("Get User By Card ID With Holds", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:card", "leash.users.get.holds:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/card/"+*responseUser.CardID, fiber.MethodGet).
			WithQuery(QueryArgs{"with_api_keys": "true"}).
			Test("Get User By Card ID With Api Keys", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:card", "leash.users.get.apikeys:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/card/"+*responseUser.CardID, fiber.MethodGet).
			WithQuery(QueryArgs{"with_updates": "true"}).
			Test("Get User By Card ID With Updates", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:card", "leash.users.get.updates:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/get/card/"+*responseUser.CardID, fiber.MethodGet).
			WithQuery(QueryArgs{"with_notifications": "true"}).
			Test("Get User By Card ID With Notifications", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.get:card", "leash.users.get.notifications:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		db.Unscoped().Delete(&models.User{}, &models.User{Email: responseUser.Email})
	})

	tester.Test("Self User Endpoints", func(test *Tester) {
		responseUser := models.User{
			Name:  "Response Test User",
			Email: "response@mkr.cx",
			Role:  "admin",
			Type:  "other",
		}

		test.Endpoint("/api/users/self", fiber.MethodGet).
			Test("Get Self", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:get"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/self", fiber.MethodGet).
			WithQuery(QueryArgs{"with_trainings": "true"}).
			Test("Get Self With Trainings", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:get", "leash.users.self.trainings:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/self", fiber.MethodGet).
			WithQuery(QueryArgs{"with_holds": "true"}).
			Test("Get Self With Holds", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:get", "leash.users.self.holds:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		apiKeyResponseUser := responseUser
		apiKeyResponseUser.APIKeys = []models.APIKey{{
			Key:         "test",
			UserID:      responseUser.ID,
			FullAccess:  true,
			Permissions: []string{},
		}}

		getSelfWithApiKeys := ResponseTester{
			Name: "Get Self With Api Keys",
			Test: func(t *testing.T, testPrefix string, _ int, b []byte) {
				var responseUser models.User
				err := json.Unmarshal(b, &responseUser)
				if err != nil {
					t.Fatal(err)
				}

				if len(responseUser.APIKeys) != 1 {
					t.Fatalf("Expected 1 api key, got %d", len(responseUser.APIKeys))
				}

				testAPIKey := responseUser.APIKeys[0]

				expectApiKey := models.APIKey{
					Key:         testPrefix + ".response",
					UserID:      responseUser.ID,
					FullAccess:  true,
					Permissions: []string{},
				}

				expectApiKey.UpdatedAt = testAPIKey.UpdatedAt
				expectApiKey.CreatedAt = testAPIKey.CreatedAt

				expectUser := responseUser
				expectUser.APIKeys = []models.APIKey{expectApiKey}
				expectUser.ID = responseUser.ID
				expectUser.UpdatedAt = responseUser.UpdatedAt
				expectUser.CreatedAt = responseUser.CreatedAt

				expected := string(encode(expectUser))

				if expected != string(b) {
					t.Fatalf("Expected %v, got %v", expected, string(b))
				} else {
					t.Logf("Got user %v", expected)
				}
			},
		}

		test.Endpoint("/api/users/self", fiber.MethodGet).
			WithQuery(QueryArgs{"with_api_keys": "true"}).
			Test("Get Self With Api Keys", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:get", "leash.users.self.apikeys:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						getSelfWithApiKeys,
					)
			})

		test.Endpoint("/api/users/self", fiber.MethodGet).
			WithQuery(QueryArgs{"with_updates": "true"}).
			Test("Get Self With Updates", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:get", "leash.users.self.updates:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/self", fiber.MethodGet).
			WithQuery(QueryArgs{"with_notifications": "true"}).
			Test("Get Self With Notifications", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:get", "leash.users.self.notifications:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		test.Endpoint("/api/users/self/updates", fiber.MethodGet).
			Test("Get Self Updates", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.updates:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(0),
					)
			})

		test.Endpoint("/api/users/self", fiber.MethodPatch).
			Test("Update Self Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:update"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(responseUser),
					)
			})

		updateResponseUser := responseUser
		updateResponseUser.Name = "New Name"

		test.Endpoint("/api/users/self", fiber.MethodPatch).
			WithBody(encode(map[string]interface{}{
				"name": "New Name",
			})).
			Test("Update Self Normal", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:update"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(updateResponseUser),
					)
			})

		updateResponseUser = responseUser
		updateResponseUser.Role = "member"

		test.Endpoint("/api/users/self", fiber.MethodPatch).
			WithBody(encode(map[string]interface{}{
				"role": "member",
			})).
			Test("Update Self Role", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:update", "leash.users.self:update_role"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(updateResponseUser),
					)
			})

		updateResponseUser = responseUser
		card_id := "1234567890"
		updateResponseUser.CardID = &card_id

		test.Endpoint("/api/users/self", fiber.MethodPatch).
			WithBody(encode(map[string]interface{}{
				"card_id": "1234567890",
			})).
			Test("Update Self Card ID", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self:update", "leash.users.self:update_card_id"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(updateResponseUser),
					)
			})

		test.Endpoint("/api/users/self/updates", fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				update := models.UserUpdate{
					UserID:   user.ID,
					EditedBy: user.ID,
					Field:    "test",
					OldValue: "old",
					NewValue: "new",
				}

				return db.Create(&update).Error
			}).
			Test("Get Self Updates Not Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.updates:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		test.Endpoint("/api/users/self/trainings", fiber.MethodGet).
			Test("Get Self Trainings Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.trainings:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(0),
					)
			})

		test.Endpoint("/api/users/self/trainings", fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				training := models.Training{
					TrainingType: "other",
					UserID:       user.ID,
					AddedBy:      user.ID,
				}

				return db.Create(&training).Error
			}).
			Test("Get Self Trainings Not Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.trainings:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		newTraining := models.Training{
			TrainingType: "other",
			UserID:       responseUser.ID,
			AddedBy:      responseUser.ID,
		}

		test.Endpoint("/api/users/self/trainings", fiber.MethodPost).
			WithBody(encode(map[string]interface{}{
				"training_type": "other",
			})).
			Test("Create Self Training", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.trainings:create"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						trainingEQ(newTraining),
					)
			})

		test.Endpoint("/api/users/self/trainings/other", fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				training := newTraining
				training.UserID = user.ID
				return db.Create(&training).Error
			}).
			Test("Get Self Single Training", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.trainings:target", "leash.users.self.trainings:get"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						trainingEQ(newTraining),
					)
			})

		test.Endpoint("/api/users/self/trainings/other", fiber.MethodDelete).
			SetupUser(func(_ string, user models.User) error {
				training := newTraining
				training.UserID = user.ID
				return db.Create(&training).Error
			}).
			Test("Delete Self Training", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.trainings:target", "leash.users.self.trainings:delete"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

		test.Endpoint("/api/users/self/holds", fiber.MethodGet).
			Test("Get Self Holds Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.holds:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(0),
					)
			})

		newHold := models.Hold{
			Reason:    "Test Hold",
			HoldType:  "other",
			UserID:    responseUser.ID,
			AddedBy:   responseUser.ID,
			Priority:  10,
			HoldStart: nil,
			HoldEnd:   nil,
		}

		test.Endpoint("/api/users/self/holds", fiber.MethodPost).
			WithBody(encode(map[string]interface{}{
				"reason":    "Test Hold",
				"hold_type": "other",
				"priority":  10,
			})).
			Test("Create Self Hold", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.holds:create"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						holdEQ(newHold),
					)
			})

		test.Endpoint("/api/users/self/holds/other", fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				hold := newHold
				hold.UserID = user.ID
				return db.Create(&hold).Error
			}).
			Test("Get Self Single Hold", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.holds:target", "leash.users.self.holds:get"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						holdEQ(newHold),
					)
			})

		test.Endpoint("/api/users/self/holds/other", fiber.MethodDelete).
			SetupUser(func(_ string, user models.User) error {
				hold := newHold
				hold.UserID = user.ID
				return db.Create(&hold).Error
			}).
			Test("Delete Self Hold", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.holds:target", "leash.users.self.holds:delete"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

		test.Endpoint("/api/users/self/apikeys", fiber.MethodGet).
			Test("Get Self Api Keys Start", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.apikeys:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		test.Endpoint("/api/users/self/apikeys", fiber.MethodPost).
			WithBody(encode(map[string]interface{}{
				"full_access": true,
				"permissions": []string{},
			})).
			CleanupUser(func(_ string, user models.User) error {
				return db.Delete(&models.APIKey{}, &models.APIKey{UserID: user.ID}).Error
			}).
			Test("Create Self Api Key", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.apikeys:create"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						apiKeyEQ(models.APIKey{
							Key:         "",
							UserID:      responseUser.ID,
							FullAccess:  true,
							Permissions: []string{},
						}),
					)
			})

		test.Endpoint("/api/users/self/apikeys/test", fiber.MethodDelete).
			SetupUser(func(_ string, user models.User) error {
				apiKey := models.APIKey{
					Key:         "test",
					UserID:      user.ID,
					FullAccess:  true,
					Permissions: []string{},
				}
				return db.Create(&apiKey).Error
			}).
			CleanupUser(func(_ string, user models.User) error {
				return db.Unscoped().Delete(&models.APIKey{}, &models.APIKey{Key: "test"}).Error
			}).
			Test("Delete Self Api Key", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.apikeys:target", "leash.users.self.apikeys:delete"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

		test.Endpoint("/api/users/self/apikeys/test", fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				apiKey := models.APIKey{
					Key:         "test",
					UserID:      user.ID,
					FullAccess:  true,
					Permissions: []string{},
				}
				return db.Create(&apiKey).Error
			}).
			CleanupUser(func(_ string, user models.User) error {
				return db.Unscoped().Delete(&models.APIKey{}, &models.APIKey{Key: "test"}).Error
			}).
			Test("Get Self Single Api Key", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.apikeys:target", "leash.users.self.apikeys:get"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						apiKeyEQ(models.APIKey{
							Key:         "test",
							UserID:      responseUser.ID,
							FullAccess:  true,
							Permissions: []string{},
						}),
					)
			})

		test.Endpoint("/api/users/self/notifications", fiber.MethodGet).
			Test("Get Self Notifications Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.notifications:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(0),
					)
			})

		newNotification := models.Notification{
			Title:   "Test Notification",
			Message: "Test Message",
			UserID:  responseUser.ID,
			AddedBy: responseUser.ID,
		}

		test.Endpoint("/api/users/self/notifications", fiber.MethodPost).
			WithBody(encode(map[string]interface{}{
				"title":   "Test Notification",
				"message": "Test Message",
			})).
			Test("Create Self Notification", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.notifications:create"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						notificationEQ(newNotification),
					)
			})

		test.Endpoint("/api/users/self/notifications", fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				notification := newNotification
				notification.UserID = user.ID
				return db.Create(&notification).Error
			}).
			Test("Get Self Notifications Not Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.notifications:list"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		testNotification := newNotification
		db.FirstOrCreate(&testNotification, &testNotification)
		db.Unscoped().Delete(&testNotification)
		notificationID := testNotification.ID

		test.Endpoint(fmt.Sprintf("/api/users/self/notifications/%d", notificationID), fiber.MethodDelete).
			SetupUser(func(_ string, user models.User) error {
				notification := testNotification
				notification.UserID = user.ID
				return db.Create(&notification).Error
			}).
			CleanupUser(func(_ string, user models.User) error {
				return db.Unscoped().Delete(&models.Notification{}, &models.Notification{ID: notificationID}).Error
			}).
			Test("Delete Self Notification", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.notifications:target", "leash.users.self.notifications:delete"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/self/notifications/%d", notificationID), fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				notification := testNotification
				notification.UserID = user.ID
				return db.Create(&notification).Error
			}).
			CleanupUser(func(_ string, user models.User) error {
				return db.Unscoped().Delete(&models.Notification{}, &models.Notification{ID: notificationID}).Error
			}).
			Test("Get Self Single Notification", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_self", "leash.users.self.notifications:target", "leash.users.self.notifications:get"}).
					MinimumRole(ROLE_MEMBER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						notificationEQ(testNotification),
					)
			})
	})

	tester.Test("Other User Endpoints", func(test *Tester) {
		testingUser := models.User{
			Name:           "New User",
			Email:          "new@testing.mkr.cx",
			Role:           "member",
			Type:           "other",
			GraduationYear: 0,
			Major:          "",
		}

		db.FirstOrCreate(&testingUser, &testingUser)
		db.Unscoped().Delete(&testingUser)

		createUser := func(_ string, _ models.User) error {
			return db.FirstOrCreate(&testingUser, &testingUser).Error
		}

		cleanupUser := func(_ string, _ models.User) error {
			purgeUser(db, testingUser)
			return db.Unscoped().Delete(&models.User{}, &models.User{Email: "new@testing.mkr.cx"}).Error
		}

		userEP := fmt.Sprintf("/api/users/%d", testingUser.ID)

		test.Endpoint(userEP, fiber.MethodGet).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:get"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(testingUser),
					)
			})

		test.Endpoint(userEP, fiber.MethodGet).
			WithQuery(QueryArgs{"with_trainings": "true"}).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User With Trainings", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:get", "leash.users.others.trainings:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(testingUser),
					)
			})

		test.Endpoint(userEP, fiber.MethodGet).
			WithQuery(QueryArgs{"with_holds": "true"}).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User With Holds", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:get", "leash.users.others.holds:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(testingUser),
					)
			})

		test.Endpoint(userEP, fiber.MethodGet).
			WithQuery(QueryArgs{"with_api_keys": "true"}).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User With Api Keys", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:get", "leash.users.others.apikeys:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(testingUser),
					)
			})

		test.Endpoint(userEP, fiber.MethodGet).
			WithQuery(QueryArgs{"with_updates": "true"}).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User With Updates", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:get", "leash.users.others.updates:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(testingUser),
					)
			})

		test.Endpoint(userEP, fiber.MethodGet).
			WithQuery(QueryArgs{"with_notifications": "true"}).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User With Notifications", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:get", "leash.users.others.notifications:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(testingUser),
					)
			})

		updateUser := testingUser
		updateUser.Name = "New Name"

		test.Endpoint(userEP, fiber.MethodPatch).
			WithBody(encode(map[string]interface{}{
				"name": "New Name",
			})).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Update User", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:update"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(updateUser),
					)
			})

		updateUser = testingUser
		updateUser.Role = "member"

		test.Endpoint(userEP, fiber.MethodPatch).
			WithBody(encode(map[string]interface{}{
				"role": "member",
			})).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Update User Role", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:update", "leash.users.others:update_role"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(updateUser),
					)
			})

		updateUser = testingUser
		card_id := "1234567890"
		updateUser.CardID = &card_id

		test.Endpoint(userEP, fiber.MethodPatch).
			WithBody(encode(map[string]interface{}{
				"card_id": "1234567890",
			})).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Update User Card ID", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:update", "leash.users.others:update_card_id"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(updateUser),
					)
			})

		test.Endpoint(userEP, fiber.MethodDelete).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Delete User", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:delete"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/updates", testingUser.ID), fiber.MethodGet).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User Updates", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.updates:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(0),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/trainings", testingUser.ID), fiber.MethodGet).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User Trainings Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.trainings:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(0),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/trainings", testingUser.ID), fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				training := models.Training{
					TrainingType: "other",
					UserID:       testingUser.ID,
					AddedBy:      user.ID,
				}

				return db.Create(&training).Error
			}).
			CleanupUser(cleanupUser).
			Test("Get User Trainings Not Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.trainings:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		newTraining := models.Training{
			TrainingType: "other",
			UserID:       testingUser.ID,
			AddedBy:      testingUser.ID,
		}

		test.Endpoint(fmt.Sprintf("/api/users/%d/trainings", testingUser.ID), fiber.MethodPost).
			WithBody(encode(map[string]interface{}{
				"training_type": "other",
			})).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Create User Training", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.trainings:create"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						trainingEQ(newTraining),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/trainings/other", testingUser.ID), fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				training := newTraining
				training.UserID = testingUser.ID
				return db.Create(&training).Error
			}).
			CleanupUser(cleanupUser).
			Test("Get User Single Training", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.trainings:target", "leash.users.others.trainings:get"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						trainingEQ(newTraining),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/trainings/other", testingUser.ID), fiber.MethodDelete).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				training := newTraining
				training.UserID = testingUser.ID
				return db.Create(&training).Error
			}).
			CleanupUser(cleanupUser).
			Test("Delete User Training", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.trainings:target", "leash.users.others.trainings:delete"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/holds", testingUser.ID), fiber.MethodGet).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User Holds Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.holds:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(0),
					)
			})

		newHold := models.Hold{
			Reason:    "Test Hold",
			HoldType:  "other",
			UserID:    testingUser.ID,
			AddedBy:   testingUser.ID,
			Priority:  10,
			HoldStart: nil,
			HoldEnd:   nil,
		}

		test.Endpoint(fmt.Sprintf("/api/users/%d/holds", testingUser.ID), fiber.MethodPost).
			WithBody(encode(map[string]interface{}{
				"reason":    "Test Hold",
				"hold_type": "other",
				"priority":  10,
			})).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Create User Hold", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.holds:create"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						holdEQ(newHold),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/holds/other", testingUser.ID), fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				hold := newHold
				hold.UserID = testingUser.ID
				return db.Create(&hold).Error
			}).
			CleanupUser(cleanupUser).
			Test("Get User Single Hold", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.holds:target", "leash.users.others.holds:get"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						holdEQ(newHold),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/holds/other", testingUser.ID), fiber.MethodDelete).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				hold := newHold
				hold.UserID = testingUser.ID
				return db.Create(&hold).Error
			}).
			CleanupUser(cleanupUser).
			Test("Delete User Hold", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.holds:target", "leash.users.others.holds:delete"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/apikeys", testingUser.ID), fiber.MethodGet).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User Api Keys Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.apikeys:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(0),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/apikeys", testingUser.ID), fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				apiKey := models.APIKey{
					Key:         "test",
					UserID:      testingUser.ID,
					FullAccess:  true,
					Permissions: []string{},
				}
				return db.Create(&apiKey).Error
			}).
			CleanupUser(cleanupUser).
			Test("Get User Api Keys Not Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.apikeys:list"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/apikeys", testingUser.ID), fiber.MethodPost).
			WithBody(encode(map[string]interface{}{
				"full_access": true,
				"permissions": []string{},
			})).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Create User Api Key", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.apikeys:create"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						apiKeyEQ(models.APIKey{
							Key:         "",
							UserID:      testingUser.ID,
							FullAccess:  true,
							Permissions: []string{},
						}),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/apikeys/test", testingUser.ID), fiber.MethodDelete).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				apiKey := models.APIKey{
					Key:         "test",
					UserID:      testingUser.ID,
					FullAccess:  true,
					Permissions: []string{},
				}
				return db.Create(&apiKey).Error
			}).
			CleanupUser(cleanupUser).
			Test("Delete User Api Key", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.apikeys:target", "leash.users.others.apikeys:delete"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/apikeys/test", testingUser.ID), fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				apiKey := models.APIKey{
					Key:         "test",
					UserID:      testingUser.ID,
					FullAccess:  true,
					Permissions: []string{},
				}
				return db.Create(&apiKey).Error
			}).
			CleanupUser(cleanupUser).
			Test("Get User Single Api Key", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.apikeys:target", "leash.users.others.apikeys:get"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						apiKeyEQ(models.APIKey{
							Key:         "test",
							UserID:      testingUser.ID,
							FullAccess:  true,
							Permissions: []string{},
						}),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/notifications", testingUser.ID), fiber.MethodGet).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get User Notifications Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.notifications:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(0),
					)
			})

		newNotification := models.Notification{
			Title:   "Test Notification",
			Message: "Test Message",
			UserID:  testingUser.ID,
			AddedBy: testingUser.ID,
		}

		test.Endpoint(fmt.Sprintf("/api/users/%d/notifications", testingUser.ID), fiber.MethodPost).
			WithBody(encode(map[string]interface{}{
				"title":   "Test Notification",
				"message": "Test Message",
			})).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Create User Notification", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.notifications:create"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						notificationEQ(newNotification),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/notifications", testingUser.ID), fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				notification := newNotification
				notification.UserID = testingUser.ID
				return db.Create(&notification).Error
			}).
			CleanupUser(cleanupUser).
			Test("Get User Notifications Not Empty", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.notifications:list"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						listLengthEQ(1),
					)
			})

		testNotification := newNotification
		db.FirstOrCreate(&testNotification, &testNotification)
		db.Unscoped().Delete(&testNotification)

		notificationID := testNotification.ID

		test.Endpoint(fmt.Sprintf("/api/users/%d/notifications/%d", testingUser.ID, notificationID), fiber.MethodDelete).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				notification := testNotification
				notification.UserID = testingUser.ID
				return db.Create(&notification).Error
			}).
			CleanupUser(cleanupUser).
			Test("Delete User Notification", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.notifications:target", "leash.users.others.notifications:delete"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d/notifications/%d", testingUser.ID, notificationID), fiber.MethodGet).
			SetupUser(func(_ string, user models.User) error {
				createUser("", user)
				notification := testNotification
				notification.UserID = testingUser.ID
				return db.Create(&notification).Error
			}).
			CleanupUser(cleanupUser).
			Test("Get User Single Notification", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others.notifications:target", "leash.users.others.notifications:get"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						notificationEQ(testNotification),
					)
			})
	})

	tester.Test("Service User Endpoints", func(test *Tester) {
		serviceUser := models.User{
			Name:  "Service User",
			Email: "service_testing@mkrcx",
			Role:  "service",
			Type:  "other",
			Permissions: []string{
				"leash.users:target_self",
			},
		}

		db.FirstOrCreate(&serviceUser, &serviceUser)
		db.Unscoped().Delete(&serviceUser)

		createUser := func(_ string, _ models.User) error {
			if err := db.FirstOrCreate(&serviceUser, &serviceUser).Error; err != nil {
				return err
			}

			test.enforcer.SetPermissionsForUser(serviceUser, serviceUser.Permissions)
			return test.enforcer.SavePolicy()
		}

		cleanupUser := func(_ string, _ models.User) error {
			purgeUser(db, serviceUser)
			return db.Unscoped().Delete(&models.User{}, &models.User{Email: serviceUser.Email}).Error
		}

		test.Endpoint("/api/users/service", fiber.MethodPost).
			CleanupUser(cleanupUser).
			WithBody(encode(map[string]interface{}{
				"name":        "Service User",
				"service_tag": "service_testing",
				"permissions": []string{
					"leash.users:target_self",
				},
			})).
			Test("Create Service User", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users.service:create"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(serviceUser),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d", serviceUser.ID), fiber.MethodGet).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Get Service User", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:get"}).
					MinimumRole(ROLE_VOLUNTEER).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(serviceUser),
					)
			})

		updateUser := serviceUser
		updateUser.Name = "New Name"

		test.Endpoint(fmt.Sprintf("/api/users/%d/service", serviceUser.ID), fiber.MethodPatch).
			WithBody(encode(map[string]interface{}{
				"name": "New Name",
			})).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Update Service User", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:service_update"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(updateUser),
					)
			})

		updateUser = serviceUser
		updateUser.Permissions = []string{
			"leash.users:target_self",
			"leash.users:target_others",
		}

		test.Endpoint(fmt.Sprintf("/api/users/%d/service", serviceUser.ID), fiber.MethodPatch).
			WithBody(encode(map[string]interface{}{
				"permissions": []string{
					"leash.users:target_self",
					"leash.users:target_others",
				},
			})).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Update Service User Permissions", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:service_update"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						userEQ(updateUser),
					)
			})

		test.Endpoint(fmt.Sprintf("/api/users/%d", serviceUser.ID), fiber.MethodDelete).
			SetupUser(createUser).
			CleanupUser(cleanupUser).
			Test("Delete Service User", func(e *EndpointTester) {
				e.RequiresPermissions([]string{"leash.users:target_others", "leash.users.others:delete"}).
					MinimumRole(ROLE_ADMIN).
					GivesResponse(
						statusCode(fiber.StatusOK),
						defaultStatusResponse,
					)
			})

	})

	tester.Test("Login Endpoints", func(test *Tester) {
		test.Endpoint("/auth/login", fiber.MethodGet).
			WithQuery(QueryArgs{
				"redirect": "/test",
				"state":    "test",
			}).
			Test("Login Redirect", func(e *EndpointTester) {
				e.GivesResponseNoAuth(statusCode(fiber.StatusFound))
			})

		tok, err := jwt.NewBuilder().
			Issuer(leash_auth.ISSUER).
			IssuedAt(time.Now()).
			Expiration(time.Now().Add(5*time.Minute)).
			Audience([]string{"leash", "login-callback"}).
			Claim("return", "/").
			Claim("state", "state").
			Build()

		if err != nil {
			t.Fatal(err)
		}

		signed, err := keys.Sign(tok)
		if err != nil {
			t.Fatal(err)
		}

		user := models.User{
			Name:  "Test User",
			Email: "test@mkr.cx",
		}

		db.FirstOrCreate(&user, &user)
		db.Unscoped().Delete(&user)

		test.Endpoint("/auth/callback", fiber.MethodGet).
			WithQuery(QueryArgs{
				"code":  user.Email,
				"state": string(signed),
			}).
			Test("Login Callback With Non-Existent User", func(e *EndpointTester) {
				e.GivesResponseNoAuth(statusCode(fiber.StatusUnauthorized))
			})

		db.Create(&user)

		test.Endpoint("/auth/callback", fiber.MethodGet).
			WithQuery(QueryArgs{
				"code":  user.Email,
				"state": string(signed),
			}).
			Test("Login Callback With User that doesn't have login permissions", func(e *EndpointTester) {
				e.GivesResponseNoAuth(statusCode(fiber.StatusUnauthorized))
			})

		test.enforcer.SetPermissionsForUser(user, []string{"leash:login"})
		test.enforcer.SavePolicy()

		test.Endpoint("/auth/callback", fiber.MethodGet).
			WithQuery(QueryArgs{
				"code":  user.Email,
				"state": string(signed),
			}).
			Test("Login Callback With User that has login permissions", func(e *EndpointTester) {
				e.GivesResponseNoAuth(statusCode(fiber.StatusFound))
			})
	})
}
