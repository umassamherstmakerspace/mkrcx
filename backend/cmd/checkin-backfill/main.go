package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func optionalTimestamp(raw, name string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("%s must be an RFC3339 timestamp", name)
	}
	value = value.UTC()
	return &value, nil
}

func main() {
	apply := flag.Bool("apply", false, "write the import; omitted means aggregate-only dry run")
	startRaw := flag.String("start", "", "optional inclusive RFC3339 source timestamp")
	endRaw := flag.String("end", "", "optional exclusive RFC3339 source timestamp")
	batchSize := flag.Int("batch-size", defaultBatchSize, "source rows per batch")
	flag.Parse()

	start, err := optionalTimestamp(*startRaw, "start")
	if err != nil {
		log.Fatal(err)
	}
	end, err := optionalTimestamp(*endRaw, "end")
	if err != nil {
		log.Fatal(err)
	}

	sourceDSN := strings.TrimSpace(os.Getenv("CARD_DATABASE_DSN"))
	targetDSN := strings.TrimSpace(os.Getenv("MKRCX_DATABASE_DSN"))
	if sourceDSN == "" || targetDSN == "" {
		log.Fatal("CARD_DATABASE_DSN and MKRCX_DATABASE_DSN are required")
	}
	source, err := gorm.Open(mysql.Open(sourceDSN), &gorm.Config{})
	if err != nil {
		log.Fatal("could not connect to the source database")
	}
	target, err := gorm.Open(mysql.Open(targetDSN), &gorm.Config{})
	if err != nil {
		log.Fatal("could not connect to the target database")
	}

	stats, err := runBackfill(source, target, backfillOptions{
		Apply:     *apply,
		Start:     start,
		End:       end,
		BatchSize: *batchSize,
	})
	if err != nil {
		log.Fatal(err)
	}
	mode := "dry-run"
	if *apply {
		mode = "apply"
	}
	log.Printf(
		"check-in backfill %s complete: scanned=%d linked=%d unresolved=%d invalid_card=%d ambiguous_card=%d would_insert=%d inserted=%d already_exists=%d",
		mode,
		stats.Scanned,
		stats.Linked,
		stats.Unresolved,
		stats.InvalidCard,
		stats.AmbiguousCard,
		stats.WouldInsert,
		stats.Inserted,
		stats.AlreadyExists,
	)
}
