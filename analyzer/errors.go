package analyzer

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"goqkview/interfaces"
)

type ErrorAnalyzer struct {
	ipPattern   *regexp.Regexp
	portPattern *regexp.Regexp
}

func NewErrorAnalyzer() *ErrorAnalyzer {
	return &ErrorAnalyzer{
		ipPattern:   regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`),
		portPattern: regexp.MustCompile(`:\d{2,5}\b`),
	}
}

func (e *ErrorAnalyzer) Analyze(entries []interfaces.LogEntry) []TopError {
	groups := make(map[string]*errorGroup)

	for _, entry := range entries {
		if entry.Status != "ERROR" && entry.Status != "CRITICAL" && entry.Status != "SEVERE" {
			continue
		}

		normalized := e.normalizeMessage(entry.Line)

		if group, exists := groups[normalized]; exists {
			group.count++
			if entry.Timestamp.After(group.lastOccurred) {
				group.lastOccurred = entry.Timestamp
				group.representative = entry.Line
			}
		} else {
			groups[normalized] = &errorGroup{
				normalized:     normalized,
				representative: entry.Line,
				count:          1,
				lastOccurred:   entry.Timestamp,
			}
		}
	}

	result := make([]TopError, 0, len(groups))
	for _, group := range groups {
		result = append(result, TopError{
			Message:      e.extractErrorMessage(group.representative),
			Count:        group.count,
			LastOccurred: group.lastOccurred,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	if len(result) > 10 {
		result = result[:10]
	}

	return result
}

type errorGroup struct {
	normalized     string
	representative string
	count          int
	lastOccurred   time.Time
}

func (e *ErrorAnalyzer) normalizeMessage(line string) string {
	normalized := e.ipPattern.ReplaceAllString(line, "X.X.X.X")
	normalized = e.portPattern.ReplaceAllString(normalized, ":XXXX")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return strings.ToLower(normalized)
}

func (e *ErrorAnalyzer) extractErrorMessage(line string) string {
	prefixes := []string{": error:", ": warning:", ": critical:", "error:", "err:"}

	lower := strings.ToLower(line)
	for _, prefix := range prefixes {
		if idx := strings.Index(lower, prefix); idx != -1 {
			msg := strings.TrimSpace(line[idx+len(prefix):])
			if len(msg) > 150 {
				msg = msg[:150] + "..."
			}
			return msg
		}
	}

	if len(line) > 150 {
		return line[:150] + "..."
	}
	return line
}
