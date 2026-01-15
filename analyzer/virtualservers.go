package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"goqkview/interfaces"
	"goqkview/parser"
)

type VirtualServerAnalyzer struct{}

func NewVirtualServerAnalyzer() *VirtualServerAnalyzer {
	return &VirtualServerAnalyzer{}
}

type VirtualServerInfo struct {
	Name          string  `json:"name"`
	Pool          string  `json:"pool"`
	Status        string  `json:"status"`        // healthy, warning, critical
	ActiveMembers string  `json:"activeMembers"` // "X/Y" format
	LastError     *string `json:"lastError"`     // null or "Error message - YYYY-MM-DD HH:MM:SS"
}

func (v *VirtualServerAnalyzer) Analyze(config *parser.BigIPConfig, entries []interfaces.LogEntry) []VirtualServerInfo {
	if config == nil {
		return []VirtualServerInfo{}
	}

	results := []VirtualServerInfo{}

	vsNames := make([]string, 0, len(config.VirtualServers))
	for name := range config.VirtualServers {
		vsNames = append(vsNames, name)
	}
	sort.Strings(vsNames)

	for _, vsName := range vsNames {
		vs := config.VirtualServers[vsName]
		info := VirtualServerInfo{
			Name: vs.Name,
			Pool: vs.Pool,
		}

		if vs.Disabled {
			info.Status = "critical"
			info.ActiveMembers = "0/0"
		} else if pool, ok := config.Pools[vs.Pool]; ok {
			active := pool.GetActiveMembers()
			total := pool.GetTotalMembers()
			info.ActiveMembers = fmt.Sprintf("%d/%d", active, total)

			if total == 0 {
				info.Status = "critical"
			} else if active == 0 {
				info.Status = "critical"
			} else if active < total {
				info.Status = "warning"
			} else {
				info.Status = "healthy"
			}
		} else {
			info.Status = "warning"
			info.ActiveMembers = "0/0"
		}

		info.LastError = v.findLastError(vs.Name, vs.Pool, entries)

		results = append(results, info)
	}

	return results
}

func (v *VirtualServerAnalyzer) findLastError(vsName, poolName string, entries []interfaces.LogEntry) *string {
	sortedEntries := make([]interfaces.LogEntry, len(entries))
	copy(sortedEntries, entries)
	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].Timestamp.After(sortedEntries[j].Timestamp)
	})

	for _, entry := range sortedEntries {
		if entry.Status != "ERROR" && entry.Status != "CRITICAL" && entry.Status != "SEVERE" {
			continue
		}

		lineLower := strings.ToLower(entry.Line)
		vsNameLower := strings.ToLower(vsName)
		poolNameLower := strings.ToLower(poolName)

		if strings.Contains(lineLower, vsNameLower) || strings.Contains(lineLower, poolNameLower) {
			errorMsg := v.extractErrorMessage(entry.Line)
			timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
			result := fmt.Sprintf("%s - %s", errorMsg, timestamp)
			return &result
		}
	}

	return nil
}

func (v *VirtualServerAnalyzer) extractErrorMessage(line string) string {
	maxLen := 100
	if len(line) > maxLen {
		line = line[:maxLen] + "..."
	}

	parts := strings.SplitN(line, ": ", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}

	return strings.TrimSpace(line)
}
