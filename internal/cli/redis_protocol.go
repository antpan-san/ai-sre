package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func redisWriteCommand(conn interface{ Write([]byte) (int, error) }, args ...string) error {
	var b strings.Builder
	b.WriteString("*")
	b.WriteString(strconv.Itoa(len(args)))
	b.WriteString("\r\n")
	for _, arg := range args {
		b.WriteString("$")
		b.WriteString(strconv.Itoa(len(arg)))
		b.WriteString("\r\n")
		b.WriteString(arg)
		b.WriteString("\r\n")
	}
	_, err := conn.Write([]byte(b.String()))
	return err
}

func redisReadBulkString(r *bufio.Reader) (string, error) {
	head, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(head, "-") {
		return "", fmt.Errorf("%s", strings.TrimSpace(head))
	}
	if !strings.HasPrefix(head, "$") {
		return "", fmt.Errorf("unexpected reply: %s", strings.TrimSpace(head))
	}
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(head, "$")))
	if err != nil || n < 0 {
		return "", fmt.Errorf("bad bulk size: %s", strings.TrimSpace(head))
	}
	if n == 0 {
		_, _ = r.ReadString('\n')
		return "", nil
	}
	buf := make([]byte, n+2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

func parseRedisInfo(info string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(info, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		kv := strings.SplitN(s, ":", 2)
		if len(kv) == 2 {
			out[kv[0]] = kv[1]
		}
	}
	return out
}

func atoiOrZero(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func atoi64OrZero(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

func atofOrZero(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
