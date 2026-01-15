package analyzer

import (
	"fmt"
	"strings"
)

type Recommender struct{}

func NewRecommender() *Recommender {
	return &Recommender{}
}

func (r *Recommender) Generate(summary Summary, sslFindings []SSLFinding, topErrors []TopError) []Recommendation {
	recommendations := []Recommendation{}

	for _, finding := range sslFindings {
		rec := r.sslRecommendation(finding)
		if rec.Title != "" {
			recommendations = append(recommendations, rec)
		}
	}

	for _, err := range topErrors {
		rec := r.errorRecommendation(err)
		if rec.Title != "" {
			recommendations = append(recommendations, rec)
		}
	}

	if summary.Critical > 10 {
		recommendations = append(recommendations, Recommendation{
			Priority:    "critical",
			Title:       "High number of critical events",
			Description: fmt.Sprintf("%d critical events detected. Immediate investigation required.", summary.Critical),
			Impact:      "System stability and security may be compromised",
		})
	}

	if summary.CertsExpiringSoon > 0 {
		recommendations = append(recommendations, Recommendation{
			Priority:    "critical",
			Title:       "Certificate renewal required",
			Description: fmt.Sprintf("%d certificate(s) expiring soon. Plan renewal immediately.", summary.CertsExpiringSoon),
			Impact:      "Service interruption for HTTPS traffic",
		})
	}

	r.sortByPriority(recommendations)

	recommendations = r.deduplicate(recommendations)

	if len(recommendations) > 10 {
		recommendations = recommendations[:10]
	}

	return recommendations
}

func (r *Recommender) sslRecommendation(finding SSLFinding) Recommendation {
	switch finding.Type {
	case "certificate":
		return Recommendation{
			Priority:    finding.Severity,
			Title:       "SSL Certificate Issue",
			Description: finding.Message + ". " + finding.Detail,
			Impact:      r.formatAffectedVS(finding.AffectedVS),
		}
	case "cipher":
		return Recommendation{
			Priority:    finding.Severity,
			Title:       "SSL/TLS Configuration Issue",
			Description: finding.Message + ". " + finding.Detail,
			Impact:      r.formatAffectedVS(finding.AffectedVS),
		}
	case "configuration":
		return Recommendation{
			Priority:    finding.Severity,
			Title:       "SSL Configuration Issue",
			Description: finding.Message,
			Impact:      r.formatAffectedVS(finding.AffectedVS),
		}
	}
	return Recommendation{}
}

func (r *Recommender) errorRecommendation(err TopError) Recommendation {
	if err.Count < 10 {
		return Recommendation{}
	}

	priority := "medium"
	if err.Count > 100 {
		priority = "high"
	}
	if err.Count > 500 {
		priority = "critical"
	}

	return Recommendation{
		Priority:    priority,
		Title:       "Investigate recurring error",
		Description: fmt.Sprintf("Error occurred %d times: %s", err.Count, err.Message),
		Impact:      fmt.Sprintf("Last occurred: %s", err.LastOccurred.Format("2006-01-02 15:04:05")),
	}
}

func (r *Recommender) formatAffectedVS(vs []string) string {
	if len(vs) == 0 {
		return "Virtual servers affected: unknown"
	}
	return "Affected: " + strings.Join(vs, ", ")
}

func (r *Recommender) sortByPriority(recs []Recommendation) {
	priorityOrder := map[string]int{
		"critical": 0,
		"high":     1,
		"medium":   2,
		"low":      3,
	}

	for i := 0; i < len(recs)-1; i++ {
		for j := i + 1; j < len(recs); j++ {
			if priorityOrder[recs[i].Priority] > priorityOrder[recs[j].Priority] {
				recs[i], recs[j] = recs[j], recs[i]
			}
		}
	}
}

func (r *Recommender) deduplicate(recs []Recommendation) []Recommendation {
	seen := make(map[string]bool)
	result := []Recommendation{}

	for _, rec := range recs {
		key := rec.Title + rec.Priority
		if !seen[key] {
			result = append(result, rec)
			seen[key] = true
		}
	}

	return result
}
