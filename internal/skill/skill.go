package skill

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Pack is one operational skill: diagnosis path + prompt hints.
type Pack struct {
	Name           string   `yaml:"name"`
	DisplayName    string   `yaml:"display_name"`
	Topics         []string `yaml:"topics"`
	MatchKeywords  []string `yaml:"match_keywords"`
	Input          []string `yaml:"input"`
	AnalysisSteps  []string `yaml:"analysis_steps"`
	OutputFormat   []string `yaml:"output_format"`
	ExtraGuidance  string   `yaml:"extra_guidance,omitempty"`
	PromptTemplate string   `yaml:"prompt_template,omitempty"`
}

// Registry holds loaded skill packs.
type Registry struct {
	Packs []Pack
}

// LoadDir reads all *.yaml under dir from fs.FS.
func LoadDir(fsys fs.FS, dir string) (*Registry, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read skills dir: %w", err)
	}
	var packs []Pack
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var p Pack
		if err := yaml.Unmarshal(b, &p); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if p.Name == "" {
			continue
		}
		packs = append(packs, p)
	}
	return &Registry{Packs: packs}, nil
}

// MatchTopic picks the best skill for a CLI topic (e.g. "kafka", "k8s").
func (r *Registry) MatchTopic(topic string) *Pack {
	topic = strings.ToLower(strings.TrimSpace(topic))
	var best *Pack
	bestScore := -1
	for i := range r.Packs {
		p := &r.Packs[i]
		for _, t := range p.Topics {
			if strings.EqualFold(t, topic) {
				return p
			}
		}
		score := 0
		for _, t := range p.Topics {
			if strings.Contains(topic, strings.ToLower(t)) || strings.Contains(strings.ToLower(t), topic) {
				score += 3
			}
		}
		for _, kw := range p.MatchKeywords {
			if strings.Contains(topic, strings.ToLower(kw)) {
				score += 1
			}
		}
		if score > bestScore {
			bestScore = score
			best = p
		}
	}
	if best != nil && bestScore > 0 {
		return best
	}
	// fallback: substring match on name
	for i := range r.Packs {
		p := &r.Packs[i]
		if strings.Contains(strings.ToLower(p.Name), topic) {
			return p
		}
	}
	if len(r.Packs) == 0 {
		return nil
	}
	return &r.Packs[0]
}

// ByName returns a pack by its YAML name field.
func (r *Registry) ByName(name string) *Pack {
	for i := range r.Packs {
		if r.Packs[i].Name == name {
			return &r.Packs[i]
		}
	}
	return nil
}

// MatchAnalyze resolves skill for `analyze <topic>` plus CLI context (e.g. k8s + pod=pending).
func (r *Registry) MatchAnalyze(topic string, ctx map[string]string) *Pack {
	t := strings.ToLower(strings.TrimSpace(topic))
	if t == "k8s" {
		pod := strings.ToLower(ctx["pod"])
		issue := strings.ToLower(ctx["issue"])
		if strings.Contains(pod, "pending") || issue == "pending" {
			if p := r.ByName("k8s_pod_pending"); p != nil {
				return p
			}
		}
		if strings.Contains(pod, "crash") || issue == "crashloop" || issue == "crashloopbackoff" {
			if p := r.ByName("k8s_crashloop"); p != nil {
				return p
			}
		}
	}
	return r.MatchTopic(topic)
}

// MatchQuery picks a skill for free-text (ask / runbook) using keyword overlap.
func (r *Registry) MatchQuery(query string) *Pack {
	q := strings.ToLower(query)
	var best *Pack
	bestScore := -1
	for i := range r.Packs {
		p := &r.Packs[i]
		score := 0
		for _, kw := range p.MatchKeywords {
			if kw != "" && strings.Contains(q, strings.ToLower(kw)) {
				score += 2
			}
		}
		for _, t := range p.Topics {
			if t != "" && strings.Contains(q, strings.ToLower(t)) {
				score += 1
			}
		}
		if score > bestScore {
			bestScore = score
			best = p
		}
	}
	if best != nil && bestScore > 0 {
		return best
	}
	if len(r.Packs) == 0 {
		return nil
	}
	return &r.Packs[0]
}
