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

func (e *EndpointTester) testEndpoint(endpoint string, method string, body []byte, auth string, expectedStatus int) []byte {
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
		e.t.Fatalf("Expected status %d, got %d\n Reponse: %s", expectedStatus, status, string(b))
	}

	return b
}

func (e *EndpointTester) TestPermissions(endpoint string, method string, body []byte, permissions []string, expectedStatus int) {
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

		e.testEndpoint(endpoint, method, body, "API-Key "+e.testUser.APIKey.Key, status)
	}

	e.testUser.User = tmpUser
	e.db.Save(&e.testUser.User)
	e.enforcer.SetPermissionsForUser(e.testUser.User, e.testUser.User.Permissions)
}

func (e *EndpointTester) TestRoles(endpoint string, method string, body []byte, minimumRole int, expectedStatus int) {
	e.t.Log("Role test for: ", endpoint)

	for i := 0; i < len(e.roleUsers); i++ {
		testApikey := e.roleUsers[i].APIKey.Key
		testRole := e.roleUsers[i].User.Role

		tmpUser := e.roleUsers[i].User

		e.t.Log("Testing role: ", testRole)

		status := fiber.StatusUnauthorized

		if i >= minimumRole {
			status = expectedStatus
		}

		e.testEndpoint(endpoint, method, body, "API-Key "+testApikey, status)

		e.roleUsers[i].User = tmpUser
		e.db.Save(&e.roleUsers[i].User)
	}
}

type BodyTester func([]byte)

func (e *EndpointTester) TestReponse(endpoint string, method string, body []byte, bodyTester BodyTester, expectedStatus int) {
	e.t.Log("Response test for: ", endpoint)

	tmpUser := e.starUser.User

	b := e.testEndpoint(endpoint, method, body, "API-Key "+e.starUser.APIKey.Key, expectedStatus)

	bodyTester(b)
	e.starUser.User = tmpUser
	e.db.Save(&e.starUser.User)
}

func (e *EndpointTester) TestAll(testName string, endpoint string, method string, body []byte, permissions []string, minimumRole int, bodyTester BodyTester, expectedStatus int) {
	e.t.Logf("Running test: %s\n", testName)

	e.TestPermissions(endpoint, method, body, permissions, expectedStatus)
	e.TestRoles(endpoint, method, body, minimumRole, expectedStatus)
	e.TestReponse(endpoint, method, body, bodyTester, expectedStatus)
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
	// Initalize DB
	t.Log("Initalizing DB...")
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
	t.Log("Initalizing JWT Keys...")
	set, err := leash_auth.GenerateJWTKeySet()
	if err != nil {
		t.Fatal(err)
	}

	keys, err := leash_auth.CreateKeys(set)
	if err != nil {
		t.Fatal(err)
	}

	// Initalize RBAC
	t.Log("Initalizing RBAC...")
	enforcer, err := leash_auth.InitalizeCasbin(db)
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
	t.Log("Initalizing Fiber...")

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
			u := encode(u1)

			if string(u) != string(b) {
				t.Fatalf("Expected %v, got %v", string(u), string(b))
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
	t.Log("Testing Self Endpoints...")

	endpointTester.TestAll("Get Self",
		"/api/users/self", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK)

	endpointTester.TestAll("Get Self With Trainings",
		"/api/users/self?with_trainings=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.trainings:list"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK)

	endpointTester.TestAll("Get Self With Holds",
		"/api/users/self?with_holds=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.holds:list"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK)

	newUser = endpointTester.starUser.User
	newUser.APIKeys = append(newUser.APIKeys, endpointTester.starUser.APIKey)
	endpointTester.TestAll("Get Self With Api Keys",
		"/api/users/self?with_api_keys=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.apikeys:list"}, ROLE_MEMBER,
		userEQ(newUser),
		fiber.StatusOK)

	endpointTester.TestAll("Get Self With Updates",
		"/api/users/self?with_updates=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.updates:list"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK)

	endpointTester.TestAll("Get Self With Notifications",
		"/api/users/self?with_notifications=true", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:read", "leash.users.self.notifications:list"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK)

	endpointTester.TestAll("Get Self Updates Before Updates",
		"/api/users/self/updates", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.updates:list"}, ROLE_MEMBER,
		listCountEQ(0),
		fiber.StatusOK)

	endpointTester.TestAll("Update Self Empty",
		"/api/users/self", "PATCH",
		nil,
		[]string{"leash.users:target_self", "leash.users.self:update"}, ROLE_MEMBER,
		userEQ(endpointTester.starUser.User),
		fiber.StatusOK)

	newUser = endpointTester.starUser.User
	newUser.Name = "New Name"
	endpointTester.TestAll("Update Self Normal",
		"/api/users/self", "PATCH",
		[]byte("{\"name\":\"New Name\"}"),
		[]string{"leash.users:target_self", "leash.users.self:update"}, ROLE_MEMBER,
		userEQ(newUser),
		fiber.StatusOK)

	newUser = endpointTester.starUser.User
	newUser.Role = "member"
	endpointTester.TestAll("Update Self Role",
		"/api/users/self", "PATCH",
		[]byte("{\"role\":\"member\"}"),
		[]string{"leash.users:target_self", "leash.users.self:update", "leash.users.self:update_role"}, ROLE_ADMIN,
		userEQ(newUser),
		fiber.StatusOK)

	newUser = endpointTester.starUser.User
	card_id := "1234567890"
	newUser.CardID = &card_id
	endpointTester.TestAll("Update Self Card ID",
		"/api/users/self", "PATCH",
		[]byte("{\"card_id\":\"1234567890\"}"),
		[]string{"leash.users:target_self", "leash.users.self:update", "leash.users.self:update_card_id"}, ROLE_ADMIN,
		userEQ(newUser),
		fiber.StatusOK)

	endpointTester.TestAll("Get Self Updates After Updates",
		"/api/users/self/updates", "GET",
		nil,
		[]string{"leash.users:target_self", "leash.users.self.updates:list"}, ROLE_MEMBER,
		listCountEQ(3),
		fiber.StatusOK)
}
