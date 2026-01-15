package parser

import (
	"regexp"
	"strings"
	"time"
)

var (
	// Matches: "Oct 14 13:00:00 2020" (month day time year)
	dateWithYearPattern = regexp.MustCompile(
		`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2})\s+(\d{2}:\d{2}:\d{2})\s+(\d{4})\b`)

	// Matches: "2023-10-24 13:00:00" (ISO format)
	isoDatePattern = regexp.MustCompile(
		`\b(\d{4})-(\d{2})-(\d{2})\s+(\d{2}:\d{2}:\d{2})\b`)

	// Matches: "2023-10-24T13:00:00" or "2023-10-24T13:00:00Z" (RFC3339-like)
	rfc3339Pattern = regexp.MustCompile(
		`\b(\d{4})-(\d{2})-(\d{2})T(\d{2}:\d{2}:\d{2})(?:Z|[+-]\d{2}:?\d{2})?\b`)

	// Matches: "Oct 14 13:00:00" (without year - common syslog format)
	dateWithoutYearPattern = regexp.MustCompile(
		`\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2})\s+(\d{2}:\d{2}:\d{2})\b`)

	// Status pattern (case insensitive)
	statusPattern = regexp.MustCompile(`(?i)\b(warning|error|severe|critical|notice)\b`)
)

// Date format layouts for time.Parse
const (
	layoutWithYear    = "Jan 2 15:04:05 2006"
	layoutISO         = "2006-01-02 15:04:05"
	layoutRFC3339     = "2006-01-02T15:04:05"
	layoutWithoutYear = "Jan 2 15:04:05"
)

type DateParseOptions struct {
	ReferenceTime time.Time
	DefaultYear int
}

func ParseDate(line string, opts DateParseOptions) (time.Time, bool) {
	if matches := dateWithYearPattern.FindStringSubmatch(line); len(matches) >= 5 {
		dateStr := matches[1] + " " + matches[2] + " " + matches[3] + " " + matches[4]
		if t, err := time.Parse(layoutWithYear, dateStr); err == nil {
			return t, true
		}
	}

	if matches := isoDatePattern.FindStringSubmatch(line); len(matches) >= 5 {
		dateStr := matches[1] + "-" + matches[2] + "-" + matches[3] + " " + matches[4]
		if t, err := time.Parse(layoutISO, dateStr); err == nil {
			return t, true
		}
	}

	if matches := rfc3339Pattern.FindStringSubmatch(line); len(matches) >= 5 {
		dateStr := matches[1] + "-" + matches[2] + "-" + matches[3] + "T" + matches[4]
		if t, err := time.Parse(layoutRFC3339, dateStr); err == nil {
			return t, true
		}
	}

	if matches := dateWithoutYearPattern.FindStringSubmatch(line); len(matches) >= 4 {
		year := inferYear(opts)
		dateStr := matches[1] + " " + matches[2] + " " + matches[3]

		if t, err := time.Parse(layoutWithoutYear, dateStr); err == nil {
			t = time.Date(year, t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local)
			now := opts.ReferenceTime
			if now.IsZero() {
				now = time.Now()
			}
			if t.After(now) {
				t = t.AddDate(-1, 0, 0)
			}
			return t, true
		}
	}

	return time.Time{}, false
}

func inferYear(opts DateParseOptions) int {
	if opts.DefaultYear != 0 {
		return opts.DefaultYear
	}
	if !opts.ReferenceTime.IsZero() {
		return opts.ReferenceTime.Year()
	}
	return time.Now().Year()
}

func ParseStatus(line string) (string, bool) {
	if matches := statusPattern.FindStringSubmatch(line); len(matches) > 0 {
		return strings.ToUpper(matches[1]), true
	}
	return "", false
}

func HasStatus(line string) bool {
	return statusPattern.MatchString(line)
}
