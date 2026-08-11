package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

const (
	historicalScope      = "card-server:historical"
	historicalSource     = "card-server-history"
	historicalResolution = "historical_current_link"
	defaultBatchSize     = 500
	maxBatchSize         = 5000
)

var historicalCardPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

type sourceSwipe struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	CardNumber string
}

func (sourceSwipe) TableName() string { return "swipes" }

type targetUserCard struct {
	ID     uint
	CardID *string
}

type backfillOptions struct {
	Apply     bool
	Start     *time.Time
	End       *time.Time
	BatchSize int
}

type backfillStats struct {
	Scanned       int64
	Linked        int64
	Unresolved    int64
	InvalidCard   int64
	AmbiguousCard int64
	WouldInsert   int64
	Inserted      int64
	AlreadyExists int64
}

func normalizeHistoricalCard(raw string) (string, bool) {
	card := strings.ToLower(strings.TrimSpace(raw))
	return card, historicalCardPattern.MatchString(card)
}

func loadCurrentCardUsers(db *gorm.DB) (map[string]uint, map[string]struct{}, error) {
	var users []targetUserCard
	if err := db.Unscoped().Table("users").Select("id", "card_id").Where("card_id IS NOT NULL AND card_id <> ''").Scan(&users).Error; err != nil {
		return nil, nil, err
	}

	owners := make(map[string]uint, len(users))
	ambiguous := make(map[string]struct{})
	for _, user := range users {
		if user.ID == 0 || user.CardID == nil {
			continue
		}
		card, valid := normalizeHistoricalCard(*user.CardID)
		if !valid {
			continue
		}
		if existing, found := owners[card]; found && existing != user.ID {
			delete(owners, card)
			ambiguous[card] = struct{}{}
			continue
		}
		if _, isAmbiguous := ambiguous[card]; !isAmbiguous {
			owners[card] = user.ID
		}
	}
	return owners, ambiguous, nil
}

func getOrCreateHistoricalIdentity(db *gorm.DB, userID uint) (string, error) {
	if userID == 0 {
		return "", nil
	}
	var identity models.CheckinIdentity
	result := db.Where("user_id = ?", userID).First(&identity)
	if result.Error == nil {
		return identity.MemberUUID, nil
	}
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return "", result.Error
	}

	identity = models.CheckinIdentity{UserID: userID, MemberUUID: uuid.NewString()}
	if err := db.Create(&identity).Error; err == nil {
		return identity.MemberUUID, nil
	}
	if err := db.Where("user_id = ?", userID).First(&identity).Error; err != nil {
		return "", err
	}
	return identity.MemberUUID, nil
}

func checkBackfillSchema(source, target *gorm.DB) error {
	if !source.Migrator().HasTable(&sourceSwipe{}) {
		return errors.New("source swipes table is missing")
	}
	for _, table := range []interface{}{&models.User{}, &models.CheckinIdentity{}, &models.CheckinEvent{}} {
		if !target.Migrator().HasTable(table) {
			return errors.New("target check-in schema is not migrated")
		}
	}
	return nil
}

func validateBackfillOptions(options *backfillOptions) error {
	if options.BatchSize == 0 {
		options.BatchSize = defaultBatchSize
	}
	if options.BatchSize < 1 || options.BatchSize > maxBatchSize {
		return fmt.Errorf("batch size must be between 1 and %d", maxBatchSize)
	}
	if options.Start != nil {
		start := options.Start.UTC()
		options.Start = &start
	}
	if options.End != nil {
		end := options.End.UTC()
		options.End = &end
	}
	if options.Start != nil && options.End != nil && !options.End.After(*options.Start) {
		return errors.New("end must be after start")
	}
	return nil
}

func runBackfill(source, target *gorm.DB, options backfillOptions) (backfillStats, error) {
	var stats backfillStats
	if err := validateBackfillOptions(&options); err != nil {
		return stats, err
	}
	if err := checkBackfillSchema(source, target); err != nil {
		return stats, err
	}
	owners, ambiguous, err := loadCurrentCardUsers(target)
	if err != nil {
		return stats, errors.New("could not load target card ownership")
	}

	var lastID uint
	for {
		query := source.Where("id > ?", lastID).Order("id ASC").Limit(options.BatchSize)
		if options.Start != nil {
			query = query.Where("created_at >= ?", *options.Start)
		}
		if options.End != nil {
			query = query.Where("created_at < ?", *options.End)
		}
		var swipes []sourceSwipe
		if err := query.Find(&swipes).Error; err != nil {
			return stats, errors.New("could not read source swipes")
		}
		if len(swipes) == 0 {
			break
		}

		for _, swipe := range swipes {
			if swipe.ID == 0 || swipe.CreatedAt.IsZero() {
				return stats, errors.New("source swipe has an invalid ID or timestamp")
			}
			lastID = swipe.ID
			stats.Scanned++

			card, valid := normalizeHistoricalCard(swipe.CardNumber)
			userID := uint(0)
			resolution := "unresolved"
			if !valid {
				stats.InvalidCard++
				stats.Unresolved++
			} else if _, isAmbiguous := ambiguous[card]; isAmbiguous {
				stats.AmbiguousCard++
				stats.Unresolved++
			} else if owner := owners[card]; owner != 0 {
				userID = owner
				resolution = historicalResolution
				stats.Linked++
			} else {
				stats.Unresolved++
			}

			key := strconv.FormatUint(uint64(swipe.ID), 10)
			var existing models.CheckinEvent
			result := target.Where("idempotency_scope = ? AND idempotency_key = ?", historicalScope, key).First(&existing)
			if result.Error == nil {
				if !existing.OccurredAt.Equal(swipe.CreatedAt.UTC()) || existing.Source != historicalSource {
					return stats, fmt.Errorf("existing historical event %s conflicts with its source row", key)
				}
				stats.AlreadyExists++
				continue
			}
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return stats, errors.New("could not check target event")
			}
			stats.WouldInsert++
			if !options.Apply {
				continue
			}

			if err := target.Transaction(func(tx *gorm.DB) error {
				memberUUID, err := getOrCreateHistoricalIdentity(tx, userID)
				if err != nil {
					return err
				}
				event := models.CheckinEvent{
					OccurredAt:         swipe.CreatedAt.UTC(),
					UserID:             userID,
					MemberUUID:         memberUUID,
					LinkedAtTap:        false,
					IdentityResolution: resolution,
					Decision:           "unknown",
					Source:             historicalSource,
					IdempotencyScope:   historicalScope,
					IdempotencyKey:     key,
				}
				return tx.Create(&event).Error
			}); err != nil {
				return stats, fmt.Errorf("could not import historical event %s", key)
			}
			stats.Inserted++
		}
	}

	return stats, nil
}
