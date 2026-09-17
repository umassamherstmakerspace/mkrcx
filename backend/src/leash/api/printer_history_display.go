package leash_backend_api

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
)

type printerHistoryEstimate struct {
	Hours *float64 `json:"hours"`
	Jobs  int64    `json:"jobs"`
	Since string   `json:"since"`
}

// Approximate wear, not a reconstruction of every old job. Use the latest meter
// in each counter era. Small backwards corrections do not count as resets, and
// an isolated low reading followed by a return to the old range is ignored.
func printerMeterEstimate(meters []models.PrinterHistoricalEntry) (float64, time.Time, bool) {
	byDay := map[string]models.PrinterHistoricalEntry{}
	for _, meter := range meters {
		if meter.MeterHours == nil || math.IsNaN(*meter.MeterHours) || math.IsInf(*meter.MeterHours, 0) || *meter.MeterHours < 0 {
			continue
		}
		day := meter.RecordedAt.Format("2006-01-02")
		if old, ok := byDay[day]; !ok || *meter.MeterHours > *old.MeterHours {
			byDay[day] = meter
		}
	}
	ordered := make([]models.PrinterHistoricalEntry, 0, len(byDay))
	for _, meter := range byDay {
		ordered = append(ordered, meter)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].RecordedAt.Before(ordered[j].RecordedAt) })
	if len(ordered) == 0 {
		return 0, time.Time{}, false
	}
	offset, previous := 0.0, *ordered[0].MeterHours
	for i := 1; i < len(ordered); i++ {
		value := *ordered[i].MeterHours
		if previous >= 100 && value < previous/2 {
			if i+1 < len(ordered) && *ordered[i+1].MeterHours >= previous*0.8 {
				continue
			}
			// Do not turn a single high outlier into a counter reset.
			if i >= 2 && previous > *ordered[i-2].MeterHours*2 && value >= *ordered[i-2].MeterHours {
				previous = value
				continue
			}
			offset += previous
		}
		previous = value
	}
	last := ordered[len(ordered)-1]
	through := last.RecordedAt
	if last.DateOnly {
		through = time.Date(through.Year(), through.Month(), through.Day()+1, 0, 0, 0, 0, time.UTC)
	}
	return offset + previous, through, true
}

func readPrinterHistoryEstimate(db *gorm.DB, printer string, usage printerUsage, jobs int64, first *time.Time, meters, origins []models.PrinterHistoricalEntry) (printerHistoryEstimate, error) {
	estimate := printerHistoryEstimate{Jobs: jobs + usage.Jobs}
	if first != nil {
		estimate.Since = first.Format("2006-01-02")
	}
	if estimate.Since == "" && usage.FirstOutcome != nil {
		estimate.Since = usage.FirstOutcome.Format("2006-01-02")
	}
	if len(origins) > 0 {
		origin := origins[0]
		if strings.HasPrefix(origin.Body, "In service since ") {
			estimate.Since = strings.TrimSuffix(strings.SplitN(strings.TrimPrefix(origin.Body, "In service since "), " (", 2)[0], ".")
		} else {
			estimate.Since = origin.RecordedAt.Format("2006-01-02")
		}
	}
	hours, through, hasMeter := printerMeterEstimate(meters)
	if hasMeter {
		var later struct{ Seconds float64 }
		if err := db.Model(&models.PrinterHistoryEvent{}).Select("COALESCE(SUM(duration_seconds),0) AS seconds").
			Where("printer_id = ? AND recorded_at > ? AND event_type IN ?", printer, through, []string{"printer_completed", "printer_cancelled", "printer_failed"}).Scan(&later).Error; err != nil {
			return estimate, err
		}
		hours += later.Seconds / 3600
		estimate.Hours = &hours
	} else if usage.Seconds > 0 {
		hours = usage.Seconds / 3600
		estimate.Hours = &hours
	}
	return estimate, nil
}

var printerAliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._+@-]{0,199}$`)

// Resolve exact handles/emails only; never guess a surname from a first name.
// Scan display fields directly to avoid User's permission-loading hook.
func printerDisplayNames(db *gorm.DB, values []string) (map[string]string, error) {
	keys := map[string]bool{}
	emails := map[string]bool{}
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if !strings.Contains(key, "@") && strings.TrimSpace(value) != key {
			continue
		}
		if !printerAliasPattern.MatchString(key) {
			continue
		}
		keys[key] = true
		if strings.Contains(key, "@") {
			emails[key] = true
		} else {
			emails[key+"@umass.edu"] = true
		}
	}
	names := map[string]string{}
	if len(keys) == 0 {
		return names, nil
	}
	aliases, addresses := []string{}, []string{}
	for key := range keys {
		aliases = append(aliases, key)
	}
	for email := range emails {
		addresses = append(addresses, email)
	}
	var users []struct{ Email, Name string }
	if err := db.Model(&models.User{}).Unscoped().Select("email", "name").Where("LOWER(email) IN ?", addresses).Scan(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		if strings.TrimSpace(user.Name) == "" {
			continue
		}
		email := strings.ToLower(user.Email)
		names[email] = user.Name
		if strings.HasSuffix(email, "@umass.edu") {
			names[strings.TrimSuffix(email, "@umass.edu")] = user.Name
		}
	}
	var reviewed []models.PrinterIdentityAlias
	if err := db.Where("alias IN ?", aliases).Find(&reviewed).Error; err != nil {
		return nil, err
	}
	for _, alias := range reviewed {
		if strings.TrimSpace(alias.Name) != "" {
			names[alias.Alias] = alias.Name
		}
	}
	return names, nil
}

func applyPrinterDisplayNames(db *gorm.DB, events []models.PrinterHistoryEvent, history []models.PrinterHistoricalEntry, changes []fiber.Map) error {
	values := []string{}
	for _, event := range events {
		values = append(values, event.ActorName, event.Person)
	}
	for _, entry := range history {
		values = append(values, entry.Reporter, entry.Person)
	}
	for _, change := range changes {
		if name, ok := change["actorName"].(string); ok {
			values = append(values, name)
		}
	}
	names, err := printerDisplayNames(db, values)
	if err != nil {
		return err
	}
	display := func(value string) string {
		if name := names[strings.ToLower(strings.TrimSpace(value))]; name != "" {
			return name
		}
		return value
	}
	for i := range events {
		if events[i].ActorMethod == "ucard" {
			events[i].ActorName = display(events[i].ActorName)
		}
		events[i].Person = display(events[i].Person)
	}
	for i := range history {
		history[i].Reporter = display(history[i].Reporter)
		history[i].Person = display(history[i].Person)
	}
	for _, change := range changes {
		if name, ok := change["actorName"].(string); ok {
			change["actorName"] = display(name)
		}
	}
	return nil
}
