package services

import (
	"strings"
)

// MergeSkillPackWithRegistry merges an approved diagnostic pack into the registry match for its topic.
// When no existing pack is registered, incoming is returned unchanged.
func MergeSkillPackWithRegistry(reg *SkillRegistry, incoming *SkillPack) (*SkillPack, bool) {
	if incoming == nil || len(incoming.Topics) == 0 {
		return incoming, false
	}
	if reg == nil {
		return incoming, false
	}
	topic := strings.ToLower(strings.TrimSpace(incoming.Topics[0]))
	base := reg.Match(topic, nil)
	if base == nil {
		return incoming, false
	}
	out := base.Pack
	out.Name = base.Pack.Name
	if strings.TrimSpace(out.DisplayName) == "" {
		out.DisplayName = incoming.DisplayName
	}
	out.AnalysisSteps = mergeStringLists(base.Pack.AnalysisSteps, incoming.AnalysisSteps, 16)
	out.MatchKeywords = mergeStringLists(base.Pack.MatchKeywords, incoming.MatchKeywords, 24)
	out.Input = mergeStringLists(base.Pack.Input, incoming.Input, 16)
	out.OutputFormat = mergeStringLists(base.Pack.OutputFormat, incoming.OutputFormat, 12)
	if len(out.OutputFormat) == 0 {
		out.OutputFormat = incoming.OutputFormat
	}
	eg := strings.TrimSpace(base.Pack.ExtraGuidance)
	inc := strings.TrimSpace(incoming.ExtraGuidance)
	if inc != "" {
		if eg != "" {
			eg += "\n\n---\n\n"
		}
		eg += inc
	}
	out.ExtraGuidance = eg
	return &out, true
}

func mergeStringLists(base, extra []string, max int) []string {
	out := mergeStringListsUnlimited(base, extra)
	if max > 0 && len(out) > max {
		out = out[:max]
	}
	return out
}

func mergeStringListsUnlimited(base, extra []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(base)+len(extra))
	for _, s := range append(base, extra...) {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
