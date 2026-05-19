package cli

import "strings"

// registeredCheckTopics lists canonical check topics (first batch).
var registeredCheckTopics = map[string]struct{}{
	"redis":         {},
	"linux":         {},
	"domain":        {},
	"k8s":           {},
	"go":            {},
	"kafka":         {},
	"mysql":         {},
	"postgresql":    {},
	"nginx":         {},
	"elasticsearch": {},
	"code":          {},
}

// normalizeCheckTopicAlias maps user-facing aliases to canonical topic names.
func normalizeCheckTopicAlias(topic string) string {
	t := strings.ToLower(strings.TrimSpace(topic))
	switch t {
	case "postgres":
		return "postgresql"
	case "es":
		return "elasticsearch"
	case "dns", "url":
		return "domain"
	case "system", "host":
		return "linux"
	case "kubernetes":
		return "k8s"
	default:
		return t
	}
}

func isRegisteredCheckTopic(topic string) bool {
	_, ok := registeredCheckTopics[normalizeCheckTopicAlias(topic)]
	return ok
}

func publicCheckTopicList() []string {
	return []string{
		"redis", "linux", "domain", "k8s", "go", "kafka",
		"mysql", "postgresql", "nginx", "elasticsearch", "code",
	}
}

func checkTopicRequiresTarget(topic string) bool {
	switch normalizeCheckTopicAlias(topic) {
	case "code":
		return true
	default:
		return false
	}
}

func checkTopicAcceptsOptionalTarget(topic string) bool {
	t := normalizeCheckTopicAlias(topic)
	if _, ok := checkTargetSpecs[t]; ok {
		return true
	}
	switch t {
	case "domain", "go", "k8s", "linux", "nginx", "code":
		return true
	default:
		return false
	}
}
