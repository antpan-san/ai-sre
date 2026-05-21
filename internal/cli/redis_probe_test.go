package cli

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func readRedisCommand(r *bufio.Reader) string {
	line, err := r.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "*") {
		return ""
	}
	n, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "*")))
	var parts []string
	for i := 0; i < n; i++ {
		_, _ = r.ReadString('\n')
		ln, _ := r.ReadString('\n')
		parts = append(parts, strings.TrimSpace(ln))
	}
	return strings.ToUpper(strings.Join(parts, " "))
}

func TestCollectRedisProbeNOAUTH(t *testing.T) {
	requireLocalTCPListen(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		if conn == nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		cmd := readRedisCommand(r)
		if strings.Contains(cmd, "PING") {
			_, _ = conn.Write([]byte("-NOAUTH Authentication required.\r\n"))
		}
	}()
	report := CollectRedisProbe(redisProbeOptions{Address: ln.Addr().String(), Timeout: time.Second})
	if !report.AuthRequired {
		t.Fatalf("want auth_required, got %+v", report)
	}
}

func TestCollectRedisProbePINGAndINFO(t *testing.T) {
	requireLocalTCPListen(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	infoBody := "redis_version:7.2.0\r\nrole:master\r\nredis_mode:standalone\r\ncluster_enabled:0\r\nused_memory_human:2M\r\nconnected_clients:3\r\n"
	go func() {
		conn, _ := ln.Accept()
		if conn == nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		for {
			cmd := readRedisCommand(r)
			if cmd == "" {
				return
			}
			switch {
			case strings.Contains(cmd, "PING"):
				_, _ = conn.Write([]byte("+PONG\r\n"))
			case strings.Contains(cmd, "INFO ALL"), strings.Contains(cmd, "INFO"):
				_, _ = conn.Write([]byte("$" + strconv.Itoa(len(infoBody)) + "\r\n" + infoBody + "\r\n"))
			default:
				_, _ = conn.Write([]byte("-ERR noperm\r\n"))
			}
		}
	}()
	report := CollectRedisProbe(redisProbeOptions{Address: ln.Addr().String(), Timeout: 2 * time.Second})
	if report.RedisVersion != "7.2.0" {
		t.Fatalf("version=%q errors=%v", report.RedisVersion, report.Errors)
	}
	if report.AuthRequired {
		t.Fatal("unexpected auth")
	}
}

func TestRedisTargetFromFlags(t *testing.T) {
	addr := redisTargetFromFlags(map[string]string{"addr": "10.0.0.8"})
	if addr != "10.0.0.8:6379" {
		t.Fatalf("got %q", addr)
	}
}
