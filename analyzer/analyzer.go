package analyzer

import (
	"goqkview/interfaces"
	"goqkview/parser"
)

type Analyzer struct {
	sslAnalyzer     *SSLAnalyzer
	errorAnalyzer   *ErrorAnalyzer
	timelineBuilder *TimelineBuilder
	recommender     *Recommender
	vsAnalyzer      *VirtualServerAnalyzer
}

func New() *Analyzer {
	return &Analyzer{
		sslAnalyzer:     NewSSLAnalyzer(),
		errorAnalyzer:   NewErrorAnalyzer(),
		timelineBuilder: NewTimelineBuilder(),
		recommender:     NewRecommender(),
		vsAnalyzer:      NewVirtualServerAnalyzer(),
	}
}

func (a *Analyzer) Analyze(entries []interfaces.LogEntry, bigipConfig *parser.BigIPConfig) (*AnalysisResult, error) {
	result := &AnalysisResult{}

	result.ErrorTimeline = a.timelineBuilder.Build(entries)
	result.SSLFindings = a.sslAnalyzer.Analyze(entries)
	result.TopErrors = a.errorAnalyzer.Analyze(entries)
	result.VirtualServers = a.vsAnalyzer.Analyze(bigipConfig, entries)
	result.Summary = a.buildSummary(result.VirtualServers, result.SSLFindings)
	result.EntryLogs = a.convertToEntryLogs(entries)
	result.Recommendations = a.recommender.Generate(
		result.Summary,
		result.SSLFindings,
		result.TopErrors,
	)

	return result, nil
}

func (a *Analyzer) buildSummary(virtualServers []VirtualServerInfo, sslFindings []SSLFinding) Summary {
	summary := Summary{}

	for _, vs := range virtualServers {
		switch vs.Status {
		case "critical":
			summary.Critical++
		case "warning":
			summary.Warning++
		case "healthy":
			summary.Healthy++
		}
	}

	for _, finding := range sslFindings {
		if finding.Type == "certificate" &&
			(finding.Severity == "critical" || finding.Severity == "warning") {
			summary.CertsExpiringSoon++
		}
	}

	return summary
}

func (a *Analyzer) convertToEntryLogs(entries []interfaces.LogEntry) []EntryLog {
	logs := make([]EntryLog, len(entries))
	for i, e := range entries {
		date := ""
		if !e.Timestamp.IsZero() {
			date = e.Timestamp.Format("2006-01-02T15:04:05Z")
		}
		logs[i] = EntryLog{
			Message: e.Line,
			Level:   e.Status,
			Date:    date,
		}
	}
	return logs
}
