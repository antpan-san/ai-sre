package cli

import (
	"regexp"
	"strings"
)

var cliMarkdownHeading = regexp.MustCompile(`(?m)^#{1,3}\s+(.+?)\s*$`)

// isMiddlewareEvidenceTopic reports middleware topics using probe JSON evidence.
func isMiddlewareEvidenceTopic(topic string) bool {
	switch normalizeCheckTopic(topic) {
	case "redis", "kafka", "mysql", "postgresql", "elasticsearch":
		return true
	default:
		if t := strings.ToLower(strings.TrimSpace(topic)); t == "nginx" || t == "es" {
			return true
		}
		return false
	}
}

// formatCheckAnswerText normalizes server AI output to plain text for terminal display.
func formatCheckAnswerText(topic, answer string) string {
	s := strings.TrimSpace(answer)
	if s == "" {
		return s
	}
	if isMiddlewareEvidenceTopic(topic) || isDomainTopic(topic) {
		s = normalizePlainTextDiagnose(s)
	}
	return s
}

func normalizePlainTextDiagnose(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			continue
		}
		if m := cliMarkdownHeading.FindStringSubmatch(trim); len(m) == 2 {
			title := strings.TrimSpace(m[1])
			title = strings.TrimSuffix(title, "（一句话）")
			if !strings.HasPrefix(title, "【") {
				title = "【" + title + "】"
			}
			out = append(out, title)
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
