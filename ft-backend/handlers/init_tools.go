package handlers

import (
	"encoding/json"
	"net/http"

	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
)

// ---- System Parameter Optimization ----

// GetSystemParams returns current system parameter configuration.
func GetSystemParams(c *gin.Context) {
	// Return default recommended kernel parameters
	params := map[string]interface{}{
		"vm.swappiness":                        10,
		"net.core.somaxconn":                   65535,
		"net.ipv4.tcp_max_syn_backlog":         65535,
		"net.ipv4.ip_forward":                  1,
		"net.bridge.bridge-nf-call-iptables":   1,
		"net.bridge.bridge-nf-call-ip6tables":  1,
		"fs.file-max":                          1048576,
		"fs.inotify.max_user_watches":          524288,
		"net.ipv4.tcp_keepalive_time":          600,
		"net.ipv4.tcp_keepalive_intvl":         30,
		"net.ipv4.tcp_keepalive_probes":        10,
		"kernel.pid_max":                       65535,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": params,
		"msg":  "success",
	})
}

// ApplySystemParams creates tasks to apply system parameter optimization on selected machines.
func ApplySystemParams(c *gin.Context) {
	var req struct {
		MachineIDs []string               `json:"machine_ids" binding:"required"`
		Params     map[string]interface{} `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	username, _ := c.Get("username")

	// Build the sysctl script with input sanitization
	script := "#!/bin/bash\nset -e\n"
	if req.Params != nil {
		for k, v := range req.Params {
			// Sanitize: only allow safe sysctl key characters (alphanumeric, dots, underscores)
			if !isSafeSysctlKey(k) {
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "非法的参数名: " + k})
				return
			}
			valStr := toString(v)
			if !isSafeSysctlValue(valStr) {
				c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "非法的参数值: " + valStr})
				return
			}
			script += "sysctl -w " + k + "=" + valStr + "\n"
		}
	}
	script += "sysctl -p\necho 'System parameters optimized successfully'\n"

	task, err := createInitTask(username.(string), "系统参数优化", string(models.TaskTypeSysInit),
		req.MachineIDs, map[string]interface{}{"script": script})
	if err != nil {
		logger.Error("创建系统参数优化任务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"task_id": task.ID.String()},
		"msg":  "系统参数优化任务已创建",
	})
}

// ---- Time Synchronization ----

// ApplyTimeSync creates tasks to configure time synchronization on selected machines.
func ApplyTimeSync(c *gin.Context) {
	var req struct {
		MachineIDs []string `json:"machine_ids" binding:"required"`
		NTPServer  string   `json:"ntp_server"`
		Timezone   string   `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	if req.NTPServer == "" {
		req.NTPServer = "ntp.aliyun.com"
	}
	if req.Timezone == "" {
		req.Timezone = "Asia/Shanghai"
	}

	username, _ := c.Get("username")

	script := `#!/bin/bash
set -e
# Install chrony if not present
if ! command -v chronyd &> /dev/null; then
    if command -v yum &> /dev/null; then
        yum install -y chrony
    elif command -v apt-get &> /dev/null; then
        apt-get install -y chrony
    fi
fi
# Configure NTP server
sed -i '/^server /d; /^pool /d' /etc/chrony.conf 2>/dev/null || true
echo "server ` + req.NTPServer + ` iburst" >> /etc/chrony.conf
# Set timezone
timedatectl set-timezone ` + req.Timezone + `
# Restart chrony
systemctl enable chronyd
systemctl restart chronyd
# Force sync
chronyc makestep
echo "Time synchronization configured successfully"
`
	task, err := createInitTask(username.(string), "时间同步配置", string(models.TaskTypeTimeSync),
		req.MachineIDs, map[string]interface{}{"script": script, "ntp_server": req.NTPServer, "timezone": req.Timezone})
	if err != nil {
		logger.Error("创建时间同步任务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"task_id": task.ID.String()},
		"msg":  "时间同步任务已创建",
	})
}

// ---- Security Hardening ----

// ApplySecurityHarden creates tasks for security hardening on selected machines.
func ApplySecurityHarden(c *gin.Context) {
	var req struct {
		MachineIDs []string               `json:"machine_ids" binding:"required"`
		Options    map[string]interface{} `json:"options"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	username, _ := c.Get("username")

	script := `#!/bin/bash
set -e

# --- SSH Hardening ---
# Disable root login
sed -i 's/^#*PermitRootLogin.*/PermitRootLogin no/' /etc/ssh/sshd_config
# Disable password authentication (only if keys are set up)
# sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
# Limit SSH attempts
sed -i 's/^#*MaxAuthTries.*/MaxAuthTries 3/' /etc/ssh/sshd_config
# Disable empty passwords
sed -i 's/^#*PermitEmptyPasswords.*/PermitEmptyPasswords no/' /etc/ssh/sshd_config

# --- Firewall ---
if command -v firewall-cmd &> /dev/null; then
    systemctl enable firewalld
    systemctl start firewalld
    firewall-cmd --permanent --add-service=ssh
    firewall-cmd --reload
fi

# --- Disable unused services ---
for svc in telnet rsh rlogin rexec; do
    systemctl disable "$svc" 2>/dev/null || true
    systemctl stop "$svc" 2>/dev/null || true
done

# --- Set password policy ---
if [ -f /etc/login.defs ]; then
    sed -i 's/^PASS_MAX_DAYS.*/PASS_MAX_DAYS   90/' /etc/login.defs
    sed -i 's/^PASS_MIN_DAYS.*/PASS_MIN_DAYS   7/' /etc/login.defs
    sed -i 's/^PASS_MIN_LEN.*/PASS_MIN_LEN    12/' /etc/login.defs
fi

# Restart SSH
systemctl restart sshd

echo "Security hardening completed successfully"
`

	task, err := createInitTask(username.(string), "安全加固", string(models.TaskTypeSecurityHarden),
		req.MachineIDs, map[string]interface{}{"script": script})
	if err != nil {
		logger.Error("创建安全加固任务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"task_id": task.ID.String()},
		"msg":  "安全加固任务已创建",
	})
}

// ---- Disk Partition Optimization ----

// ApplyDiskOptimize creates tasks for disk partition optimization.
func ApplyDiskOptimize(c *gin.Context) {
	var req struct {
		MachineIDs []string               `json:"machine_ids" binding:"required"`
		Options    map[string]interface{} `json:"options"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	username, _ := c.Get("username")

	script := `#!/bin/bash
set -e

# --- Swap management ---
echo "Disabling swap..."
swapoff -a
sed -i '/\sswap\s/s/^/#/' /etc/fstab

# --- Check disk usage ---
echo "Current disk usage:"
df -h

# --- I/O scheduler optimization ---
for disk in $(lsblk -nd -o NAME); do
    current=$(cat /sys/block/$disk/queue/scheduler 2>/dev/null || echo "")
    if [ -n "$current" ]; then
        echo "mq-deadline" > /sys/block/$disk/queue/scheduler 2>/dev/null || true
    fi
done

echo "Disk optimization completed successfully"
`
	task, err := createInitTask(username.(string), "磁盘分区优化", string(models.TaskTypeDiskOptimize),
		req.MachineIDs, map[string]interface{}{"script": script})
	if err != nil {
		logger.Error("创建磁盘优化任务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"task_id": task.ID.String()},
		"msg":  "磁盘优化任务已创建",
	})
}

// ---- Helper Functions ----

// createInitTask is a helper to create init-tool tasks with sub-tasks.
func createInitTask(username, name, taskType string, machineIDs []string, payload map[string]interface{}) (*models.Task, error) {
	payloadJSON, _ := json.Marshal(payload)
	targetIDsJSON, _ := json.Marshal(machineIDs)

	task := models.Task{
		Name:       name,
		Type:       taskType,
		Status:     string(models.TaskStatusPending),
		CreatedBy:  username,
		Payload:    models.JSONB(payloadJSON),
		TargetIDs:  models.JSONB(targetIDsJSON),
		TotalCount: len(machineIDs),
		TimeoutSec: 600,
	}

	tx := database.DB.Begin()
	if err := tx.Create(&task).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, machineID := range machineIDs {
		var machine models.Machine
		if err := tx.Where("id = ?", machineID).First(&machine).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		subTask := models.SubTask{
			TaskID:    task.ID,
			MachineID: machine.ID,
			ClientID:  machine.IP,
			Command:   mapTaskTypeToCommand(taskType),
			Status:    string(models.TaskStatusPending),
			Payload:   models.JSONB(payloadJSON),
			MaxRetry:  1,
		}
		if err := tx.Create(&subTask).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	tx.Create(&models.TaskLog{
		TaskID:  task.ID,
		Level:   "info",
		Message: name + " 任务已创建",
	})

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// isSafeSysctlKey validates sysctl parameter names (only dots, letters, digits, underscores, hyphens).
func isSafeSysctlKey(key string) bool {
	if key == "" {
		return false
	}
	for _, ch := range key {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}

// isSafeSysctlValue validates sysctl values (only digits, dots, minus sign).
func isSafeSysctlValue(val string) bool {
	if val == "" {
		return false
	}
	for _, ch := range val {
		if !((ch >= '0' && ch <= '9') || ch == '.' || ch == '-') {
			return false
		}
	}
	return true
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		b, _ := json.Marshal(v)
		s := string(b)
		// Remove surrounding quotes for numeric values
		if len(s) > 1 && s[0] == '"' {
			s = s[1 : len(s)-1]
		}
		return s
	}
}
