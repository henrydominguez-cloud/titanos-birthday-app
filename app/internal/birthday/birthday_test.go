package birthday

import (
	"testing"
	"time"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(Layout, s)
	if err != nil {
		t.Fatalf("bad date %q: %v", s, err)
	}
	return d
}

func TestMessage(t *testing.T) {
	cases := []struct {
		name string
		dob  string
		now  string
		want string
	}{
		{"birthday today", "1990-09-06", "2026-09-06", "Hello, jdoe! Happy birthday!"},
		{"in one day", "1990-09-07", "2026-09-06", "Hello, jdoe! Your birthday is in 1 day(s)"},
		{"in a few days", "1990-09-10", "2026-09-06", "Hello, jdoe! Your birthday is in 4 day(s)"},
		{"already passed this year", "1990-09-05", "2026-09-06", "Hello, jdoe! Your birthday is in 364 day(s)"},
		{"leap-year birthday on non-leap year", "2000-02-29", "2026-02-28", "Hello, jdoe! Your birthday is in 1 day(s)"},
		{"leap-year birthday on leap year", "2000-02-29", "2028-02-28", "Hello, jdoe! Your birthday is in 1 day(s)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Message("jdoe", mustDate(t, tc.dob), mustDate(t, tc.now))
			if got != tc.want {
				t.Errorf("Message() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDaysUntilNextBirthdayIgnoresTimeOfDay(t *testing.T) {
	dob := mustDate(t, "1990-09-10")
	now := time.Date(2026, 9, 6, 23, 59, 59, 0, time.UTC)
	if got := DaysUntilNextBirthday(dob, now); got != 4 {
		t.Errorf("DaysUntilNextBirthday() = %d, want 4", got)
	}
}
