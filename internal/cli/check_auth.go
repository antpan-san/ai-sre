package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"golang.org/x/term"
)

var errAuthCredentialsRequired = errors.New("auth_credentials_required")

func isAuthCredentialsError(err error) bool {
	return errors.Is(err, errAuthCredentialsRequired) || errors.Is(err, errRedisAuthRequired)
}

func promptSecret(label string) (string, error) {
	_, _ = fmt.Fprintf(os.Stderr, "[%s] %s: ", progName, label)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		_, _ = fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	var line string
	_, err := fmt.Fscanln(os.Stdin, &line)
	return strings.TrimSpace(line), err
}

func promptLine(label string) (string, error) {
	_, _ = fmt.Fprintf(os.Stderr, "[%s] %s: ", progName, label)
	var line string
	_, err := fmt.Fscanln(os.Stdin, &line)
	return strings.TrimSpace(line), err
}

func emitAuthRequiredJSON(topic string, fields map[string]any) error {
	payload := map[string]any{
		"auth_required": true,
		"topic":         normalizeCheckTopic(topic),
		"status":        "auth_required",
	}
	for k, v := range fields {
		payload[k] = v
	}
	if strings.EqualFold(outputFormat, "json") {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
	}
	return errAuthCredentialsRequired
}

func authRequiredMessage(topic, hint string) string {
	return fmt.Sprintf("%s 需要认证信息；%s", normalizeCheckTopic(topic), hint)
}

func isMySQLAuthError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "access denied") ||
		strings.Contains(lower, "using password: no") ||
		strings.Contains(lower, "password: no")
}

func mysqlDSNHasPassword(dsn string) bool {
	at := strings.Index(dsn, "@")
	if at <= 0 {
		return false
	}
	return strings.Contains(dsn[:at], ":")
}

func injectMySQLPassword(dsn, password string) string {
	at := strings.Index(dsn, "@")
	if at <= 0 {
		return dsn
	}
	userinfo := dsn[:at]
	rest := dsn[at:]
	if idx := strings.Index(userinfo, ":"); idx > 0 {
		userinfo = userinfo[:idx+1] + password
	} else {
		userinfo = userinfo + ":" + password
	}
	return userinfo + rest
}

func isPostgreSQLAuthError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "password authentication failed") ||
		strings.Contains(lower, "no password supplied") ||
		strings.Contains(lower, "fe_sendauth")
}

func postgresqlDSNHasPassword(dsn string) bool {
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return false
	}
	_, ok := u.User.Password()
	return ok
}

func injectPostgreSQLPassword(dsn, password string) string {
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return dsn
	}
	user := u.User.Username()
	u.User = url.UserPassword(user, password)
	return u.String()
}

func isElasticsearchAuthError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "http 401") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "authentication")
}

func isKafkaAuthLikely(text string) bool {
	lower := strings.ToLower(text)
	for _, s := range []string{
		"sasl", "authentication", "auth failed", "authentication failed",
		"invalid credentials",
	} {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
