package commands

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

func newCheckinExportAccessTest(t *testing.T) (*gorm.DB, leash_auth.EnforcerWrapper) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:checkin-export-access-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&checkinExportAccessTarget{}); err != nil {
		t.Fatal(err)
	}
	enforcer, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		t.Fatal(err)
	}
	return db, leash_auth.EnforcerWrapper{Enforcer: enforcer}
}

func directExportAccess(t *testing.T, wrapper leash_auth.EnforcerWrapper, id uint) bool {
	t.Helper()
	allowed, err := wrapper.HasDirectPermissionForUser(models.User{ID: id}, checkinExportAccessPermission)
	if err != nil {
		t.Fatal(err)
	}
	return allowed
}

func TestCheckinExportAccessGrantIsDryRunByDefaultAndPreservesOtherPermissions(t *testing.T) {
	db, wrapper := newCheckinExportAccessTest(t)
	target := checkinExportAccessTarget{ID: 7, Email: "operator@example.edu", Role: "staff", Type: "employee"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := wrapper.Enforcer.AddPermissionForUser("user:7", "leash.users.self:get"); err != nil {
		t.Fatal(err)
	}

	preview, err := changeCheckinExportAccess(db, wrapper.Enforcer, " OPERATOR@example.edu ", "grant", false)
	if err != nil {
		t.Fatal(err)
	}
	if preview.HadPermission || !preview.HasPermission || !preview.ChangeRequired || preview.Applied {
		t.Fatalf("unexpected grant preview: %+v", preview)
	}
	if directExportAccess(t, wrapper, target.ID) {
		t.Fatal("dry run granted export access")
	}

	applied, err := changeCheckinExportAccess(db, wrapper.Enforcer, target.Email, "grant", true)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.ChangeRequired || !applied.Applied || !directExportAccess(t, wrapper, target.ID) {
		t.Fatalf("grant was not applied: %+v", applied)
	}
	unrelated, err := wrapper.HasDirectPermissionForUser(models.User{ID: target.ID}, "leash.users.self:get")
	if err != nil || !unrelated {
		t.Fatalf("grant removed an unrelated direct permission: allowed=%t err=%v", unrelated, err)
	}

	repeated, err := changeCheckinExportAccess(db, wrapper.Enforcer, target.Email, "grant", true)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.ChangeRequired || repeated.Applied {
		t.Fatalf("idempotent grant reported a write: %+v", repeated)
	}
}

func TestCheckinExportAccessGrantRejectsStudentStaff(t *testing.T) {
	db, wrapper := newCheckinExportAccessTest(t)
	target := checkinExportAccessTarget{ID: 8, Email: "student@example.edu", Role: "staff", Type: "undergrad"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := changeCheckinExportAccess(db, wrapper.Enforcer, target.Email, "grant", true); err == nil {
		t.Fatal("student staff member received export access")
	}
	if directExportAccess(t, wrapper, target.ID) {
		t.Fatal("rejected grant changed authorization policy")
	}
}

func TestCheckinExportAccessRevokeWorksForAnyTargetAndPreservesOtherPermissions(t *testing.T) {
	db, wrapper := newCheckinExportAccessTest(t)
	target := checkinExportAccessTarget{ID: 9, Email: "former@example.edu", Role: "member", Type: "undergrad"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := wrapper.Enforcer.AddPermissionForUser("user:9", checkinExportAccessPermission); err != nil {
		t.Fatal(err)
	}
	if _, err := wrapper.Enforcer.AddPermissionForUser("user:9", "leash.users.self:get"); err != nil {
		t.Fatal(err)
	}

	preview, err := changeCheckinExportAccess(db, wrapper.Enforcer, target.Email, "revoke", false)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.HadPermission || preview.HasPermission || !preview.ChangeRequired || preview.Applied {
		t.Fatalf("unexpected revoke preview: %+v", preview)
	}
	if !directExportAccess(t, wrapper, target.ID) {
		t.Fatal("dry-run revoke changed authorization policy")
	}

	applied, err := changeCheckinExportAccess(db, wrapper.Enforcer, target.Email, "revoke", true)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || directExportAccess(t, wrapper, target.ID) {
		t.Fatalf("revoke was not applied: %+v", applied)
	}
	unrelated, err := wrapper.HasDirectPermissionForUser(models.User{ID: target.ID}, "leash.users.self:get")
	if err != nil || !unrelated {
		t.Fatalf("revoke removed an unrelated direct permission: allowed=%t err=%v", unrelated, err)
	}
}
