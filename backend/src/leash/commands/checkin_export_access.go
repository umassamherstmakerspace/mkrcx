package commands

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/google/subcommands"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const checkinExportAccessPermission = "leash.checkins:export"

type checkinExportAccessTarget struct {
	ID        uint
	Email     string
	Role      string
	Type      string
	DeletedAt gorm.DeletedAt
}

func (checkinExportAccessTarget) TableName() string { return "users" }

type checkinExportAccessResult struct {
	Role           string
	Type           string
	HadPermission  bool
	HasPermission  bool
	ChangeRequired bool
	Applied        bool
}

func changeCheckinExportAccess(db *gorm.DB, enforcer *casbin.SyncedEnforcer, email, action string, apply bool) (checkinExportAccessResult, error) {
	var result checkinExportAccessResult
	email = strings.TrimSpace(email)
	if email == "" {
		return result, errors.New("target email is required")
	}
	action = strings.ToLower(strings.TrimSpace(action))
	if action != "grant" && action != "revoke" {
		return result, errors.New("action must be grant or revoke")
	}

	var targets []checkinExportAccessTarget
	if err := db.Where("LOWER(email) = LOWER(?)", email).Limit(2).Find(&targets).Error; err != nil {
		return result, errors.New("could not look up the target user")
	}
	if len(targets) != 1 {
		return result, errors.New("target must match exactly one active user")
	}
	target := targets[0]
	result.Role = target.Role
	result.Type = target.Type

	if action == "grant" && (target.Type != "employee" || (target.Role != "staff" && target.Role != "admin")) {
		return result, errors.New("export access can only be granted to an employee with the staff or admin role")
	}

	if err := enforcer.LoadPolicy(); err != nil {
		return result, errors.New("could not load authorization policy")
	}
	wrapper := leash_auth.EnforcerWrapper{Enforcer: enforcer}
	hadPermission, err := wrapper.HasDirectPermissionForUser(structToUser(target), checkinExportAccessPermission)
	if err != nil {
		return result, errors.New("could not read direct export access")
	}
	result.HadPermission = hadPermission
	result.HasPermission = action == "grant"
	result.ChangeRequired = result.HadPermission != result.HasPermission
	if !apply || !result.ChangeRequired {
		return result, nil
	}

	subject := fmt.Sprintf("user:%d", target.ID)
	if action == "grant" {
		if _, err := enforcer.AddPermissionForUser(subject, checkinExportAccessPermission); err != nil {
			return result, errors.New("could not grant export access")
		}
	} else if _, err := enforcer.DeletePermissionForUser(subject, checkinExportAccessPermission); err != nil {
		return result, errors.New("could not revoke export access")
	}
	result.Applied = true
	return result, nil
}

func structToUser(target checkinExportAccessTarget) models.User {
	return models.User{ID: target.ID, Role: target.Role, Type: target.Type}
}

type CheckinExportAccessCmd struct {
	action string
	apply  bool
}

func (*CheckinExportAccessCmd) Name() string { return "checkin_export_access" }
func (*CheckinExportAccessCmd) Synopsis() string {
	return "Dry-run, grant, or revoke direct access to the privileged check-in CSV"
}
func (*CheckinExportAccessCmd) Usage() string {
	return `checkin_export_access -action grant|revoke [-apply]:
	Reads the exact target email from CHECKIN_EXPORT_USER_EMAIL or standard input.
	Without -apply, reports the planned change without writing authorization policy.
`
}
func (command *CheckinExportAccessCmd) SetFlags(flags *flag.FlagSet) {
	flags.StringVar(&command.action, "action", "", "grant or revoke")
	flags.BoolVar(&command.apply, "apply", false, "write the permission change; omitted means dry run")
}

func (command *CheckinExportAccessCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	targetEmail := strings.TrimSpace(os.Getenv("CHECKIN_EXPORT_USER_EMAIL"))
	if targetEmail == "" {
		fmt.Fprint(os.Stderr, "Target account email: ")
		value, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && len(value) == 0 {
			log.Print("could not read the target email")
			return subcommands.ExitFailure
		}
		targetEmail = strings.TrimSpace(value)
	}

	dsn := os.Getenv("DB_USERNAME") + ":" + os.Getenv("DB_PASSWORD") + "@tcp(" + os.Getenv("DB_HOST") + ")/" + os.Getenv("DB_TABLE") + "?parseTime=true"
	parameterizedLogger := logger.New(log.New(os.Stderr, "", log.LstdFlags), logger.Config{
		SlowThreshold:        200 * time.Millisecond,
		LogLevel:             logger.Warn,
		ParameterizedQueries: true,
		Colorful:             false,
	})
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: parameterizedLogger})
	if err != nil {
		log.Print("could not connect to the mkr.cx database")
		return subcommands.ExitFailure
	}
	enforcer, err := leash_auth.InitializeCasbin(db)
	if err != nil {
		log.Print("could not initialize authorization policy")
		return subcommands.ExitFailure
	}

	result, err := changeCheckinExportAccess(db, enforcer, targetEmail, command.action, command.apply)
	if err != nil {
		log.Print(err)
		return subcommands.ExitFailure
	}
	mode := "dry-run"
	if command.apply {
		mode = "apply"
	}
	log.Printf(
		"check-in export access %s complete: action=%s role=%s type=%s had_permission=%t has_permission=%t change_required=%t applied=%t",
		mode,
		command.action,
		result.Role,
		result.Type,
		result.HadPermission,
		result.HasPermission,
		result.ChangeRequired,
		result.Applied,
	)
	return subcommands.ExitSuccess
}
