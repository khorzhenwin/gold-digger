package ticker_price

import (
	"time"
)

// List of known US market holidays (non-exhaustive for example)
var usMarketHolidays = map[string]struct{}{
	"2025-01-01": {}, // New Year's Day
	"2025-01-20": {}, // Martin Luther King Jr. Day
	"2025-02-17": {}, // Presidents' Day
	"2025-03-17": {}, // St. Patrick's Day (optional, not market-closed)
	"2025-05-26": {}, // Memorial Day
	"2025-07-04": {}, // Independence Day
	"2025-09-01": {}, // Labor Day
	"2025-11-11": {}, // Veterans Day
	"2025-11-27": {}, // Thanksgiving Day
	"2025-12-25": {}, // Christmas Day
}

func IsTradingHours(t time.Time) bool {
	utc := t.UTC()
	dateStr := utc.Format("2006-01-02")

	// ✋ Skip weekends
	if utc.Weekday() == time.Saturday || utc.Weekday() == time.Sunday {
		return false
	}

	// ✋ Skip known market holidays
	if _, holiday := usMarketHolidays[dateStr]; holiday {
		return false
	}

	// ✅ Market open between 14:30 and 21:00 UTC
	openingHours := time.Date(utc.Year(), utc.Month(), utc.Day(), 14, 30, 0, 0, time.UTC)
	closingHours := time.Date(utc.Year(), utc.Month(), utc.Day(), 21, 0, 0, 0, time.UTC)

	return utc.After(openingHours) && utc.Before(closingHours)
}
