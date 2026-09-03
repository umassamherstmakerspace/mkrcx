package leash_backend_api

import (
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	leash_auth "github.com/mkrcx/mkrcx/src/shared/authentication"
	"github.com/mkrcx/mkrcx/src/shared/models"
	"gorm.io/gorm"
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
	Visitors    int `json:"visitors"`
	Checkins    int `json:"checkins"`
	NewAccounts int `json:"new_accounts"`
}

type activityPoint struct {
	Start             string `json:"start"`
	Visitors          int    `json:"visitors"`
	Checkins          int    `json:"checkins"`
	NewAccounts       int    `json:"new_accounts"`
	CumulativeVisitors int    `json:"cumulative_visitors"`
}

type activityHeatCell struct {
	Weekday int `json:"weekday"`
	Hour    int `json:"hour"`
	Checkins int `json:"checkins"`
}

type activityCoverage struct {
	IdentifiedCheckins int     `json:"identified_checkins"`
	TotalCheckins      int     `json:"total_checkins"`
	IdentifiedPercent  float64 `json:"identified_percent"`
	FirstCheckin       string  `json:"first_checkin,omitempty"`
}

type activityResponse struct {
	Timezone string                       `json:"timezone"`
	Range    activityRange                `json:"range"`
	Today    activitySummary              `json:"today"`
	Week     activitySummary              `json:"week"`
	Selected activitySummary              `json:"selected"`
	Daily    []activityPoint              `json:"daily"`
	Weekly   []activityPoint              `json:"weekly"`
	Heatmap  []activityHeatCell           `json:"heatmap"`
	Coverage activityCoverage             `json:"coverage"`
}

type activityEvent struct {
	OccurredAt time.Time
	MemberUUID string
}

type activityAccount struct {
	CreatedAt time.Time
}

func localDayStart(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
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
		year := today.Year()
		if today.Month() < time.August {
			year--
		}
		return key, "This academic year", time.Date(year, time.August, 1, 0, 0, 0, 0, location), end
	default:
		month, day := time.August, 1
		if today.Month() >= time.January && today.Month() <= time.May {
			month, day = time.January, 1
		} else if today.Month() >= time.June && today.Month() < time.August {
			month, day = time.June, 1
		}
		return "semester", "This semester", time.Date(today.Year(), month, day, 0, 0, 0, 0, location), end
	}
}

func mondayStart(value time.Time, location *time.Location) time.Time {
	start := localDayStart(value, location)
	daysSinceMonday := (int(start.Weekday()) + 6) % 7
	return start.AddDate(0, 0, -daysSinceMonday)
}

func summaryFor(events []activityEvent, accounts []activityAccount, start, end time.Time) activitySummary {
	visitors := map[string]struct{}{}
	checkins := 0
	newAccounts := 0
	for _, event := range events {
		if !event.OccurredAt.Before(start) && event.OccurredAt.Before(end) {
			checkins++
			if event.MemberUUID != "" {
				visitors[event.MemberUUID] = struct{}{}
			}
		}
	}
	for _, account := range accounts {
		if !account.CreatedAt.Before(start) && account.CreatedAt.Before(end) {
			newAccounts++
		}
	}
	return activitySummary{Visitors: len(visitors), Checkins: checkins, NewAccounts: newAccounts}
}

func buildActivityResponse(db *gorm.DB, requested string, now time.Time, location *time.Location) (activityResponse, error) {
	key, label, rangeStart, rangeEnd := activityPreset(requested, now, location)
	todayStart := localDayStart(now, location)
	weekStart := mondayStart(now, location)
	queryStart := rangeStart
	if weekStart.Before(queryStart) {
		queryStart = weekStart
	}

	var events []activityEvent
	if err := db.Table("checkin_events").
		Select("occurred_at", "member_uuid").
		Where("occurred_at >= ? AND occurred_at < ?", queryStart.UTC(), rangeEnd.UTC()).
		Order("occurred_at ASC").Scan(&events).Error; err != nil {
		return activityResponse{}, err
	}

	var accounts []activityAccount
	if err := db.Unscoped().Table("users").
		Select("created_at").
		Where("created_at >= ? AND created_at < ? AND role <> ?", queryStart.UTC(), rangeEnd.UTC(), "service").
		Order("created_at ASC").Scan(&accounts).Error; err != nil {
		return activityResponse{}, err
	}

	response := activityResponse{
		Timezone: activityTimezone,
		Range: activityRange{
			Key: key, Label: label, Start: rangeStart.Format("2006-01-02"),
			End: rangeEnd.AddDate(0, 0, -1).Format("2006-01-02"),
		},
		Today:    summaryFor(events, accounts, todayStart, rangeEnd),
		Week:     summaryFor(events, accounts, weekStart, rangeEnd),
		Selected: summaryFor(events, accounts, rangeStart, rangeEnd),
		Daily:    []activityPoint{},
		Weekly:   []activityPoint{},
		Heatmap:  []activityHeatCell{},
	}

	dailyVisitors := map[string]map[string]struct{}{}
	dailyCheckins := map[string]int{}
	dailyAccounts := map[string]int{}
	weeklyVisitors := map[string]map[string]struct{}{}
	weeklyCheckins := map[string]int{}
	weeklyAccounts := map[string]int{}
	heat := map[[2]int]int{}
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
		heat[[2]int{int(local.Weekday()), local.Hour()}]++
		if event.MemberUUID != "" {
			identified++
			if dailyVisitors[dayKey] == nil {
				dailyVisitors[dayKey] = map[string]struct{}{}
			}
			if weeklyVisitors[weekKey] == nil {
				weeklyVisitors[weekKey] = map[string]struct{}{}
			}
			dailyVisitors[dayKey][event.MemberUUID] = struct{}{}
			weeklyVisitors[weekKey][event.MemberUUID] = struct{}{}
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

	cumulative := map[string]struct{}{}
	for day := rangeStart; day.Before(rangeEnd); day = day.AddDate(0, 0, 1) {
		dayKey := day.Format("2006-01-02")
		for visitor := range dailyVisitors[dayKey] {
			cumulative[visitor] = struct{}{}
		}
		response.Daily = append(response.Daily, activityPoint{
			Start: dayKey, Visitors: len(dailyVisitors[dayKey]), Checkins: dailyCheckins[dayKey],
			NewAccounts: dailyAccounts[dayKey], CumulativeVisitors: len(cumulative),
		})
	}

	for week := mondayStart(rangeStart, location); week.Before(rangeEnd); week = week.AddDate(0, 0, 7) {
		bucketStart := week
		if bucketStart.Before(rangeStart) {
			bucketStart = rangeStart
		}
		weekKey := bucketStart.Format("2006-01-02")
		response.Weekly = append(response.Weekly, activityPoint{
			Start: weekKey, Visitors: len(weeklyVisitors[weekKey]), Checkins: weeklyCheckins[weekKey],
			NewAccounts: weeklyAccounts[weekKey],
		})
	}

	keys := make([][2]int, 0, len(heat))
	for cell := range heat {
		keys = append(keys, cell)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] == keys[j][0] {
			return keys[i][1] < keys[j][1]
		}
		return keys[i][0] < keys[j][0]
	})
	for _, cell := range keys {
		response.Heatmap = append(response.Heatmap, activityHeatCell{Weekday: cell[0], Hour: cell[1], Checkins: heat[cell]})
	}
	response.Coverage = activityCoverage{
		IdentifiedCheckins: identified,
		TotalCheckins:      response.Selected.Checkins,
	}
	if response.Coverage.TotalCheckins > 0 {
		response.Coverage.IdentifiedPercent = float64(identified) * 100 / float64(response.Coverage.TotalCheckins)
		response.Coverage.FirstCheckin = firstCheckin.In(location).Format("2006-01-02")
	}
	return response, nil
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
	response, err := buildActivityResponse(leash_auth.GetDB(c), request.Range, time.Now(), location)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.JSON(response)
}

func registerActivityEndpoints(api fiber.Router) {
	api.Get("/activity", authorizeActivity, models.GetQueryMiddleware[activityRequest], getActivity)
}
