package helpers

import "time"

func FormatDateWithTime(date time.Time) string {
	// Paksa UTC agar konsisten dengan data di database
	return date.UTC().Format("02-01-2006 15:04")
}

func FormatDateDMY(date time.Time) string {
	return date.UTC().Format("02-01-2006")
}