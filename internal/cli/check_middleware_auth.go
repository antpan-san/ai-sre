package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func finishMySQLCheckEvidence(topic string, ctx map[string]string) error {
	if normalizeCheckTopic(topic) != "mysql" {
		return nil
	}
	dsn := strings.TrimSpace(ctx["dsn"])
	if dsn == "" {
		return nil
	}
	if body := strings.TrimSpace(ctx["mysql_diagnose_json"]); body != "" && !mysqlProbeNeedsAuth(body) {
		return nil
	}
	body, err := collectMySQLProbeJSON(context.Background(), ctx)
	if err == nil {
		if body != "" {
			ctx["mysql_diagnose_json"] = body
		}
		return nil
	}
	if !errorsIsAuth(err) {
		return err
	}
	return handleMySQLAuth(topic, dsn, ctx)
}

func finishPostgreSQLCheckEvidence(topic string, ctx map[string]string) error {
	if normalizeCheckTopic(topic) != "postgresql" {
		return nil
	}
	dsn := strings.TrimSpace(ctx["dsn"])
	if dsn == "" {
		return nil
	}
	if body := strings.TrimSpace(ctx["postgresql_diagnose_json"]); body != "" && !postgresqlProbeNeedsAuth(body) {
		return nil
	}
	body, err := collectPostgreSQLProbeJSON(context.Background(), ctx)
	if err == nil {
		if body != "" {
			ctx["postgresql_diagnose_json"] = body
		}
		return nil
	}
	if !errorsIsAuth(err) {
		return err
	}
	return handlePostgreSQLAuth(topic, dsn, ctx)
}

func finishElasticsearchCheckEvidence(topic string, ctx map[string]string) error {
	t := normalizeCheckTopic(topic)
	if t != "elasticsearch" {
		return nil
	}
	base := elasticsearchURLFromFlags(ctx)
	if base == "" {
		return nil
	}
	if body := strings.TrimSpace(ctx["es_diagnose_json"]); body != "" && !elasticsearchProbeNeedsAuth(body) {
		return nil
	}
	body, err := collectElasticsearchProbeJSON(context.Background(), ctx)
	if err == nil {
		if body != "" {
			ctx["es_diagnose_json"] = body
		}
		return nil
	}
	if !errorsIsAuth(err) {
		return err
	}
	return handleElasticsearchAuth(topic, base, ctx)
}

func finishKafkaCheckEvidence(topic string, ctx map[string]string) error {
	if normalizeCheckTopic(topic) != "kafka" {
		return nil
	}
	bootstrap := kafkaBootstrapFromFlags(ctx)
	if bootstrap == "" {
		return nil
	}
	if body := strings.TrimSpace(ctx["kafka_diagnose_json"]); body != "" && !kafkaProbeNeedsConfig(body, ctx) {
		return nil
	}
	body, err := collectKafkaProbeJSON(context.Background(), ctx)
	if err == nil {
		if body != "" {
			ctx["kafka_diagnose_json"] = body
		}
		return nil
	}
	if !errorsIsAuth(err) {
		return err
	}
	return handleKafkaAuth(topic, bootstrap, ctx)
}

func errorsIsAuth(err error) bool {
	return isAuthCredentialsError(err)
}

func collectMySQLProbeJSON(ctx context.Context, flags map[string]string) (string, error) {
	dsn := strings.TrimSpace(flags["dsn"])
	if dsn == "" {
		return "", fmt.Errorf("mysql DSN 未配置")
	}
	report := runMySQLDiagnose(ctx, mysqlDiagnoseOptions{DSN: dsn, Timeout: 20 * time.Second})
	if len(report.Errors) > 0 && isMySQLAuthError(strings.Join(report.Errors, "; ")) && !mysqlDSNHasPassword(dsn) {
		return marshalProbeReport(report, errAuthCredentialsRequired)
	}
	b, err := json.Marshal(report)
	return string(b), err
}

func collectPostgreSQLProbeJSON(ctx context.Context, flags map[string]string) (string, error) {
	dsn := strings.TrimSpace(flags["dsn"])
	if dsn == "" {
		return "", fmt.Errorf("postgresql DSN 未配置")
	}
	report := runPostgreSQLDiagnose(ctx, postgresqlDiagnoseOptions{DSN: dsn, Timeout: 20 * time.Second})
	if len(report.Errors) > 0 && isPostgreSQLAuthError(strings.Join(report.Errors, "; ")) && !postgresqlDSNHasPassword(dsn) {
		return marshalProbeReport(report, errAuthCredentialsRequired)
	}
	b, err := json.Marshal(report)
	return string(b), err
}

func collectElasticsearchProbeJSON(ctx context.Context, flags map[string]string) (string, error) {
	base := elasticsearchURLFromFlags(flags)
	if base == "" {
		return "", fmt.Errorf("elasticsearch 地址未配置")
	}
	user := strings.TrimSpace(flags["user"])
	pass := strings.TrimSpace(flags["password"])
	report, err := runElasticsearchDiagnose(ctx, elasticsearchDiagnoseOptions{
		BaseURL: base, User: user, Password: pass, Timeout: 15 * time.Second,
	})
	if report == nil {
		return "", err
	}
	errText := strings.Join(report.Errors, "; ")
	if isElasticsearchAuthError(errText) && user == "" && pass == "" {
		b, _ := json.Marshal(report)
		return string(b), errAuthCredentialsRequired
	}
	b, mErr := json.Marshal(report)
	if mErr != nil {
		return "", mErr
	}
	return string(b), nil
}

func collectKafkaProbeJSON(ctx context.Context, flags map[string]string) (string, error) {
	bootstrap := kafkaBootstrapFromFlags(flags)
	if bootstrap == "" {
		return "", fmt.Errorf("kafka bootstrap 未配置")
	}
	opts := kafkaDiagnoseOptions{
		BootstrapServer: bootstrap,
		ClientConfig:    kafkaConfigFromFlags(flags),
		Timeout:         20 * time.Second,
	}
	report, err := runKafkaDiagnose(ctx, opts)
	if report == nil {
		return "", err
	}
	errText := strings.Join(report.Errors, "; ")
	if opts.ClientConfig == "" && isKafkaAuthLikely(errText) {
		b, _ := json.Marshal(report)
		return string(b), errAuthCredentialsRequired
	}
	b, mErr := json.Marshal(report)
	if mErr != nil {
		return "", mErr
	}
	return string(b), nil
}

func handleMySQLAuth(topic, dsn string, ctx map[string]string) error {
	hint := "非 TTY 请通过 DSN 携带密码或设置 AI_SRE_MYSQL_DSN"
	if !isStdinTTY() {
		_ = emitAuthRequiredJSON(topic, map[string]any{"field": "password", "message": authRequiredMessage(topic, hint)})
		return fmt.Errorf("%s", authRequiredMessage(topic, hint))
	}
	pw, err := promptSecret("MySQL 密码")
	if err != nil {
		return err
	}
	ctx["dsn"] = injectMySQLPassword(dsn, pw)
	body, err := collectMySQLProbeJSON(context.Background(), ctx)
	if err != nil {
		return err
	}
	ctx["mysql_diagnose_json"] = body
	return nil
}

func handlePostgreSQLAuth(topic, dsn string, ctx map[string]string) error {
	hint := "非 TTY 请在 DSN 中包含 password 或设置 AI_SRE_POSTGRES_DSN"
	if !isStdinTTY() {
		_ = emitAuthRequiredJSON(topic, map[string]any{"field": "password", "message": authRequiredMessage(topic, hint)})
		return fmt.Errorf("%s", authRequiredMessage(topic, hint))
	}
	pw, err := promptSecret("PostgreSQL 密码")
	if err != nil {
		return err
	}
	ctx["dsn"] = injectPostgreSQLPassword(dsn, pw)
	body, err := collectPostgreSQLProbeJSON(context.Background(), ctx)
	if err != nil {
		return err
	}
	ctx["postgresql_diagnose_json"] = body
	return nil
}

func handleElasticsearchAuth(topic, base string, ctx map[string]string) error {
	hint := "非 TTY 请使用 -d user= -d password= 或环境变量"
	if !isStdinTTY() {
		_ = emitAuthRequiredJSON(topic, map[string]any{"field": "basic_auth", "url": base, "message": authRequiredMessage(topic, hint)})
		return fmt.Errorf("%s", authRequiredMessage(topic, hint))
	}
	user, err := promptLine("Elasticsearch 用户名")
	if err != nil {
		return err
	}
	pass, err := promptSecret("Elasticsearch 密码")
	if err != nil {
		return err
	}
	ctx["user"] = user
	ctx["password"] = pass
	body, err := collectElasticsearchProbeJSON(context.Background(), ctx)
	if err != nil {
		return err
	}
	ctx["es_diagnose_json"] = body
	return nil
}

func handleKafkaAuth(topic, bootstrap string, ctx map[string]string) error {
	hint := "非 TTY 请通过 -d config=/path/to/client.properties 指定 SASL/TLS 配置"
	if !isStdinTTY() {
		_ = emitAuthRequiredJSON(topic, map[string]any{
			"field":     "client.properties",
			"bootstrap": bootstrap,
			"message":   authRequiredMessage(topic, hint),
		})
		return fmt.Errorf("%s", authRequiredMessage(topic, hint))
	}
	path, err := promptLine("Kafka client.properties 路径 (SASL/TLS)")
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("Kafka 认证需要 client.properties 路径")
	}
	ctx["config"] = path
	body, err := collectKafkaProbeJSON(context.Background(), ctx)
	if err != nil {
		return err
	}
	ctx["kafka_diagnose_json"] = body
	return nil
}

func marshalProbeReport(report any, err error) (string, error) {
	b, _ := json.Marshal(report)
	return string(b), err
}

func mysqlProbeNeedsAuth(body string) bool {
	var r mysqlDiagnoseReport
	if json.Unmarshal([]byte(body), &r) != nil {
		return false
	}
	return len(r.Errors) > 0 && isMySQLAuthError(strings.Join(r.Errors, "; "))
}

func postgresqlProbeNeedsAuth(body string) bool {
	var r postgresqlDiagnoseReport
	if json.Unmarshal([]byte(body), &r) != nil {
		return false
	}
	return len(r.Errors) > 0 && isPostgreSQLAuthError(strings.Join(r.Errors, "; "))
}

func elasticsearchProbeNeedsAuth(body string) bool {
	var r elasticsearchDiagnoseReport
	if json.Unmarshal([]byte(body), &r) != nil {
		return false
	}
	return isElasticsearchAuthError(strings.Join(r.Errors, "; "))
}

func kafkaProbeNeedsConfig(body string, ctx map[string]string) bool {
	if kafkaConfigFromFlags(ctx) != "" {
		return false
	}
	var r kafkaDiagnoseReport
	if json.Unmarshal([]byte(body), &r) != nil {
		return false
	}
	return isKafkaAuthLikely(strings.Join(r.Errors, "; "))
}

func elasticsearchURLFromFlags(flags map[string]string) string {
	if flags == nil {
		return ""
	}
	for _, k := range []string{"url", "base_url", "addr"} {
		if v := strings.TrimSpace(flags[k]); v != "" {
			return normalizeCheckTargetValue("elasticsearch", v)
		}
	}
	return ""
}

func kafkaBootstrapFromFlags(flags map[string]string) string {
	if flags == nil {
		return ""
	}
	for _, k := range []string{"bootstrap", "bootstrap_server", "addr", "kafka_bootstrap"} {
		if v := strings.TrimSpace(flags[k]); v != "" {
			return normalizeCheckTargetValue("kafka", v)
		}
	}
	return ""
}

func kafkaConfigFromFlags(flags map[string]string) string {
	if flags == nil {
		return ""
	}
	for _, k := range []string{"config", "client_config", "client.properties"} {
		if v := strings.TrimSpace(flags[k]); v != "" {
			return v
		}
	}
	return ""
}
