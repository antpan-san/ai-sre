package redis

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"ft-backend/common/config"
	"ft-backend/common/logger"

	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
	ctx    = context.Background()
)

// Key prefixes for namespacing.
const (
	PrefixTask         = "ofp:task:"           // Task queue
	PrefixSubTask      = "ofp:subtask:"        // Sub-task dispatch queue
	PrefixCache        = "ofp:cache:"          // General cache
	PrefixHeartbeat    = "ofp:hb:"             // Heartbeat last-seen
	PrefixLock         = "ofp:lock:"           // Distributed locks
	PrefixAgentVersion = "ofp:agent_version"   // Latest agent binary info
	PrefixOnline       = "ofp:online:"         // Machine online status (TTL-based)
	PrefixMetrics      = "ofp:metrics:"        // Machine real-time metrics (TTL-based)
	QueueHeartbeat     = "ofp:queue:heartbeat" // Heartbeat processing queue
)

// Connect initializes the Redis client.
func Connect(cfg *config.RedisConfig) error {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	logger.Info("Connecting to Redis: %s (db=%d)", addr, cfg.DB)
	if cfg.Password == "" {
		logger.Warn("Redis password is empty. Remote Redis may reject connections in protected mode")
	}

	Client = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     20,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	if err := Client.Ping(ctx).Err(); err != nil {
		diagnosis := diagnose(addr, cfg.Password)
		logger.Error("Redis connection failed: %v; diagnosis: %s", err, diagnosis)
		return fmt.Errorf("failed to connect redis: %w (diagnosis: %s)", err, diagnosis)
	}

	logger.Info("Redis connection established")
	return nil
}

// diagnose runs a low-level RESP PING probe to provide actionable Redis errors.
func diagnose(addr, password string) string {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return fmt.Sprintf("tcp dial failed: %v", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	// If password is configured, probe with AUTH first.
	if password != "" {
		authCmd := fmt.Sprintf("*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n", len(password), password)
		if _, err := conn.Write([]byte(authCmd)); err != nil {
			return fmt.Sprintf("auth write failed: %v", err)
		}
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			return fmt.Sprintf("auth read failed: %v", err)
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "-") {
			return "AUTH failed: " + line
		}
	}

	// PING probe
	if _, err := conn.Write([]byte("*1\r\n$4\r\nPING\r\n")); err != nil {
		return fmt.Sprintf("ping write failed: %v", err)
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Sprintf("ping read failed: %v", err)
	}
	line = strings.TrimSpace(line)

	switch {
	case strings.HasPrefix(line, "+PONG"):
		return "tcp reachable and redis responded to PING; check go-redis options"
	case strings.HasPrefix(line, "-DENIED"):
		return "server in protected-mode and rejected remote connection; set redis password/ACL and use it in conf/config.yaml"
	case strings.HasPrefix(line, "-NOAUTH"):
		return "server requires authentication; set redis.password in conf/config.yaml"
	case strings.HasPrefix(line, "-WRONGPASS"):
		return "invalid redis password in conf/config.yaml"
	default:
		return "redis replied: " + line
	}
}

// Close closes the Redis connection.
func Close() error {
	if Client == nil {
		return nil
	}
	logger.Info("Closing Redis connection")
	return Client.Close()
}

// IsConnected checks if Redis is available.
func IsConnected() bool {
	if Client == nil {
		return false
	}
	return Client.Ping(ctx).Err() == nil
}

// ---- Cache Layer ----

// Set stores a value with TTL.
func Set(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Client.Set(ctx, PrefixCache+key, data, ttl).Err()
}

// Get retrieves a cached value. Returns false if key doesn't exist.
func Get(key string, dest interface{}) (bool, error) {
	data, err := Client.Get(ctx, PrefixCache+key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(data, dest)
}

// Del removes a cached key.
func Del(key string) error {
	return Client.Del(ctx, PrefixCache+key).Err()
}

// ---- Task Queue (Redis List-based) ----

// EnqueueTask pushes a task payload to the dispatch queue for a client.
func EnqueueTask(clientID string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return Client.RPush(ctx, PrefixSubTask+clientID, data).Err()
}

// DequeueTask pops tasks from a client's dispatch queue (up to limit).
func DequeueTask(clientID string, limit int) ([]json.RawMessage, error) {
	var results []json.RawMessage
	for i := 0; i < limit; i++ {
		data, err := Client.LPop(ctx, PrefixSubTask+clientID).Bytes()
		if err == redis.Nil {
			break
		}
		if err != nil {
			return results, err
		}
		results = append(results, json.RawMessage(data))
	}
	return results, nil
}

// TaskQueueLen returns the number of pending tasks for a client.
func TaskQueueLen(clientID string) (int64, error) {
	return Client.LLen(ctx, PrefixSubTask+clientID).Result()
}

// ---- Heartbeat Tracking ----

// RecordHeartbeat marks a client as alive with a TTL (e.g., 30s).
func RecordHeartbeat(clientID string, ttl time.Duration) error {
	return Client.Set(ctx, PrefixHeartbeat+clientID, time.Now().Unix(), ttl).Err()
}

// IsClientAlive checks if a client has sent a heartbeat recently.
func IsClientAlive(clientID string) bool {
	_, err := Client.Get(ctx, PrefixHeartbeat+clientID).Result()
	return err == nil
}

// ---- Distributed Lock ----

// AcquireLock attempts to acquire a distributed lock. Returns true if acquired.
func AcquireLock(name string, ttl time.Duration) (bool, error) {
	ok, err := Client.SetNX(ctx, PrefixLock+name, "1", ttl).Result()
	return ok, err
}

// ReleaseLock releases a distributed lock.
func ReleaseLock(name string) error {
	return Client.Del(ctx, PrefixLock+name).Err()
}

// ---- Agent Version Management ----

// AgentVersionInfo stores the latest agent binary info.
type AgentVersionInfo struct {
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Checksum    string `json:"checksum"`
	UpdatedAt   int64  `json:"updated_at"`
}

// SetAgentVersion stores the latest agent version info.
func SetAgentVersion(os string, arch string, info AgentVersionInfo) error {
	key := fmt.Sprintf("%s:%s_%s", PrefixAgentVersion, os, arch)
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, 0).Err()
}

// GetAgentVersion retrieves the latest agent version info for a given OS/arch.
func GetAgentVersion(os string, arch string) (*AgentVersionInfo, error) {
	key := fmt.Sprintf("%s:%s_%s", PrefixAgentVersion, os, arch)
	data, err := Client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var info AgentVersionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ---- Heartbeat Queue (decoupled processing) ----

// EnqueueHeartbeat pushes raw heartbeat JSON to the processing queue.
func EnqueueHeartbeat(data []byte) error {
	return Client.LPush(ctx, QueueHeartbeat, data).Err()
}

// DequeueHeartbeat blocks until a heartbeat is available or timeout expires.
// Returns nil, nil on timeout (no data).
func DequeueHeartbeat(timeout time.Duration) ([]byte, error) {
	result, err := Client.BRPop(ctx, timeout, QueueHeartbeat).Result()
	if err == redis.Nil {
		return nil, nil // timeout, no data
	}
	if err != nil {
		return nil, err
	}
	// BRPop returns [key, value]
	if len(result) < 2 {
		return nil, nil
	}
	return []byte(result[1]), nil
}

// ---- Machine Online Status (TTL-based) ----

// SetMachineOnline marks a machine as online with a TTL.
// When the TTL expires, the machine is implicitly offline.
func SetMachineOnline(clientID string, ttl time.Duration) error {
	return Client.Set(ctx, PrefixOnline+clientID, time.Now().Unix(), ttl).Err()
}

// IsMachineOnline checks if a machine is currently online (key exists and not expired).
func IsMachineOnline(clientID string) bool {
	_, err := Client.Get(ctx, PrefixOnline+clientID).Result()
	return err == nil
}

// GetOnlineMachineIDs returns all currently online machine client IDs by scanning keys.
func GetOnlineMachineIDs() ([]string, error) {
	var clientIDs []string
	var cursor uint64

	for {
		keys, nextCursor, err := Client.Scan(ctx, cursor, PrefixOnline+"*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			id := key[len(PrefixOnline):]
			clientIDs = append(clientIDs, id)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return clientIDs, nil
}

// MachineMetrics stores fast-changing machine metrics in Redis.
type MachineMetrics struct {
	OSVersion     string  `json:"os_version,omitempty"`
	KernelVersion string  `json:"kernel_version,omitempty"`
	CPUCores      int     `json:"cpu_cores,omitempty"`
	CPUUsage      float64 `json:"cpu_usage,omitempty"`
	MemoryTotal   int64   `json:"memory_total,omitempty"`
	MemoryUsed    int64   `json:"memory_used,omitempty"`
	MemoryUsage   float64 `json:"memory_usage,omitempty"`
	DiskTotal     int64   `json:"disk_total,omitempty"`
	DiskUsed      int64   `json:"disk_used,omitempty"`
	DiskUsage     float64 `json:"disk_usage,omitempty"`
}

// SetMachineMetrics stores machine metrics with TTL.
func SetMachineMetrics(clientID string, metrics MachineMetrics, ttl time.Duration) error {
	if clientID == "" {
		return nil
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return Client.Set(ctx, PrefixMetrics+clientID, data, ttl).Err()
}

// GetMachineMetrics returns metrics from Redis.
// If key does not exist, returns nil, nil.
func GetMachineMetrics(clientID string) (*MachineMetrics, error) {
	if clientID == "" {
		return nil, nil
	}
	data, err := Client.Get(ctx, PrefixMetrics+clientID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var metrics MachineMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, err
	}
	return &metrics, nil
}
