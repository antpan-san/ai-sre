package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"ft-backend/models"
)

// serviceProductNameForDashboard resolves 功能名称: platform catalog / product hints in config,
// then description, then a readable default from deploy type.
func serviceProductNameForDashboard(s *models.Service) string {
	if s == nil {
		return "—"
	}
	if v := extractProductNameFromServiceConfig(s.Config); v != "" {
		return v
	}
	if d := strings.TrimSpace(s.Description); d != "" {
		return d
	}
	return defaultServiceTypeProductLabel(s.Type)
}

func extractProductNameFromServiceConfig(cfg models.JSONB) string {
	if len(cfg) == 0 || string(cfg) == "null" {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(cfg, &m); err != nil {
		return ""
	}
	keys := []string{
		"productName", "product_name",
		"featureName", "feature_name",
		"catalogService", "catalog_service",
		"serviceKey", "service_key",
		"platformService", "platform_service",
		"component",
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok {
				if t := strings.TrimSpace(s); t != "" {
					return t
				}
			}
		}
	}
	// generic "service" only if it looks like a component key (short slug), not a sentence
	if v, ok := m["service"]; ok {
		if s, ok := v.(string); ok {
			t := strings.TrimSpace(s)
			if t != "" && len(t) <= 64 && !strings.ContainsAny(t, "\n\r") {
				return t
			}
		}
	}
	return ""
}

func defaultServiceTypeProductLabel(typ string) string {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "docker":
		return "容器应用"
	case "k8s", "kubernetes":
		return "Kubernetes 工作负载"
	case "linux":
		return "Linux 服务"
	case "":
		return "—"
	default:
		return typ
	}
}

// serviceResourceSummaryForDashboard builds the 资源 column: image / host hint / port / type fallback.
func serviceResourceSummaryForDashboard(s *models.Service) string {
	if s == nil {
		return "—"
	}
	var parts []string
	if img := strings.TrimSpace(s.Image); img != "" {
		parts = append(parts, img)
	}
	if mid := strings.TrimSpace(s.MachineID); mid != "" {
		parts = append(parts, "主机 "+shortIDForDisplay(mid))
	}
	if s.Port > 0 {
		parts = append(parts, fmt.Sprintf("端口 %d", s.Port))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " · ")
	}
	t := strings.TrimSpace(s.Type)
	if t != "" && !strings.EqualFold(t, "docker") {
		return defaultServiceTypeProductLabel(t)
	}
	return "—"
}

func shortIDForDisplay(id string) string {
	id = strings.TrimSpace(id)
	if len(id) <= 14 {
		return id
	}
	return id[:8] + "…"
}
