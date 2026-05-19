package cli

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type domainDiagnoseOptions struct {
	Domain string
	Scheme string
	Port   string
	JSON   bool
}

type domainDNSRecord struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type domainHTTPProbe struct {
	URL          string `json:"url"`
	StatusCode   int    `json:"status_code,omitempty"`
	LatencyMs    int64  `json:"latency_ms,omitempty"`
	Error        string `json:"error,omitempty"`
	RedirectTo   string `json:"redirect_to,omitempty"`
	ServerHeader string `json:"server_header,omitempty"`
}

type domainTLSProbe struct {
	Host      string `json:"host"`
	NotBefore string `json:"not_before,omitempty"`
	NotAfter  string `json:"not_after,omitempty"`
	Issuer    string `json:"issuer,omitempty"`
	DNSNames  string `json:"dns_names,omitempty"`
	Error     string `json:"error,omitempty"`
}

type domainDiagnoseReport struct {
	Domain   string            `json:"domain"`
	DNS      []domainDNSRecord `json:"dns,omitempty"`
	HTTP     []domainHTTPProbe `json:"http,omitempty"`
	TLS      *domainTLSProbe   `json:"tls,omitempty"`
	Findings []string          `json:"findings,omitempty"`
}

func domainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "域名 / DNS / HTTP(S) 只读诊断",
	}
	cmd.AddCommand(domainDiagnoseCmd())
	return cmd
}

func domainDiagnoseCmd() *cobra.Command {
	var opts domainDiagnoseOptions
	cmd := &cobra.Command{
		Use:        "domain <fqdn>",
		Short:      "（已弃用）请改用 probe domain",
		Deprecated: "use \"probe domain\" instead",
		Args:       cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Domain = normalizeDomainName(args[0])
			report := runDomainDiagnose(opts)
			if opts.JSON {
				b, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(b))
				return nil
			}
			printDomainDiagnoseHuman(report)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.Scheme, "scheme", "", "优先探测协议 https 或 http（默认两者都试）")
	cmd.Flags().StringVar(&opts.Port, "port", "", "覆盖端口，如 443、80、9080")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "JSON 输出")
	return cmd
}

func probeDomainCmd() *cobra.Command {
	c := domainDiagnoseCmd()
	c.Use = "domain <fqdn>"
	c.Short = "域名只读快采：DNS 解析、HTTP(S) 可达、TLS 证书"
	c.Deprecated = ""
	return c
}

func normalizeDomainName(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if h, _, err := net.SplitHostPort(s); err == nil {
		s = h
	}
	return strings.Trim(strings.ToLower(s), ".")
}

func mergeDomainIntoContext(ctx map[string]string, topic string, args []string) {
	if ctx == nil {
		return
	}
	if _, ok := ctx["domain"]; ok && strings.TrimSpace(ctx["domain"]) != "" {
		return
	}
	if len(args) >= 2 && isDomainTopic(topic) {
		ctx["domain"] = normalizeDomainName(args[1])
	}
}

func isDomainTopic(topic string) bool {
	switch strings.ToLower(strings.TrimSpace(topic)) {
	case "domain", "dns":
		return true
	default:
		return false
	}
}

func runDomainDiagnose(opts domainDiagnoseOptions) *domainDiagnoseReport {
	d := normalizeDomainName(opts.Domain)
	report := &domainDiagnoseReport{Domain: d}
	if d == "" {
		report.Findings = append(report.Findings, "域名为空")
		return report
	}

	resolver := net.Resolver{PreferGo: true}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	if cname, err := resolver.LookupCNAME(ctx, d); err == nil && cname != "" && cname != d+"." {
		report.DNS = append(report.DNS, domainDNSRecord{Type: "CNAME", Value: strings.TrimSuffix(cname, ".")})
	} else if err != nil && !isDNSNotFound(err) {
		report.Findings = append(report.Findings, "CNAME 查询: "+err.Error())
	}

	if ips, err := resolver.LookupIP(ctx, "ip4", d); err == nil {
		for _, ip := range ips {
			report.DNS = append(report.DNS, domainDNSRecord{Type: "A", Value: ip.String()})
		}
	} else if err != nil && !isDNSNotFound(err) {
		report.Findings = append(report.Findings, "A 记录: "+err.Error())
	}
	if ips, err := resolver.LookupIP(ctx, "ip6", d); err == nil {
		for _, ip := range ips {
			report.DNS = append(report.DNS, domainDNSRecord{Type: "AAAA", Value: ip.String()})
		}
	}

	if len(report.DNS) == 0 {
		report.Findings = append(report.Findings, "未解析到任何 A/AAAA/CNAME 记录")
	}

	schemes := []string{"https", "http"}
	if s := strings.TrimSpace(opts.Scheme); s != "" {
		schemes = []string{strings.ToLower(s)}
	}
	for _, scheme := range schemes {
		port := defaultPortForScheme(scheme, opts.Port)
		url := fmt.Sprintf("%s://%s", scheme, d)
		if port != "" && !((scheme == "https" && port == "443") || (scheme == "http" && port == "80")) {
			url = fmt.Sprintf("%s://%s:%s", scheme, d, port)
		}
		report.HTTP = append(report.HTTP, probeDomainHTTP(url))
	}

	if report.TLS == nil || report.TLS.Error != "" {
		host := d
		port := strings.TrimSpace(opts.Port)
		if port == "" {
			port = "443"
		}
		report.TLS = probeDomainTLS(host, port)
	}

	report.Findings = append(report.Findings, summarizeDomainFindings(report)...)
	return report
}

func defaultPortForScheme(scheme, override string) string {
	if p := strings.TrimSpace(override); p != "" {
		return p
	}
	if scheme == "https" {
		return "443"
	}
	return "80"
}

func probeDomainHTTP(url string) domainHTTPProbe {
	p := domainHTTPProbe{URL: url}
	client := &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	start := time.Now()
	resp, err := client.Get(url)
	p.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		p.Error = err.Error()
		return p
	}
	defer resp.Body.Close()
	p.StatusCode = resp.StatusCode
	p.ServerHeader = resp.Header.Get("Server")
	if loc := resp.Header.Get("Location"); loc != "" && resp.StatusCode >= 300 && resp.StatusCode < 400 {
		p.RedirectTo = loc
	}
	return p
}

func probeDomainTLS(host, port string) *domainTLSProbe {
	t := &domainTLSProbe{Host: net.JoinHostPort(host, port)}
	addr := t.Host
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		t.Error = err.Error()
		return t
	}
	defer conn.Close()
	cert := conn.ConnectionState().PeerCertificates
	if len(cert) == 0 {
		t.Error = "no peer certificate"
		return t
	}
	leaf := cert[0]
	t.NotBefore = leaf.NotBefore.UTC().Format(time.RFC3339)
	t.NotAfter = leaf.NotAfter.UTC().Format(time.RFC3339)
	t.Issuer = leaf.Issuer.String()
	if len(leaf.DNSNames) > 0 {
		t.DNSNames = strings.Join(leaf.DNSNames, ",")
	}
	now := time.Now()
	if now.After(leaf.NotAfter) {
		t.Error = "certificate expired"
	} else if now.Add(14 * 24 * time.Hour).After(leaf.NotAfter) {
		t.Error = "certificate expires within 14 days"
	}
	return t
}

func isDNSNotFound(err error) bool {
	if err == nil {
		return false
	}
	if de, ok := err.(*net.DNSError); ok {
		return de.IsNotFound
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such host") || strings.Contains(msg, "not found")
}

func summarizeDomainFindings(r *domainDiagnoseReport) []string {
	var out []string
	if len(r.DNS) == 0 {
		return out
	}
	okHTTP := false
	for _, h := range r.HTTP {
		if h.Error == "" && h.StatusCode > 0 && h.StatusCode < 500 {
			okHTTP = true
		}
	}
	if !okHTTP {
		out = append(out, "HTTP(S) 探测未得到成功响应，请检查解析 IP、端口、防火墙与 Nginx/应用")
	}
	if r.TLS != nil && r.TLS.Error != "" {
		out = append(out, "TLS: "+r.TLS.Error)
	}
	return out
}

func printDomainDiagnoseHuman(r *domainDiagnoseReport) {
	fmt.Printf("域名: %s\n", r.Domain)
	if len(r.DNS) > 0 {
		fmt.Println("DNS:")
		for _, rec := range r.DNS {
			fmt.Printf("  %s  %s\n", rec.Type, rec.Value)
		}
	}
	for _, h := range r.HTTP {
		if h.Error != "" {
			fmt.Printf("HTTP %s: 失败 %s\n", h.URL, h.Error)
			continue
		}
		fmt.Printf("HTTP %s: %d (%dms)", h.URL, h.StatusCode, h.LatencyMs)
		if h.RedirectTo != "" {
			fmt.Printf(" -> %s", h.RedirectTo)
		}
		fmt.Println()
	}
	if r.TLS != nil {
		if r.TLS.Error != "" {
			fmt.Printf("TLS %s: %s\n", r.TLS.Host, r.TLS.Error)
		} else {
			fmt.Printf("TLS: 有效期 %s ~ %s\n", r.TLS.NotBefore, r.TLS.NotAfter)
		}
	}
	for _, f := range r.Findings {
		fmt.Printf("- %s\n", f)
	}
}

func gatherDomainEvidence(ctx context.Context, flags map[string]string, out map[string]string) {
	d := strings.TrimSpace(flags["domain"])
	if d == "" {
		return
	}
	opts := domainDiagnoseOptions{Domain: d, Scheme: flags["scheme"], Port: flags["port"]}
	report := runDomainDiagnose(opts)
	text := formatDomainProbeText(report)
	if text != "" {
		out["domain_probe_text"] = truncateBytes(text, maxBytesPerTopicEvidence)
	}
	b, err := json.Marshal(report)
	if err != nil {
		return
	}
	out["domain_probe_json"] = truncateBytes(string(b), maxBytesPerTopicEvidence)
}

// formatDomainProbeText 生成域名诊断的纯文本报告（供 check 直出与服务端 AI 引用）。
func formatDomainProbeText(r *domainDiagnoseReport) string {
	if r == nil || strings.TrimSpace(r.Domain) == "" {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "=== 域名诊断：%s ===\n", r.Domain)
	fmt.Fprintf(&b, "采集时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05 MST"))

	b.WriteString("【DNS 解析】\n")
	if len(r.DNS) == 0 {
		b.WriteString("  (无 A/AAAA/CNAME 记录)\n")
	} else {
		for _, rec := range r.DNS {
			fmt.Fprintf(&b, "  %-6s %s\n", rec.Type, rec.Value)
		}
	}
	b.WriteString("\n【HTTP(S) 探测】\n")
	if len(r.HTTP) == 0 {
		b.WriteString("  (未执行)\n")
	} else {
		for _, h := range r.HTTP {
			if h.Error != "" {
				fmt.Fprintf(&b, "  %s\n    结果: 失败\n    错误: %s\n", h.URL, h.Error)
				continue
			}
			fmt.Fprintf(&b, "  %s\n    状态: %d  耗时: %dms\n", h.URL, h.StatusCode, h.LatencyMs)
			if h.ServerHeader != "" {
				fmt.Fprintf(&b, "    Server: %s\n", h.ServerHeader)
			}
			if h.RedirectTo != "" {
				fmt.Fprintf(&b, "    重定向: %s\n", h.RedirectTo)
			}
		}
	}
	b.WriteString("\n【TLS / 443】\n")
	if r.TLS == nil {
		b.WriteString("  (未探测)\n")
	} else if r.TLS.Error != "" {
		fmt.Fprintf(&b, "  目标: %s\n    结果: %s\n", r.TLS.Host, r.TLS.Error)
	} else {
		fmt.Fprintf(&b, "  目标: %s\n    有效期: %s ~ %s\n", r.TLS.Host, r.TLS.NotBefore, r.TLS.NotAfter)
		if r.TLS.Issuer != "" {
			fmt.Fprintf(&b, "    颁发者: %s\n", r.TLS.Issuer)
		}
		if r.TLS.DNSNames != "" {
			fmt.Fprintf(&b, "    SAN: %s\n", r.TLS.DNSNames)
		}
	}
	if len(r.Findings) > 0 {
		b.WriteString("\n【采集结论】\n")
		for _, f := range r.Findings {
			fmt.Fprintf(&b, "  - %s\n", f)
		}
	}
	return strings.TrimSpace(b.String())
}
