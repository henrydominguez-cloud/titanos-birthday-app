// Package birthday calculates the days until a user's next birthday.
// It has no HTTP or DB dependencies, so it is easy to unit-test.
package birthday

import (
	"fmt"
	"time"
)

// Layout is the date format the API accepts and stores.
const Layout = "2006-01-02"

// Message builds the greeting for a date of birth, evaluated on the day `now`.
func Message(username string, dob time.Time, now time.Time) string {
	days := DaysUntilNextBirthday(dob, now)
	if days == 0 {
		return fmt.Sprintf("Hello, %s! Happy birthday!", username)
	}
	return fmt.Sprintf("Hello, %s! Your birthday is in %d day(s)", username, days)
}

// DaysUntilNextBirthday returns the days between now and the next birthday.
// It returns 0 when the birthday is today.
func DaysUntilNextBirthday(dob time.Time, now time.Time) int {
	today := dateOnly(now)
	next := nextBirthday(dob, today)
	return int(next.Sub(today).Hours() / 24)
}

func nextBirthday(dob, today time.Time) time.Time {
	b := birthdayInYear(dob, today.Year())
	if b.Before(today) {
		b = birthdayInYear(dob, today.Year()+1)
	}
	return b
}

func birthdayInYear(dob time.Time, year int) time.Time {
	m, d := dob.Month(), dob.Day()
	// Feb 29 birthdays fall on Mar 1 in non-leap years.
	if m == time.February && d == 29 && !isLeap(year) {
		return time.Date(year, time.March, 1, 0, 0, 0, 0, time.UTC)
	}
	return time.Date(year, m, d, 0, 0, 0, 0, time.UTC)
}

// dateOnly drops the time-of-day and works in UTC.
func dateOnly(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}
