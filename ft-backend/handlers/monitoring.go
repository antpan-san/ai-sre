package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ---- Monitoring Configs ----

// GetMonitoringConfigList returns all monitoring configurations.
func GetMonitoringConfigList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	configType := c.Query("type")

	offset := (page - 1) * pageSize
	db := database.DB.Model(&models.MonitoringConfig{})

	if configType != "" {
		db = db.Where("type = ?", configType)
	}

	var total int64
	db.Count(&total)

	var configs []models.MonitoringConfig
	db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&configs)

	list := make([]gin.H, 0, len(configs))
	for _, cfg := range configs {
		list = append(list, monitoringConfigDTO(cfg, ""))
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": list, "total": total}, "msg": "success"})
}

// GetMonitoringConfig returns a single monitoring configuration.
func GetMonitoringConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的配置ID"})
		return
	}

	var config models.MonitoringConfig
	if err := database.DB.Where("id = ?", id).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "配置不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": monitoringConfigDTO(config, ""), "msg": "success"})
}

// CreateMonitoringConfig creates a new monitoring configuration and optionally dispatches an install task.
func CreateMonitoringConfig(c *gin.Context) {
	config, raw, err := bindMonitoringConfig(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	if err := database.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建配置失败"})
		return
	}

	taskID := ""
	if strings.TrimSpace(config.MachineID) != "" {
		task, err := createMonitoringInstallTask(c, []string{config.MachineID}, config.Type, raw)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 200, "data": monitoringConfigDTO(config, ""), "msg": "配置已保存，安装任务创建失败: " + err.Error()})
			return
		}
		taskID = task.ID.String()
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": monitoringConfigDTO(config, taskID), "msg": "创建成功"})
}

// UpdateMonitoringConfig updates an existing monitoring configuration.
func UpdateMonitoringConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的配置ID"})
		return
	}

	var existing models.MonitoringConfig
	if err := database.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "配置不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询配置失败"})
		return
	}

	config, _, err := bindMonitoringConfig(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	config.ID = id
	config.TenantID = existing.TenantID
	if err := database.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": monitoringConfigDTO(config, ""), "msg": "更新成功"})
}

// DeleteMonitoringConfig deletes a monitoring configuration.
func DeleteMonitoringConfig(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的配置ID"})
		return
	}

	if err := database.DB.Where("id = ?", id).Delete(&models.MonitoringConfig{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}

// InstallMonitoring dispatches exporter/prometheus installation tasks without creating a saved config first.
func InstallMonitoring(c *gin.Context) {
	var req struct {
		MachineIDs []string               `json:"machine_ids" binding:"required"`
		Type       string                 `json:"type" binding:"required"`
		Config     map[string]interface{} `json:"config"`
		Name       string                 `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}
	cfg := req.Config
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	cfg["type"] = req.Type
	cfg["name"] = req.Name
	task, err := createMonitoringInstallTask(c, req.MachineIDs, req.Type, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建监控安装任务失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"task_id": task.ID.String(), "status": task.Status}, "msg": "监控安装任务已创建"})
}

func bindMonitoringConfig(c *gin.Context) (models.MonitoringConfig, map[string]interface{}, error) {
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		return models.MonitoringConfig{}, nil, err
	}
	name := stringField(raw, "name", "")
	cfgType := normalizeMonitoringType(stringField(raw, "type", ""))
	if name == "" || cfgType == "" {
		return models.MonitoringConfig{}, nil, fmt.Errorf("missing name or type")
	}
	status := "inactive"
	if boolField(raw, "enabled", false) || stringField(raw, "status", "") == "active" {
		status = "active"
	}
	machineID := stringField(raw, "machine_id", "")
	if machineID == "" {
		machineID = stringField(raw, "machineId", "")
	}
	configRaw, _ := json.Marshal(raw)
	return models.MonitoringConfig{
		Name:        name,
		Type:        cfgType,
		Status:      status,
		Config:      models.JSONB(configRaw),
		Description: stringField(raw, "description", ""),
		MachineID:   machineID,
	}, raw, nil
}

func monitoringConfigDTO(cfg models.MonitoringConfig, taskID string) gin.H {
	out := gin.H{}
	var extra map[string]interface{}
	_ = json.Unmarshal(cfg.Config, &extra)
	for k, v := range extra {
		out[k] = v
	}
	out["id"] = cfg.ID.String()
	out["name"] = cfg.Name
	out["type"] = cfg.Type
	out["status"] = cfg.Status
	out["enabled"] = cfg.Status == "active"
	out["description"] = cfg.Description
	out["machine_id"] = cfg.MachineID
	out["machineId"] = cfg.MachineID
	out["createTime"] = cfg.CreatedAt.Format("2006-01-02 15:04:05")
	out["updateTime"] = cfg.UpdatedAt.Format("2006-01-02 15:04:05")
	if taskID != "" {
		out["task_id"] = taskID
	}
	return out
}

func createMonitoringInstallTask(c *gin.Context, machineIDs []string, typ string, cfg map[string]interface{}) (*models.Task, error) {
	typ = normalizeMonitoringType(typ)
	script, err := monitoringInstallScript(typ, cfg)
	if err != nil {
		return nil, err
	}
	return createManagedTask(c, managedTaskRequest{
		Name:        "监控安装: " + typ,
		Type:        string(models.TaskTypeInstallMonitor),
		Command:     "install_monitor",
		Description: "安装或刷新 " + typ + " 监控组件",
		MachineIDs:  machineIDs,
		Payload:     map[string]interface{}{"script": script, "type": typ, "config": cfg},
		TimeoutSec:  600,
		MaxRetry:    1,
	})
}

func monitoringInstallScript(typ string, cfg map[string]interface{}) (string, error) {
	switch typ {
	case "node-exporter", "node_exporter":
		port := intField(cfg, "port", 9100)
		return exporterSystemdScript("node_exporter", "node_exporter", port, ""), nil
	case "redis-exporter", "redis_exporter":
		port := intField(cfg, "port", 9121)
		redisAddr := stringField(cfg, "redisAddr", "redis://127.0.0.1:6379")
		return exporterSystemdScript("redis_exporter", "redis_exporter", port, "--redis.addr "+shellQuote(redisAddr)), nil
	case "blackbox-exporter", "blackbox_exporter":
		port := intField(cfg, "port", 9115)
		return exporterSystemdScript("blackbox_exporter", "blackbox_exporter", port, ""), nil
	case "prometheus":
		return prometheusReloadScript(cfg), nil
	default:
		return "", fmt.Errorf("unsupported monitoring type %s", typ)
	}
}

func exporterSystemdScript(service, binary string, port int, extraArgs string) string {
	return fmt.Sprintf(`set -euo pipefail
BIN="$(command -v %[2]s || true)"
if [ -z "$BIN" ]; then
  if command -v apt-get >/dev/null 2>&1; then apt-get update && apt-get install -y %[2]s || true; fi
  if command -v yum >/dev/null 2>&1; then yum install -y %[2]s || true; fi
  BIN="$(command -v %[2]s || true)"
fi
if [ -z "$BIN" ]; then
  echo "%[2]s binary not found; install binary or package first" >&2
  exit 2
fi
cat >/etc/systemd/system/%[1]s.service <<EOF_UNIT
[Unit]
Description=OpsFleet %[1]s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$BIN --web.listen-address=:%[3]d %[4]s
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF_UNIT
systemctl daemon-reload
systemctl enable --now %[1]s.service
systemctl --no-pager status %[1]s.service || true
`, service, binary, port, extraArgs)
}

func prometheusReloadScript(cfg map[string]interface{}) string {
	config := stringField(cfg, "prometheus_config", "")
	if config == "" {
		config = "global:\n  scrape_interval: 15s\nscrape_configs:\n  - job_name: opsfleet-node\n    static_configs:\n      - targets: ['127.0.0.1:9100']\n"
	}
	return fmt.Sprintf(`set -euo pipefail
mkdir -p /etc/prometheus
cat >/etc/prometheus/prometheus.yml <<'EOF_PROM'
%s
EOF_PROM
if systemctl list-unit-files | grep -q '^prometheus.service'; then
  systemctl reload prometheus || systemctl restart prometheus
else
  echo "prometheus.service not found; config generated at /etc/prometheus/prometheus.yml"
fi
`, config)
}

func normalizeMonitoringType(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

func stringField(m map[string]interface{}, key, def string) string {
	if v, ok := m[key]; ok && v != nil {
		return strings.TrimSpace(fmt.Sprint(v))
	}
	return def
}

func intField(m map[string]interface{}, key string, def int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case string:
			if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
				return i
			}
		}
	}
	return def
}

func boolField(m map[string]interface{}, key string, def bool) bool {
	if v, ok := m[key]; ok {
		switch b := v.(type) {
		case bool:
			return b
		case string:
			return b == "true" || b == "1" || b == "active"
		}
	}
	return def
}

// ---- Alert Rules ----

// GetAlertRules returns all alert rules.
func GetAlertRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	offset := (page - 1) * pageSize
	db := database.DB.Model(&models.AlertRule{})

	var total int64
	db.Count(&total)

	var rules []models.AlertRule
	db.Limit(pageSize).Offset(offset).Order("created_at DESC").Find(&rules)

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": gin.H{"list": rules, "total": total}, "msg": "success"})
}

// CreateAlertRule creates a new alert rule.
func CreateAlertRule(c *gin.Context) {
	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建告警规则失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": rule, "msg": "创建成功"})
}

// UpdateAlertRule updates an existing alert rule.
func UpdateAlertRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的规则ID"})
		return
	}

	var existing models.AlertRule
	if err := database.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "规则不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "查询规则失败"})
		return
	}

	var rule models.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的请求参数"})
		return
	}

	rule.ID = id
	rule.TenantID = existing.TenantID
	if err := database.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "更新规则失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": rule, "msg": "更新成功"})
}

// DeleteAlertRule deletes an alert rule.
func DeleteAlertRule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的规则ID"})
		return
	}

	if err := database.DB.Where("id = ?", id).Delete(&models.AlertRule{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除规则失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "删除成功"})
}
