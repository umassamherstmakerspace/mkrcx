package leash_backend_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	activityReadPermission = "leash.activity:read"
	activityTimezone       = "America/New_York"
)

type activityRequest struct {
	Range string `query:"range" validate:"omitempty,oneof=semester academic_year 30_days"`
}

type activityRange struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type activitySummary struct {
	Visitors           int `json:"visitors"`
	UnlinkedCardHolders int `json:"unlinked_card_holders"`
	Checkins           int `json:"checkins"`
	NewAccounts        int `json:"new_accounts"`
	NewlyLinkedCards   int `json:"newly_linked_cards"`
}

type activityPoint struct {
	Start               string `json:"start"`
	Visitors            int    `json:"visitors"`
	UnlinkedCardHolders int    `json:"unlinked_card_holders"`
	Checkins            int    `json:"checkins"`
	NewAccounts         int    `json:"new_accounts"`
	NewlyLinkedCards    int    `json:"newly_linked_cards"`
	CumulativeVisitors  int    `json:"cumulative_visitors"`
	UnknownCards        int    `json:"unknown_cards"`
}

// activityPulse is one fixed look-back window for the at-a-glance page.
// People counts one person per day: members linked at the tap, members whose
// card was not linked yet, and distinct unknown cards.
type activityPulse struct {
	Key              string  `json:"key"`
	Label            string  `json:"label"`
	OpenDays         int     `json:"open_days"`
	People           int     `json:"people"`
	AvgDailyPeople   float64 `json:"avg_daily_people"`
	NotLinkedPeople  int     `json:"not_linked_people"`
	NotLinkedPercent float64 `json:"not_linked_percent"`
	Checkins         int     `json:"checkins"`
	NewAccounts      int     `json:"new_accounts"`
	NewlyLinkedCards int     `json:"newly_linked_cards"`
}

type activityHeatCell struct {
	Weekday int `json:"weekday"`
	Hour    int `json:"hour"`
	Members int `json:"members"`
	Taps    int `json:"taps"`
}

type activityCoverage struct {
	IdentifiedCheckins int     `json:"identified_checkins"`
	TotalCheckins      int     `json:"total_checkins"`
	IdentifiedPercent  float64 `json:"identified_percent"`
	FirstCheckin       string  `json:"first_checkin,omitempty"`
	FirstCardLink      string  `json:"first_card_link,omitempty"`
}

type activityAcademicYear struct {
	Label            string `json:"label"`
	Start            string `json:"start"`
	End              string `json:"end"`
	NewAccounts      int    `json:"new_accounts"`
	NewlyLinkedCards int    `json:"newly_linked_cards"`
	Current          bool   `json:"current"`
}

type activityResponse struct {
	Timezone     string                 `json:"timezone"`
	SnapshotAt   string                 `json:"snapshot_at,omitempty"`
	Range        activityRange          `json:"range"`
	Today        activitySummary        `json:"today"`
	Week         activitySummary        `json:"week"`
	Selected     activitySummary        `json:"selected"`
	Daily        []activityPoint        `json:"daily"`
	Weekly       []activityPoint        `json:"weekly"`
	Heatmap      []activityHeatCell     `json:"heatmap"`
	HeatmapOpenDays [7]int              `json:"heatmap_open_days"`
	Pulse        []activityPulse        `json:"pulse"`
	AcademicYears []activityAcademicYear `json:"academic_years"`
	Coverage     activityCoverage       `json:"coverage"`
}

func loadActivitySnapshot(path, requested string) (activityResponse, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return activityResponse{}, err
	}
	responses := map[string]activityResponse{}
	if err := json.Unmarshal(contents, &responses); err != nil {
		return activityResponse{}, err
	}
	key := strings.TrimSpace(requested)
	if key == "" {
		key = "semester"
	}
	response, ok := responses[key]
	if !ok {
		return activityResponse{}, os.ErrNotExist
	}
	return response, nil
}

type activityEvent struct {
	OccurredAt  time.Time
	MemberUUID  string
	LinkedAtTap bool
}

type activityAccount struct {
	CreatedAt time.Time
}

type activityCardLink struct {
	UserID    uint
	CreatedAt time.Time
}

func localDayStart(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}

func academicYearStart(value time.Time, location *time.Location) time.Time {
	today := localDayStart(value, location)
	year := today.Year()
	if today.Month() < time.August {
		year--
	}
	return time.Date(year, time.August, 1, 0, 0, 0, 0, location)
}

func academicYearLabel(start time.Time) string {
	return fmt.Sprintf("%d–%02d", start.Year(), (start.Year()+1)%100)
}

func activityPreset(requested string, now time.Time, location *time.Location) (string, string, time.Time, time.Time) {
	key := strings.TrimSpace(requested)
	if key == "" {
		key = "semester"
	}
	today := localDayStart(now, location)
	end := today.AddDate(0, 0, 1)

	switch key {
	case "30_days":
		return key, "Past 30 days", today.AddDate(0, 0, -29), end
	case "academic_year":
		start := academicYearStart(now, location)
		return key, academicYearLabel(start) + " academic year", start, end
	default:
		month, day, season := time.August, 1, "Fall"
		if today.Month() >= time.January && today.Month() <= time.May {
			month, day, season = time.January, 1, "Spring"
		} else if today.Month() >= time.June && today.Month() < time.August {
			month, day, season = time.June, 1, "Summer"
		}
		return "semester", fmt.Sprintf("%s %d", season, today.Year()), time.Date(today.Year(), month, day, 0, 0, 0, 0, location), end
	}
}

func mondayStart(value time.Time, location *time.Location) time.Time {
	start := localDayStart(value, location)
	daysSinceMonday := (int(start.Weekday()) + 6) % 7
	return start.AddDate(0, 0, -daysSinceMonday)
}

func summaryFor(events []activityEvent, accounts []activityAccount, links []activityCardLink, start, end time.Time) activitySummary {
	visitors := map[string]struct{}{}
	unlinked := map[string]struct{}{}
	checkins := 0
	newAccounts := 0
	newlyLinked := map[uint]struct{}{}
	for _, event := range events {
		if !event.OccurredAt.Before(start) && event.OccurredAt.Before(end) {
			checkins++
			if event.MemberUUID != "" {
				if event.LinkedAtTap {
					visitors[event.MemberUUID] = struct{}{}
				} else {
					unlinked[event.MemberUUID] = struct{}{}
				}
			}
		}
	}
	for _, account := range accounts {
		if !account.CreatedAt.Before(start) && account.CreatedAt.Before(end) {
			newAccounts++
		}
	}
	for _, link := range links {
		if !link.CreatedAt.Before(start) && link.CreatedAt.Before(end) {
			newlyLinked[link.UserID] = struct{}{}
		}
	}
	return activitySummary{
		Visitors: len(visitors), UnlinkedCardHolders: len(unlinked), Checkins: checkins,
		NewAccounts: newAccounts, NewlyLinkedCards: len(newlyLinked),
	}
}

// pulseFor counts one person per local day in [start, end). A day is open
// when it has at least one tap, so closed days do not pull the average down.
func pulseFor(key, label string, events []activityEvent, accounts []activityAccount, links []activityCardLink, unknownCards map[string]int, start, end time.Time, location *time.Location) activityPulse {
	linked := map[string]map[string]struct{}{}
	notLinked := map[string]map[string]struct{}{}
	openDays := map[string]struct{}{}
	for _, event := range events {
		if event.OccurredAt.Before(start) || !event.OccurredAt.Before(end) {
			continue
		}
		dayKey := event.OccurredAt.In(location).Format("2006-01-02")
		openDays[dayKey] = struct{}{}
		if event.MemberUUID == "" {
			continue
		}
		target := notLinked
		if event.LinkedAtTap {
			target = linked
		}
		if target[dayKey] == nil {
			target[dayKey] = map[string]struct{}{}
		}
		target[dayKey][event.MemberUUID] = struct{}{}
	}
	summary := summaryFor(events, accounts, links, start, end)
	pulse := activityPulse{
		Key: key, Label: label, OpenDays: len(openDays), Checkins: summary.Checkins,
		NewAccounts: summary.NewAccounts, NewlyLinkedCards: summary.NewlyLinkedCards,
	}
	for dayKey := range openDays {
		// A member who tapped both before and after linking on one day is one person.
		unlinkedOnly := 0
		for member := range notLinked[dayKey] {
			if _, both := linked[dayKey][member]; !both {
				unlinkedOnly++
			}
		}
		pulse.NotLinkedPeople += unlinkedOnly + unknownCards[dayKey]
		pulse.People += len(linked[dayKey]) + unlinkedOnly + unknownCards[dayKey]
	}
	if pulse.OpenDays > 0 {
		pulse.AvgDailyPeople = float64(pulse.People) / float64(pulse.OpenDays)
	}
	if pulse.People > 0 {
		pulse.NotLinkedPercent = float64(pulse.NotLinkedPeople) * 100 / float64(pulse.People)
	}
	return pulse
}

func BuildActivityResponse(db *gorm.DB, requested string, now time.Time, location *time.Location) (activityResponse, error) {
	key, label, rangeStart, rangeEnd := activityPreset(requested, now, location)
	todayStart := localDayStart(now, location)
	weekStart := mondayStart(now, location)
	eventQueryStart := rangeStart
	if weekStart.Before(eventQueryStart) {
		eventQueryStart = weekStart
	}
	pulseStart := todayStart.AddDate(0, 0, -29)
	if pulseStart.Before(eventQueryStart) {
		eventQueryStart = pulseStart
	}
	currentAcademicYear := academicYearStart(now, location)
	comparisonStart := currentAcademicYear.AddDate(-2, 0, 0)

	var events []activityEvent
	if err := db.Table("checkin_events").
		Select("occurred_at", "member_uuid", "linked_at_tap").
		Where("occurred_at >= ? AND occurred_at < ?", eventQueryStart.UTC(), rangeEnd.UTC()).
		Order("occurred_at ASC").Scan(&events).Error; err != nil {
		return activityResponse{}, err
	}

	var accounts []activityAccount
	if err := db.Unscoped().Table("users").
		Select("created_at").
		Where("created_at >= ? AND created_at < ? AND role <> ?", comparisonStart.UTC(), rangeEnd.UTC(), "service").
		Order("created_at ASC").Scan(&accounts).Error; err != nil {
		return activityResponse{}, err
	}

	var linkUpdates []activityCardLink
	if err := db.Unscoped().Table("user_updates").
		Select("user_id", "created_at").
		Where("field = ? AND old_value = ? AND new_value <> ?", "card_id", "", "").
		Where("created_at < ?", rangeEnd.UTC()).
		Order("created_at ASC").Scan(&linkUpdates).Error; err != nil {
		return activityResponse{}, err
	}
	links := make([]activityCardLink, 0, len(linkUpdates))
	linkedUsers := map[uint]struct{}{}
	for _, link := range linkUpdates {
		if _, exists := linkedUsers[link.UserID]; exists {
			continue
		}
		linkedUsers[link.UserID] = struct{}{}
		if !link.CreatedAt.Before(comparisonStart.UTC()) {
			links = append(links, link)
		}
	}

	unknownCards := map[string]int{}
	if db.Migrator().HasTable(&models.CheckinUnknownDaily{}) {
		var unknownRows []models.CheckinUnknownDaily
		if err := db.Where("day >= ? AND day < ?", eventQueryStart.Format("2006-01-02"), rangeEnd.Format("2006-01-02")).
			Find(&unknownRows).Error; err != nil {
			return activityResponse{}, err
		}
		for _, row := range unknownRows {
			unknownCards[row.Day] = row.DistinctCards
		}
	}

	response := activityResponse{
		Timezone: activityTimezone,
		Range: activityRange{
			Key: key, Label: label, Start: rangeStart.Format("2006-01-02"),
			End: rangeEnd.AddDate(0, 0, -1).Format("2006-01-02"),
		},
		Today:         summaryFor(events, accounts, links, todayStart, rangeEnd),
		Week:          summaryFor(events, accounts, links, weekStart, rangeEnd),
		Selected:      summaryFor(events, accounts, links, rangeStart, rangeEnd),
		Daily:         []activityPoint{},
		Weekly:        []activityPoint{},
		Heatmap:       []activityHeatCell{},
		Pulse:         []activityPulse{},
		AcademicYears: []activityAcademicYear{},
	}

	dailyVisitors := map[string]map[string]struct{}{}
	dailyUnlinked := map[string]map[string]struct{}{}
	dailyCheckins := map[string]int{}
	dailyAccounts := map[string]int{}
	dailyLinks := map[string]map[uint]struct{}{}
	weeklyVisitors := map[string]map[string]struct{}{}
	weeklyUnlinked := map[string]map[string]struct{}{}
	weeklyCheckins := map[string]int{}
	weeklyAccounts := map[string]int{}
	weeklyLinks := map[string]map[uint]struct{}{}
	heat := map[[2]int]map[string]struct{}{}
	heatTaps := map[[2]int]int{}
	heatDays := map[string]int{}
	identified := 0
	var firstCheckin time.Time
	for _, event := range events {
		if event.OccurredAt.Before(rangeStart) || !event.OccurredAt.Before(rangeEnd) {
			continue
		}
		local := event.OccurredAt.In(location)
		dayKey := local.Format("2006-01-02")
		week := mondayStart(local, location)
		if week.Before(rangeStart) {
			week = rangeStart
		}
		weekKey := week.Format("2006-01-02")
		dailyCheckins[dayKey]++
		weeklyCheckins[weekKey]++
		heatKey := [2]int{int(local.Weekday()), local.Hour()}
		heatTaps[heatKey]++
		heatDays[dayKey] = int(local.Weekday())
		if event.MemberUUID != "" {
			identified++
			if heat[heatKey] == nil {
				heat[heatKey] = map[string]struct{}{}
			}
			heat[heatKey][event.MemberUUID] = struct{}{}
			if event.LinkedAtTap {
				if dailyVisitors[dayKey] == nil {
					dailyVisitors[dayKey] = map[string]struct{}{}
				}
				if weeklyVisitors[weekKey] == nil {
					weeklyVisitors[weekKey] = map[string]struct{}{}
				}
				dailyVisitors[dayKey][event.MemberUUID] = struct{}{}
				weeklyVisitors[weekKey][event.MemberUUID] = struct{}{}
			} else {
				if dailyUnlinked[dayKey] == nil {
					dailyUnlinked[dayKey] = map[string]struct{}{}
				}
				if weeklyUnlinked[weekKey] == nil {
					weeklyUnlinked[weekKey] = map[string]struct{}{}
				}
				dailyUnlinked[dayKey][event.MemberUUID] = struct{}{}
				weeklyUnlinked[weekKey][event.MemberUUID] = struct{}{}
			}
		}
		if firstCheckin.IsZero() || event.OccurredAt.Before(firstCheckin) {
			firstCheckin = event.OccurredAt
		}
	}
	for _, account := range accounts {
		if account.CreatedAt.Before(rangeStart) || !account.CreatedAt.Before(rangeEnd) {
			continue
		}
		local := account.CreatedAt.In(location)
		dayKey := local.Format("2006-01-02")
		week := mondayStart(local, location)
		if week.Before(rangeStart) {
			week = rangeStart
		}
		dailyAccounts[dayKey]++
		weeklyAccounts[week.Format("2006-01-02")]++
	}
	for _, link := range links {
		if link.CreatedAt.Before(rangeStart) || !link.CreatedAt.Before(rangeEnd) {
			continue
		}
		local := link.CreatedAt.In(location)
		dayKey := local.Format("2006-01-02")
		week := mondayStart(local, location)
		if week.Before(rangeStart) {
			week = rangeStart
		}
		weekKey := week.Format("2006-01-02")
		if dailyLinks[dayKey] == nil {
			dailyLinks[dayKey] = map[uint]struct{}{}
		}
		if weeklyLinks[weekKey] == nil {
			weeklyLinks[weekKey] = map[uint]struct{}{}
		}
		dailyLinks[dayKey][link.UserID] = struct{}{}
		weeklyLinks[weekKey][link.UserID] = struct{}{}
	}

	cumulative := map[string]struct{}{}
	for day := rangeStart; day.Before(rangeEnd); day = day.AddDate(0, 0, 1) {
		dayKey := day.Format("2006-01-02")
		for visitor := range dailyVisitors[dayKey] {
			cumulative[visitor] = struct{}{}
		}
		response.Daily = append(response.Daily, activityPoint{
			Start: dayKey, Visitors: len(dailyVisitors[dayKey]), UnlinkedCardHolders: len(dailyUnlinked[dayKey]),
			Checkins: dailyCheckins[dayKey], NewAccounts: dailyAccounts[dayKey],
			NewlyLinkedCards: len(dailyLinks[dayKey]), CumulativeVisitors: len(cumulative),
			UnknownCards: unknownCards[dayKey],
		})
	}

	for week := mondayStart(rangeStart, location); week.Before(rangeEnd); week = week.AddDate(0, 0, 7) {
		bucketStart := week
		if bucketStart.Before(rangeStart) {
			bucketStart = rangeStart
		}
		weekKey := bucketStart.Format("2006-01-02")
		response.Weekly = append(response.Weekly, activityPoint{
			Start: weekKey, Visitors: len(weeklyVisitors[weekKey]), UnlinkedCardHolders: len(weeklyUnlinked[weekKey]),
			Checkins: weeklyCheckins[weekKey], NewAccounts: weeklyAccounts[weekKey],
			NewlyLinkedCards: len(weeklyLinks[weekKey]),
		})
	}

	keys := make([][2]int, 0, len(heatTaps))
	for cell := range heatTaps {
		keys = append(keys, cell)
	}
	for _, weekday := range heatDays {
		response.HeatmapOpenDays[weekday]++
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] == keys[j][0] {
			return keys[i][1] < keys[j][1]
		}
		return keys[i][0] < keys[j][0]
	})
	for _, cell := range keys {
		response.Heatmap = append(response.Heatmap, activityHeatCell{Weekday: cell[0], Hour: cell[1], Members: len(heat[cell]), Taps: heatTaps[cell]})
	}

	for _, window := range []struct {
		key, label string
		days       int
	}{{"today", "Today", 1}, {"7_days", "Past 7 days", 7}, {"30_days", "Past 30 days", 30}} {
		start := todayStart.AddDate(0, 0, -(window.days - 1))
		response.Pulse = append(response.Pulse, pulseFor(window.key, window.label, events, accounts, links, unknownCards, start, rangeEnd, location))
	}

	for index := 0; index < 3; index++ {
		start := comparisonStart.AddDate(index, 0, 0)
		end := start.AddDate(1, 0, 0)
		current := start.Equal(currentAcademicYear)
		if end.After(rangeEnd) {
			end = rangeEnd
		}
		summary := summaryFor(nil, accounts, links, start, end)
		response.AcademicYears = append(response.AcademicYears, activityAcademicYear{
			Label: academicYearLabel(start), Start: start.Format("2006-01-02"),
			End: end.AddDate(0, 0, -1).Format("2006-01-02"), NewAccounts: summary.NewAccounts,
			NewlyLinkedCards: summary.NewlyLinkedCards, Current: current,
		})
	}
	response.Coverage = activityCoverage{
		IdentifiedCheckins: identified,
		TotalCheckins:      response.Selected.Checkins,
	}
	if response.Coverage.TotalCheckins > 0 {
		response.Coverage.IdentifiedPercent = float64(identified) * 100 / float64(response.Coverage.TotalCheckins)
		response.Coverage.FirstCheckin = firstCheckin.In(location).Format("2006-01-02")
	}
	if len(links) > 0 {
		response.Coverage.FirstCardLink = links[0].CreatedAt.In(location).Format("2006-01-02")
	}
	return response, nil
}

// recordUnknownCardDailyCounts saves how many distinct unknown cards tapped on
// each local day, while the seven-day card fingerprints still exist. Only the
// count is kept. A day is rewritten only while it lies wholly inside the
// retention window, so a count can fall when a card is later linked (that
// person then counts as a member) but never because old rows were purged.
func recordUnknownCardDailyCounts(db *gorm.DB, now time.Time, location *time.Location) (int, error) {
	var feed models.Feed
	if result := db.Where("name = ?", checkinFeedName).First(&feed); errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return 0, nil
	} else if result.Error != nil {
		return 0, result.Error
	}
	cutoff := now.UTC().Add(-checkinFeedRetention)
	var pending []struct {
		CreatedAt              time.Time
		PendingCardFingerprint string
	}
	if err := db.Model(&models.FeedMessage{}).
		Select("created_at", "pending_card_fingerprint").
		Where("feed_id = ? AND user_id = 0 AND pending_card_fingerprint IS NOT NULL AND created_at >= ?", feed.ID, cutoff).
		Scan(&pending).Error; err != nil {
		return 0, err
	}
	cards := map[string]map[string]struct{}{}
	for _, item := range pending {
		dayKey := item.CreatedAt.In(location).Format("2006-01-02")
		if cards[dayKey] == nil {
			cards[dayKey] = map[string]struct{}{}
		}
		cards[dayKey][item.PendingCardFingerprint] = struct{}{}
	}

	written := 0
	today := localDayStart(now, location)
	for day := today; !day.Before(cutoff); day = day.AddDate(0, 0, -1) {
		dayKey := day.Format("2006-01-02")
		row := models.CheckinUnknownDaily{Day: dayKey, DistinctCards: len(cards[dayKey]), UpdatedAt: now.UTC()}
		if row.DistinctCards == 0 {
			result := db.Model(&models.CheckinUnknownDaily{}).Where("day = ?", dayKey).
				Updates(map[string]interface{}{"distinct_cards": 0, "updated_at": row.UpdatedAt})
			if result.Error != nil {
				return written, result.Error
			}
			continue
		}
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "day"}},
			DoUpdates: clause.AssignmentColumns([]string{"distinct_cards", "updated_at"}),
		}).Create(&row).Error; err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

func authorizeActivity(c *fiber.Ctx) error {
	authentication := leash_auth.GetAuthentication(c)
	if !authentication.IsUser() || authentication.Authorize(activityReadPermission) != nil {
		return fiber.ErrForbidden
	}
	return c.Next()
}

func getActivity(c *fiber.Ctx) error {
	location, err := time.LoadLocation(activityTimezone)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	request := c.Locals("query").(activityRequest)
	if snapshotPath := strings.TrimSpace(os.Getenv("ACTIVITY_SNAPSHOT_FILE")); snapshotPath != "" {
		response, err := loadActivitySnapshot(snapshotPath, request.Range)
		if err != nil {
			return fiber.ErrInternalServerError
		}
		c.Set(fiber.HeaderCacheControl, "no-store")
		return c.JSON(response)
	}
	response, err := BuildActivityResponse(leash_auth.GetDB(c), request.Range, time.Now(), location)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(response)
}

func registerActivityEndpoints(api fiber.Router) {
	api.Get("/activity", authorizeActivity, models.GetQueryMiddleware[activityRequest], getActivity)
}
