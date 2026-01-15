package analyzer

import (
	"sort"

	"goqkview/interfaces"
)

type TimelineBuilder struct{}

func NewTimelineBuilder() *TimelineBuilder {
	return &TimelineBuilder{}
}

func (t *TimelineBuilder) Build(entries []interfaces.LogEntry) []TimelineEntry {
	dateCounts := make(map[string]int)

	for _, entry := range entries {
		if entry.Status == "ERROR" || entry.Status == "CRITICAL" ||
			entry.Status == "SEVERE" || entry.Status == "WARNING" {
			date := entry.Timestamp.Format("2006-01-02")
			if date != "0001-01-01" {
				dateCounts[date]++
			}
		}
	}

	timeline := make([]TimelineEntry, 0, len(dateCounts))
	for date, count := range dateCounts {
		timeline = append(timeline, TimelineEntry{
			Date:   date,
			Errors: count,
		})
	}

	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Date < timeline[j].Date
	})

	return timeline
}
