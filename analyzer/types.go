package analyzer

import "time"

type AnalysisResult struct {
	Summary         Summary             `json:"summary"`
	ErrorTimeline   []TimelineEntry     `json:"errorTimeline"`
	SSLFindings     []SSLFinding        `json:"sslFindings"`
	TopErrors       []TopError          `json:"topErrors"`
	Recommendations []Recommendation    `json:"recommendations"`
	VirtualServers  []VirtualServerInfo `json:"virtualServers"`
	EntryLogs       []EntryLog          `json:"entryLogs"`
}

type EntryLog struct {
	Message string `json:"message"`
	Level   string `json:"level"`
	Date    string `json:"date"` // ISO8601 format
}

type Summary struct {
	Critical          int `json:"critical"`
	Warning           int `json:"warning"`
	Healthy           int `json:"healthy"`
	CertsExpiringSoon int `json:"certsExpiringSoon"`
}

type TimelineEntry struct {
	Date   string `json:"date"` // YYYY-MM-DD format
	Errors int    `json:"errors"`
}

type SSLFinding struct {
	Severity   string   `json:"severity"`   // critical, warning, info
	Type       string   `json:"type"`       // certificate, cipher, configuration
	Message    string   `json:"message"`
	Detail     string   `json:"detail"`
	AffectedVS []string `json:"affectedVS"` // Virtual servers affected
}

type TopError struct {
	Message      string    `json:"message"`
	Count        int       `json:"count"`
	LastOccurred time.Time `json:"-"` // Used internally, serialized differently
}

type Recommendation struct {
	Priority    string `json:"priority"` // critical, high, medium, low
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}
