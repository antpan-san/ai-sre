package cli

import (
	"strings"
	"testing"
)

func TestFormatDomainProbeTextSections(t *testing.T) {
	text := formatDomainProbeText(&domainDiagnoseReport{
		Domain: "opsfleetpilot.com",
		DNS:    []domainDNSRecord{{Type: "A", Value: "204.44.123.101"}},
		HTTP: []domainHTTPProbe{{
			URL: "http://opsfleetpilot.com", StatusCode: 200, LatencyMs: 100, ServerHeader: "nginx",
		}},
		TLS: &domainTLSProbe{Host: "opsfleetpilot.com:443", Error: "connection refused"},
	})
	for _, sec := range []string{"【DNS 解析】", "【HTTP(S) 探测】", "【TLS / 443】", "204.44.123.101"} {
		if !strings.Contains(text, sec) {
			t.Fatalf("missing %q in:\n%s", sec, text)
		}
	}
}
