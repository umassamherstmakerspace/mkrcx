package leash_helpers

import (
	"fmt"
	"github.com/glebarez/sqlite"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestPrinterManagementUsesExplicitPermissionAndAPIKeyScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:printer-permissions-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	e, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		t.Fatal(err)
	}
	if err = SetupCasbin(e); err != nil {
		t.Fatal(err)
	}
	wrapper := leash_auth.EnforcerWrapper{Enforcer: e}
	for _, role := range []string{"member", "volunteer", "staff", "admin"} {
		auth := leash_auth.Authentication{Authenticator: leash_auth.AUTHENTICATOR_USER, User: models.User{ID: 42, Role: role}, Enforcer: wrapper}
		if allowed := auth.Authorize("leash.printers:manage") == nil; allowed != (role == "admin") {
			t.Fatalf("unexpected manage permission for %s", role)
		}
		if allowed := auth.Authorize("leash.printers:read") == nil; allowed != (role != "member") {
			t.Fatalf("unexpected read permission for %s", role)
		}
	}
	auth := leash_auth.Authentication{Authenticator: leash_auth.AUTHENTICATOR_APIKEY, User: models.User{ID: 42, Role: "admin"}, Enforcer: wrapper, Data: models.APIKey{Key: "fixture"}}
	if auth.Authorize("leash.printers:manage") == nil {
		t.Fatal("unscoped API key may manage printers")
	}
	if _, err = e.AddPermissionForUser("apikey:fixture", "leash.printers:manage"); err != nil {
		t.Fatal(err)
	}
	if auth.Authorize("leash.printers:manage") != nil {
		t.Fatal("scoped administrator API key denied")
	}
}
