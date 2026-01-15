package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"goqkview/analyzer"
)

type Writer struct {
	stdout bool
	path   string
}

func NewWriter(path string, stdout bool) *Writer {
	return &Writer{
		stdout: stdout,
		path:   path,
	}
}

func (w *Writer) Write(result *analyzer.AnalysisResult) error {
	output := w.toJSONFormat(result)

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("output: failed to marshal JSON: %w", err)
	}

	if w.stdout {
		_, err = os.Stdout.Write(data)
		if err != nil {
			return fmt.Errorf("output: failed to write to stdout: %w", err)
		}
		fmt.Println()
		return nil
	}

	if err := os.WriteFile(w.path, data, 0644); err != nil {
		return fmt.Errorf("output: failed to write file %s: %w", w.path, err)
	}

	return nil
}

func (w *Writer) WriteToWriter(result *analyzer.AnalysisResult, writer io.Writer) error {
	output := w.toJSONFormat(result)

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

type JSONOutput struct {
	Summary         analyzer.Summary              `json:"summary"`
	ErrorTimeline   []analyzer.TimelineEntry      `json:"errorTimeline"`
	SSLFindings     []analyzer.SSLFinding         `json:"sslFindings"`
	TopErrors       []TopErrorJSON                `json:"topErrors"`
	Recommendations []analyzer.Recommendation     `json:"recommendations"`
	VirtualServers  []analyzer.VirtualServerInfo  `json:"virtualServers"`
	EntryLogs       []analyzer.EntryLog           `json:"entryLogs"`
}

type TopErrorJSON struct {
	Message      string `json:"message"`
	Count        int    `json:"count"`
	LastOccurred string `json:"lastOccurred"`
}

func (w *Writer) toJSONFormat(result *analyzer.AnalysisResult) JSONOutput {
	topErrors := make([]TopErrorJSON, len(result.TopErrors))
	for i, e := range result.TopErrors {
		lastOccurred := e.LastOccurred.Format("2006-01-02T15:04:05Z")
		if e.LastOccurred.IsZero() {
			lastOccurred = ""
		}
		topErrors[i] = TopErrorJSON{
			Message:      e.Message,
			Count:        e.Count,
			LastOccurred: lastOccurred,
		}
	}

	return JSONOutput{
		Summary:         result.Summary,
		ErrorTimeline:   result.ErrorTimeline,
		SSLFindings:     result.SSLFindings,
		TopErrors:       topErrors,
		Recommendations: result.Recommendations,
		VirtualServers:  result.VirtualServers,
		EntryLogs:       result.EntryLogs,
	}
}
