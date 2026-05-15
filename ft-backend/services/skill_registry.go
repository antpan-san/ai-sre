package services

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ft-backend/common/config"
	"ft-backend/skills"

	"gopkg.in/yaml.v3"
)

// SkillSource indicates how a registered skill was loaded.
type SkillSource string

const (
	SkillSourceBuiltin   SkillSource = "builtin"
	SkillSourceGenerated SkillSource = "generated"
)

// RegisteredSkill is a SkillPack annotated with provenance for the registry.
type RegisteredSkill struct {
	Pack    SkillPack   `json:"pack"`
	Source  SkillSource `json:"source"`
	Version string      `json:"version"`
	Path    string      `json:"path,omitempty"`
}

// SkillSummary is the public listing form for /api/ai/skills.
type SkillSummary struct {
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name"`
	Topics      []string    `json:"topics"`
	Source      SkillSource `json:"source"`
	Version     string      `json:"version"`
	Path        string      `json:"path,omitempty"`
}

// DiagnoseSample is the anonymized record appended after each successful diagnose.
type DiagnoseSample struct {
	Time        time.Time         `json:"time"`
	Topic       string            `json:"topic"`
	SkillName   string            `json:"skill_name,omitempty"`
	Style       string            `json:"style,omitempty"`
	UserContext map[string]string `json:"user_context,omitempty"`
	EvidenceKey []string          `json:"evidence_keys,omitempty"`
	AnswerHead  string            `json:"answer_head,omitempty"`
	AnswerTail  string            `json:"answer_tail,omitempty"`
	AnswerLen   int               `json:"answer_len,omitempty"`
	RequestID   string            `json:"request_id,omitempty"`
}

// SkillFeedback is what a client sends back after a diagnose.
type SkillFeedback struct {
	Time      time.Time `json:"time"`
	Topic     string    `json:"topic"`
	SkillName string    `json:"skill_name,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Helpful   *bool     `json:"helpful,omitempty"`
	Note      string    `json:"note,omitempty"`
}

// SkillRegistry holds builtin and generated skill packs and persists samples / feedback.
type SkillRegistry struct {
	mu        sync.RWMutex
	builtin   map[string]*RegisteredSkill // topic -> pack
	generated map[string]*RegisteredSkill // topic -> pack (overrides builtin)
	byName    map[string]*RegisteredSkill
	dataDir   string
}

var (
	defaultSkillRegistry     *SkillRegistry
	defaultSkillRegistryOnce sync.Once
)

// DefaultSkillRegistry returns a process-wide registry, lazily initialized.
func DefaultSkillRegistry() *SkillRegistry {
	defaultSkillRegistryOnce.Do(func() {
		r, err := NewSkillRegistry(ResolveSkillDataDir())
		if err != nil {
			r = &SkillRegistry{builtin: map[string]*RegisteredSkill{}, generated: map[string]*RegisteredSkill{}, byName: map[string]*RegisteredSkill{}}
		}
		defaultSkillRegistry = r
	})
	return defaultSkillRegistry
}

// ResolveSkillDataDir picks a writable directory for samples / feedback / generated packs.
// Override with OPSFLEET_AI_SKILL_DATA_DIR.
func ResolveSkillDataDir() string {
	if v := strings.TrimSpace(config.ResolvedAISkillDataDir()); v != "" {
		return v
	}
	for _, candidate := range []string{"/var/lib/opsfleet/ai-skills", "./data/ai-skills"} {
		if err := os.MkdirAll(candidate, 0o755); err == nil {
			return candidate
		}
	}
	return ""
}

// NewSkillRegistry constructs a fresh registry and loads builtin + generated packs.
func NewSkillRegistry(dataDir string) (*SkillRegistry, error) {
	r := &SkillRegistry{
		builtin:   map[string]*RegisteredSkill{},
		generated: map[string]*RegisteredSkill{},
		byName:    map[string]*RegisteredSkill{},
		dataDir:   strings.TrimSpace(dataDir),
	}
	if err := r.loadBuiltin(); err != nil {
		return r, fmt.Errorf("load builtin skills: %w", err)
	}
	if r.dataDir != "" {
		_ = r.loadGenerated() // best-effort; missing dir is fine
	}
	return r, nil
}

// DataDir returns the configured writable data dir (may be empty when unconfigured).
func (r *SkillRegistry) DataDir() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dataDir
}

func (r *SkillRegistry) loadBuiltin() error {
	entries, err := fs.ReadDir(skills.BuiltinFS, skills.BuiltinDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".yaml") {
			continue
		}
		raw, err := fs.ReadFile(skills.BuiltinFS, skills.BuiltinDir+"/"+e.Name())
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", e.Name(), err)
		}
		pack, err := unmarshalSkillPack(raw)
		if err != nil {
			return fmt.Errorf("parse embedded %s: %w", e.Name(), err)
		}
		if !ValidateSkillDraft(pack) {
			return fmt.Errorf("embedded skill %s failed validation", e.Name())
		}
		rs := &RegisteredSkill{Pack: *pack, Source: SkillSourceBuiltin, Version: "builtin", Path: "embed:" + skills.BuiltinDir + "/" + e.Name()}
		r.indexLocked(rs, /*replaceGenerated*/ false)
	}
	return nil
}

func (r *SkillRegistry) loadGenerated() error {
	dir := filepath.Join(r.dataDir, "generated")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".yaml") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		pack, err := unmarshalSkillPack(raw)
		if err != nil || !ValidateSkillDraft(pack) {
			continue
		}
		rs := &RegisteredSkill{Pack: *pack, Source: SkillSourceGenerated, Version: timestampVersion(full), Path: full}
		r.indexLocked(rs, true)
	}
	return nil
}

func (r *SkillRegistry) indexLocked(rs *RegisteredSkill, replaceGenerated bool) {
	if rs == nil {
		return
	}
	r.byName[rs.Pack.Name] = rs
	for _, t := range rs.Pack.Topics {
		key := strings.ToLower(strings.TrimSpace(t))
		if key == "" {
			continue
		}
		switch rs.Source {
		case SkillSourceBuiltin:
			if _, exists := r.builtin[key]; !exists {
				r.builtin[key] = rs
			}
		case SkillSourceGenerated:
			if replaceGenerated {
				r.generated[key] = rs
			} else if _, exists := r.generated[key]; !exists {
				r.generated[key] = rs
			}
		}
	}
}

// Match resolves a registered skill for the given topic, preferring generated over builtin.
func (r *SkillRegistry) Match(topic string, _ map[string]string) *RegisteredSkill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := strings.ToLower(strings.TrimSpace(topic))
	if key == "" {
		return nil
	}
	if gen, ok := r.generated[key]; ok {
		return gen
	}
	if bi, ok := r.builtin[key]; ok {
		return bi
	}
	return nil
}

// LookupByName returns a registered skill by name (generated takes precedence).
func (r *SkillRegistry) LookupByName(name string) *RegisteredSkill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byName[name]
}

// LookupErrorCode resolves a structured deploy/runtime error code to its root-cause card,
// searching across all registered skills (generated overrides builtin via byName).
// Returns the matched code entry and the skill that owns it.
func (r *SkillRegistry) LookupErrorCode(code string) (*SkillErrorCode, *RegisteredSkill) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil
	}
	upper := strings.ToUpper(code)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rs := range r.byName {
		for i := range rs.Pack.ErrorCodes {
			ec := &rs.Pack.ErrorCodes[i]
			if strings.ToUpper(strings.TrimSpace(ec.Code)) == upper {
				return ec, rs
			}
		}
	}
	return nil, nil
}

// ListErrorCodes returns all structured error code entries across all registered skills.
// Sorted by code; later entries with the same code (from generated) win.
func (r *SkillRegistry) ListErrorCodes() []SkillErrorCode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := map[string]SkillErrorCode{}
	// builtin first
	for _, rs := range r.builtin {
		for _, ec := range rs.Pack.ErrorCodes {
			seen[strings.ToUpper(strings.TrimSpace(ec.Code))] = ec
		}
	}
	// then generated overrides (later wins)
	for _, rs := range r.generated {
		for _, ec := range rs.Pack.ErrorCodes {
			seen[strings.ToUpper(strings.TrimSpace(ec.Code))] = ec
		}
	}
	out := make([]SkillErrorCode, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// List returns sorted summaries of all registered skills.
func (r *SkillRegistry) List() []SkillSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()
	uniq := map[string]*RegisteredSkill{}
	for _, rs := range r.byName {
		uniq[rs.Pack.Name] = rs
	}
	// also include topic-level skills not in byName (shouldn't happen but defensive)
	for _, rs := range r.generated {
		uniq[rs.Pack.Name] = rs
	}
	for _, rs := range r.builtin {
		if _, ok := uniq[rs.Pack.Name]; !ok {
			uniq[rs.Pack.Name] = rs
		}
	}
	out := make([]SkillSummary, 0, len(uniq))
	for _, rs := range uniq {
		out = append(out, SkillSummary{
			Name:        rs.Pack.Name,
			DisplayName: rs.Pack.DisplayName,
			Topics:      rs.Pack.Topics,
			Source:      rs.Source,
			Version:     rs.Version,
			Path:        rs.Path,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// SaveGenerated archives the previous generated pack (if any) and writes a new latest.
// Returns the path of the new generated YAML and reloads the registry index.
func (r *SkillRegistry) SaveGenerated(pack *SkillPack) (string, error) {
	if !ValidateSkillDraft(pack) {
		return "", errors.New("invalid skill pack")
	}
	if r.dataDir == "" {
		return "", errors.New("skill data dir not configured")
	}
	dir := filepath.Join(r.dataDir, "generated")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	topic := strings.ToLower(strings.TrimSpace(pack.Topics[0]))
	if topic == "" {
		return "", errors.New("skill has no topics")
	}
	newPath := filepath.Join(dir, topic+".yaml")

	// archive previous
	if oldRaw, err := os.ReadFile(newPath); err == nil && len(oldRaw) > 0 {
		archDir := filepath.Join(dir, topic+".history")
		_ = os.MkdirAll(archDir, 0o755)
		archPath := filepath.Join(archDir, time.Now().UTC().Format("2006-01-02T15-04-05Z")+".yaml")
		_ = os.WriteFile(archPath, oldRaw, 0o644)
	}

	raw, err := yaml.Marshal(pack)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(newPath, raw, 0o644); err != nil {
		return "", err
	}

	r.mu.Lock()
	rs := &RegisteredSkill{Pack: *pack, Source: SkillSourceGenerated, Version: timestampVersion(newPath), Path: newPath}
	r.indexLocked(rs, true)
	r.mu.Unlock()

	return newPath, nil
}

// AppendSample writes a sample line to samples/<topic>.jsonl. Best-effort; never blocks long.
func (r *SkillRegistry) AppendSample(s DiagnoseSample) error {
	dir := r.DataDir()
	if dir == "" {
		return nil
	}
	if s.Time.IsZero() {
		s.Time = time.Now().UTC()
	}
	topic := strings.ToLower(strings.TrimSpace(s.Topic))
	if topic == "" {
		topic = "_unknown"
	}
	full := filepath.Join(dir, "samples", topic+".jsonl")
	return appendJSONLine(full, s)
}

// AppendFeedback writes a feedback line to feedback/<topic>.jsonl.
func (r *SkillRegistry) AppendFeedback(f SkillFeedback) error {
	dir := r.DataDir()
	if dir == "" {
		return nil
	}
	if f.Time.IsZero() {
		f.Time = time.Now().UTC()
	}
	topic := strings.ToLower(strings.TrimSpace(f.Topic))
	if topic == "" {
		topic = "_unknown"
	}
	full := filepath.Join(dir, "feedback", topic+".jsonl")
	return appendJSONLine(full, f)
}

// ReadRecentSamples returns up to n most recent diagnose samples for a topic.
func (r *SkillRegistry) ReadRecentSamples(topic string, n int) ([]DiagnoseSample, error) {
	dir := r.DataDir()
	if dir == "" {
		return nil, nil
	}
	full := filepath.Join(dir, "samples", strings.ToLower(strings.TrimSpace(topic))+".jsonl")
	lines, err := readRecentJSONLines(full, n)
	if err != nil {
		return nil, err
	}
	out := make([]DiagnoseSample, 0, len(lines))
	for _, ln := range lines {
		var s DiagnoseSample
		if json.Unmarshal([]byte(ln), &s) == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

// ReadRecentFeedback returns up to n most recent feedback lines for a topic.
func (r *SkillRegistry) ReadRecentFeedback(topic string, n int) ([]SkillFeedback, error) {
	dir := r.DataDir()
	if dir == "" {
		return nil, nil
	}
	full := filepath.Join(dir, "feedback", strings.ToLower(strings.TrimSpace(topic))+".jsonl")
	lines, err := readRecentJSONLines(full, n)
	if err != nil {
		return nil, err
	}
	out := make([]SkillFeedback, 0, len(lines))
	for _, ln := range lines {
		var s SkillFeedback
		if json.Unmarshal([]byte(ln), &s) == nil {
			out = append(out, s)
		}
	}
	return out, nil
}

func unmarshalSkillPack(raw []byte) (*SkillPack, error) {
	var p SkillPack
	if err := yaml.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func timestampVersion(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return time.Now().UTC().Format("2006-01-02T15:04:05Z")
	}
	return info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
}

func appendJSONLine(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(raw, '\n')); err != nil {
		return err
	}
	return nil
}

// readRecentJSONLines reads up to maxLines from the tail of a JSONL file.
// Uses a simple buffered tail-read; bounded by maxLines for memory safety.
func readRecentJSONLines(path string, maxLines int) ([]string, error) {
	if maxLines <= 0 {
		maxLines = 50
	}
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	ring := make([]string, 0, maxLines)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if len(ring) >= maxLines {
			ring = ring[1:]
		}
		ring = append(ring, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return ring, nil
}
