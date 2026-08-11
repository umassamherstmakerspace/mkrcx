package leash_backend_api

import (
	"encoding/csv"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

func newCheckinExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:checkin-export-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.CheckinIdentity{}, &models.CheckinEvent{}, &models.CheckinExportAudit{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAuthorizeCheckinExportRequiresDirectSignedInUserPermission(t *testing.T) {
	db := newCheckinExportTestDB(t)
	enforcer, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		t.Fatal(err)
	}
	models.SetupEnforcer(enforcer)
	wrapper := leash_auth.EnforcerWrapper{Enforcer: enforcer}
	allowed := models.User{ID: 11, Role: "staff", Type: "employee"}
	if err := wrapper.SetPermissionsForUser(allowed, []string{checkinExportPermission}); err != nil {
		t.Fatal(err)
	}
	if _, err := enforcer.AddPermissionForUser("role:admin", checkinExportPermission); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		authentication leash_auth.Authentication
		wantStatus     int
	}{
		{
			name: "directly authorized user",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_USER,
				User:          allowed,
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusNoContent,
		},
		{
			name: "role permission does not authorize an admin",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_USER,
				User:          models.User{ID: 12, Role: "admin", Type: "employee"},
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusForbidden,
		},
		{
			name: "student staff with direct permission is ineligible",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_USER,
				User:          models.User{ID: 11, Role: "staff", Type: "undergrad"},
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusForbidden,
		},
		{
			name: "employee volunteer with direct permission is ineligible",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_USER,
				User:          models.User{ID: 11, Role: "volunteer", Type: "employee"},
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusForbidden,
		},
		{
			name: "api key cannot export",
			authentication: leash_auth.Authentication{
				Authenticator: leash_auth.AUTHENTICATOR_APIKEY,
				User:          allowed,
				Data:          models.APIKey{Key: "full", FullAccess: true},
				Enforcer:      wrapper,
			},
			wantStatus: fiber.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(c *fiber.Ctx) error {
				c.Locals("auth", test.authentication)
				return c.Next()
			}, authorizeCheckinExport, func(c *fiber.Ctx) error {
				return c.SendStatus(fiber.StatusNoContent)
			})
			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.wantStatus)
			}
		})
	}
}

func TestCheckinCSVUsesStableMemberIdentityAndAuditsScope(t *testing.T) {
	db := newCheckinExportTestDB(t)
	enforcer, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		t.Fatal(err)
	}
	models.SetupEnforcer(enforcer)
	wrapper := leash_auth.EnforcerWrapper{Enforcer: enforcer}
	requester := models.User{ID: 42, Role: "staff", Type: "employee"}
	if err := wrapper.SetPermissionsForUser(requester, []string{checkinExportPermission}); err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, time.August, 1, 4, 0, 0, 0, time.UTC)
	items := []models.FeedMessage{
		{
			Model: models.Model{CreatedAt: start.Add(time.Hour)}, UserID: 7,
			Title: cardLinkedTitle, CheckinDecision: checkinDecisionGreen,
			IdempotencyScope: "reader:one", IdempotencyKey: stringPointer("one"),
		},
		{
			Model: models.Model{CreatedAt: start.Add(2 * time.Hour)}, UserID: 7,
			Title: cardLinkedTitle, CheckinDecision: checkinDecisionYellow,
			IdempotencyScope: "reader:one", IdempotencyKey: stringPointer("two"),
		},
		{
			Model: models.Model{CreatedAt: start.Add(3 * time.Hour)},
			Title: cardNotLinkedTitle, CheckinDecision: checkinDecisionRed,
			IdempotencyScope: "reader:one", IdempotencyKey: stringPointer("three"),
		},
	}
	for index := range items {
		if err := persistCheckinEvent(db, &items[index]); err != nil {
			t.Fatal(err)
		}
	}

	app := fiber.New()
	app.Get("/export.csv", func(c *fiber.Ctx) error {
		c.Locals("db", db)
		c.Locals("auth", leash_auth.Authentication{
			Authenticator: leash_auth.AUTHENTICATOR_USER,
			User:          requester,
			Enforcer:      wrapper,
		})
		return c.Next()
	}, authorizeCheckinExport, models.GetQueryMiddleware[checkinExportRequest], exportCheckinsCSV)

	request := httptest.NewRequest(
		http.MethodGet,
		"/export.csv?start="+start.Format(time.RFC3339)+"&end="+start.Add(24*time.Hour).Format(time.RFC3339),
		nil,
	)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, body = %s", response.StatusCode, body)
	}
	if response.Header.Get("Cache-Control") != "no-store" || !strings.Contains(response.Header.Get("Content-Disposition"), "attachment") {
		t.Fatalf("missing safe download headers: %+v", response.Header)
	}

	rows, err := csv.NewReader(response.Body).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("CSV rows = %d, want 4: %+v", len(rows), rows)
	}
	if rows[1][2] == "" || rows[1][2] != rows[2][2] {
		t.Fatalf("member identity is missing or unstable: %q, %q", rows[1][2], rows[2][2])
	}
	if rows[3][2] != "" || rows[3][4] != "unresolved" {
		t.Fatalf("unknown card received a correlatable identity: %+v", rows[3])
	}

	var audit models.CheckinExportAudit
	if err := db.First(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if audit.RequestedBy != requester.ID || audit.RowCount != 3 || !audit.StartAt.Equal(start) || !audit.EndAt.Equal(start.Add(24*time.Hour)) {
		t.Fatalf("unexpected export audit: %+v", audit)
	}
}

func TestResolveCheckinEventIdentityPreservesOriginalUnknownState(t *testing.T) {
	db := newCheckinExportTestDB(t)
	key := "unknown-then-linked"
	item := models.FeedMessage{
		Model: models.Model{CreatedAt: time.Now().UTC().Add(-time.Minute)},
		Title: cardNotLinkedTitle, CheckinDecision: checkinDecisionRed,
		IdempotencyScope: "reader:one", IdempotencyKey: &key,
	}
	if err := persistCheckinEvent(db, &item); err != nil {
		t.Fatal(err)
	}
	if err := resolveCheckinEventIdentity(db, item, 77); err != nil {
		t.Fatal(err)
	}

	var event models.CheckinEvent
	if err := db.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.UserID != 77 || event.MemberUUID == "" || event.LinkedAtTap || event.IdentityResolution != "later_link" || event.Decision != checkinDecisionRed {
		t.Fatalf("unexpected resolved analytics event: %+v", event)
	}
}

func TestParseCheckinExportRangeRejectsAmbiguousOrOversizedRanges(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	for _, req := range []checkinExportRequest{
		{Start: "2026-01-01", End: start.Add(time.Hour).Format(time.RFC3339)},
		{Start: start.Format(time.RFC3339), End: start.Format(time.RFC3339)},
		{Start: start.Format(time.RFC3339), End: start.Add(maxCheckinExportRange + time.Hour).Format(time.RFC3339)},
	} {
		if _, _, err := parseCheckinExportRange(req); err == nil {
			t.Fatalf("invalid range was accepted: %+v", req)
		}
	}
}

func stringPointer(value string) *string {
	return &value
}
