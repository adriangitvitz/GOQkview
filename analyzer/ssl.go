package analyzer

import (
	"regexp"
	"strings"

	"goqkview/interfaces"
)

type SSLAnalyzer struct {
	certExpiryPattern    *regexp.Regexp
	tlsVersionPattern    *regexp.Regexp
	cipherPattern        *regexp.Regexp
	handshakePattern     *regexp.Regexp
	virtualServerPattern *regexp.Regexp
}

func NewSSLAnalyzer() *SSLAnalyzer {
	return &SSLAnalyzer{
		certExpiryPattern:    regexp.MustCompile(`(?i)certificate.*expir|cert.*expir|ssl.*expir|expir.*certificate`),
		tlsVersionPattern:    regexp.MustCompile(`(?i)TLS\s*(1\.0|1\.1|1\.2|1\.3)|SSLv[23]`),
		cipherPattern:        regexp.MustCompile(`(?i)cipher|RC4|DES|MD5|NULL|EXPORT|WEAK`),
		handshakePattern:     regexp.MustCompile(`(?i)ssl\s*handshake|handshake\s*fail|certificate\s*verify`),
		virtualServerPattern: regexp.MustCompile(`(?i)vs_[\w-]+|virtual[-_]?server[\s:]+(\S+)`),
	}
}

func (s *SSLAnalyzer) Analyze(entries []interfaces.LogEntry) []SSLFinding {
	findings := []SSLFinding{}
	seen := make(map[string]bool)

	for _, entry := range entries {
		line := entry.Line

		if s.certExpiryPattern.MatchString(line) {
			finding := s.analyzeCertIssue(entry)
			key := finding.Type + finding.Message
			if !seen[key] {
				findings = append(findings, finding)
				seen[key] = true
			}
		}

		if s.tlsVersionPattern.MatchString(line) {
			finding := s.analyzeTLSVersion(entry)
			if finding.Severity != "" {
				key := finding.Type + finding.Message
				if !seen[key] {
					findings = append(findings, finding)
					seen[key] = true
				}
			}
		}

		if s.cipherPattern.MatchString(line) && (entry.Status == "ERROR" || entry.Status == "WARNING" || entry.Status == "CRITICAL") {
			finding := s.analyzeCipherIssue(entry)
			if finding.Severity != "" {
				key := finding.Type + finding.Message
				if !seen[key] {
					findings = append(findings, finding)
					seen[key] = true
				}
			}
		}

		if s.handshakePattern.MatchString(line) && (entry.Status == "ERROR" || entry.Status == "CRITICAL") {
			finding := s.analyzeHandshakeIssue(entry)
			key := finding.Type + finding.Message
			if !seen[key] {
				findings = append(findings, finding)
				seen[key] = true
			}
		}
	}

	return findings
}

func (s *SSLAnalyzer) analyzeCertIssue(entry interfaces.LogEntry) SSLFinding {
	finding := SSLFinding{
		Type:       "certificate",
		AffectedVS: s.extractVirtualServers(entry.Line),
	}

	switch entry.Status {
	case "CRITICAL", "SEVERE", "ERROR":
		finding.Severity = "critical"
	default:
		finding.Severity = "warning"
	}

	finding.Message = "Certificate expiration detected"
	finding.Detail = s.extractDetail(entry.Line)

	return finding
}

func (s *SSLAnalyzer) analyzeTLSVersion(entry interfaces.LogEntry) SSLFinding {
	finding := SSLFinding{
		Type:       "cipher",
		AffectedVS: s.extractVirtualServers(entry.Line),
	}

	line := strings.ToLower(entry.Line)

	if strings.Contains(line, "tls 1.0") || strings.Contains(line, "tls1.0") ||
		strings.Contains(line, "sslv2") || strings.Contains(line, "sslv3") {
		finding.Severity = "critical"
		finding.Message = "Obsolete TLS/SSL protocol detected"
		finding.Detail = "TLS 1.0, SSLv2, and SSLv3 are deprecated and vulnerable"
	} else if strings.Contains(line, "tls 1.1") || strings.Contains(line, "tls1.1") {
		finding.Severity = "warning"
		finding.Message = "TLS 1.1 protocol in use"
		finding.Detail = "TLS 1.1 is deprecated, upgrade to TLS 1.2 or 1.3"
	}

	return finding
}

func (s *SSLAnalyzer) analyzeCipherIssue(entry interfaces.LogEntry) SSLFinding {
	finding := SSLFinding{
		Type:       "cipher",
		AffectedVS: s.extractVirtualServers(entry.Line),
	}

	line := strings.ToLower(entry.Line)

	weakCiphers := []string{"rc4", "des", "null", "export", "md5"}
	for _, cipher := range weakCiphers {
		if strings.Contains(line, cipher) {
			finding.Severity = "critical"
			finding.Message = "Weak cipher suite detected"
			finding.Detail = "Cipher contains: " + strings.ToUpper(cipher)
			break
		}
	}

	return finding
}

func (s *SSLAnalyzer) analyzeHandshakeIssue(entry interfaces.LogEntry) SSLFinding {
	return SSLFinding{
		Severity:   "warning",
		Type:       "configuration",
		Message:    "SSL handshake failure detected",
		Detail:     s.extractDetail(entry.Line),
		AffectedVS: s.extractVirtualServers(entry.Line),
	}
}

func (s *SSLAnalyzer) extractVirtualServers(line string) []string {
	matches := s.virtualServerPattern.FindAllStringSubmatch(line, -1)
	vs := []string{}
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 0 && !seen[match[0]] {
			vs = append(vs, match[0])
			seen[match[0]] = true
		}
	}
	return vs
}

func (s *SSLAnalyzer) extractDetail(line string) string {
	if len(line) > 200 {
		return line[:200] + "..."
	}
	return line
}
