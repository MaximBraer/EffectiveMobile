package postgres

import (
	"strconv"
	"time"
)

func parseMonth(s string) (time.Time, error) {
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func formatDate(date *time.Time) string {
	if date == nil {
		return ""
	}
	return date.Format("01-2006")
}

func int64ToStr(v int64) string {
	return strconv.FormatInt(v, 10)
}
