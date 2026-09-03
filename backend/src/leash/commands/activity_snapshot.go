package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/subcommands"
	leash_api "github.com/mkrcx/mkrcx/src/leash/api"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ActivitySnapshotCmd emits aggregate-only activity responses for staging review.
type ActivitySnapshotCmd struct{}

func (*ActivitySnapshotCmd) Name() string { return "activity_snapshot" }
func (*ActivitySnapshotCmd) Synopsis() string {
	return "Emit aggregate-only activity JSON for a staging snapshot"
}
func (*ActivitySnapshotCmd) Usage() string {
	return `activity_snapshot:
	Emit semester, academic-year, and 30-day aggregate activity responses as JSON.
`
}
func (*ActivitySnapshotCmd) SetFlags(_ *flag.FlagSet) {}

func (*ActivitySnapshotCmd) Execute(_ context.Context, _ *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	dsn := os.Getenv("DB_USERNAME") + ":" + os.Getenv("DB_PASSWORD") + "@tcp(" + os.Getenv("DB_HOST") + ")/" + os.Getenv("DB_TABLE") + "?parseTime=true"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to connect to activity database")
		return subcommands.ExitFailure
	}
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to load activity timezone")
		return subcommands.ExitFailure
	}

	now := time.Now()
	snapshotAt := now.UTC().Format(time.RFC3339)
	responses := map[string]interface{}{}
	for _, key := range []string{"semester", "academic_year", "30_days"} {
		response, buildErr := leash_api.BuildActivityResponse(db, key, now, location)
		if buildErr != nil {
			fmt.Fprintln(os.Stderr, "unable to build activity snapshot")
			return subcommands.ExitFailure
		}
		response.SnapshotAt = snapshotAt
		responses[key] = response
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(responses); err != nil {
		fmt.Fprintln(os.Stderr, "unable to encode activity snapshot")
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
