package services

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ft-backend/common/config"
)

type autoRefineState struct {
	LastRefineAt       time.Time `json:"last_refine_at"`
	SamplesSinceRefine int       `json:"samples_since_refine"`
	RefinesDay         string    `json:"refines_day"`
	RefinesToday       int       `json:"refines_today"`
}

var autoRefineTopicLocks sync.Map // topic -> *sync.Mutex

func loadAutoRefineConfig() config.ResolvedSkillAutoRefine {
	return config.ResolvedSkillAutoRefineConfig()
}

// AppendDiagnoseSample appends a diagnose sample and may trigger background auto-refine.
func AppendDiagnoseSample(reg *SkillRegistry, s DiagnoseSample) error {
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	if s.Time.IsZero() {
		s.Time = time.Now().UTC()
	}
	if s.EnhancementReview == nil {
		topic := strings.ToLower(strings.TrimSpace(s.Topic))
		if topic != "" && !strings.HasPrefix(topic, "_") {
			review := EvaluateSkillEnhancement(reg, PostAICallRecord{
				Topic:        topic,
				CommandKind:  s.CommandKind,
				SkillName:    s.SkillName,
				PackKey:      s.PackKey,
				ProblemKey:   s.ProblemKey,
				Style:        s.Style,
				RequestID:    s.RequestID,
				Answer:       s.AnswerHead + "\n" + s.AnswerTail,
				UserContext:  s.UserContext,
				EvidenceKeys: s.EvidenceKey,
				MatchedSkill: s.SkillName != "" && !strings.Contains(s.SkillName, "_auto"),
			})
			s.EnhancementReview = &review
		}
	}
	if err := reg.AppendSample(s); err != nil {
		return err
	}
	topic := strings.ToLower(strings.TrimSpace(s.Topic))
	if topic == "" {
		return nil
	}
	if err := incrementAutoRefineSampleCounter(reg, topic); err != nil {
		log.Printf("skill auto-refine: increment state topic=%s: %v", topic, err)
	}
	if s.EnhancementReview != nil && s.EnhancementReview.Priority == "high" {
		_ = incrementAutoRefineSampleCounter(reg, topic)
	}
	if s.EnhancementReview != nil && s.EnhancementReview.NeedsEnhancement {
		_ = appendEnhancementReviewLog(reg, *s.EnhancementReview)
	}
	go MaybeAutoRefine(reg, topic)
	return nil
}

func incrementAutoRefineSampleCounter(reg *SkillRegistry, topic string) error {
	mu := autoRefineMutex(topic)
	mu.Lock()
	defer mu.Unlock()
	st, err := readAutoRefineState(reg, topic)
	if err != nil {
		return err
	}
	st.SamplesSinceRefine++
	return writeAutoRefineState(reg, topic, st)
}

// MaybeAutoRefine runs a refine pass when thresholds and cooldown are satisfied.
func MaybeAutoRefine(reg *SkillRegistry, topic string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("skill auto-refine: panic topic=%s: %v", topic, r)
		}
	}()
	if reg == nil {
		reg = DefaultSkillRegistry()
	}
	topic = strings.ToLower(strings.TrimSpace(topic))
	if topic == "" {
		return
	}
	cfg := loadAutoRefineConfig()
	if !cfg.Enabled || !cfg.AllowsTopic(topic) {
		return
	}
	mu := autoRefineMutex(topic)
	mu.Lock()
	st, err := readAutoRefineState(reg, topic)
	if err != nil {
		mu.Unlock()
		log.Printf("skill auto-refine: read state topic=%s: %v", topic, err)
		return
	}
	today := time.Now().UTC().Format("2006-01-02")
	if st.RefinesDay != today {
		st.RefinesDay = today
		st.RefinesToday = 0
	}
	if st.SamplesSinceRefine < cfg.MinSamples {
		mu.Unlock()
		return
	}
	if !st.LastRefineAt.IsZero() && time.Since(st.LastRefineAt) < cfg.Cooldown {
		mu.Unlock()
		return
	}
	if st.RefinesToday >= cfg.MaxPerDay {
		mu.Unlock()
		return
	}
	mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	hint := "自动精炼：基于自上次 refine 以来新增的诊断样本，强化根因+证据输出格式与常见反例。"
	res, refineErr := RefineSkill(ctx, reg, RefineSkillInput{
		Topic:           topic,
		MaxSamples:      16,
		MaxFeedback:     8,
		UserHint:        hint,
		ForceLLMTimeout: 110 * time.Second,
	})
	mu.Lock()
	defer mu.Unlock()
	st, err = readAutoRefineState(reg, topic)
	if err != nil {
		log.Printf("skill auto-refine: reload state topic=%s: %v", topic, err)
		return
	}
	today = time.Now().UTC().Format("2006-01-02")
	if st.RefinesDay != today {
		st.RefinesDay = today
		st.RefinesToday = 0
	}
	if refineErr != nil {
		log.Printf("skill auto-refine: refine failed topic=%s: %v", topic, refineErr)
		return
	}
	st.LastRefineAt = time.Now().UTC()
	st.SamplesSinceRefine = 0
	st.RefinesToday++
	_ = writeAutoRefineState(reg, topic, st)
	if res != nil {
		log.Printf("skill auto-refine: ok topic=%s pack=%s path=%s samples=%d",
			topic, res.NewPack.Name, res.PersistedPath, res.SamplesUsed)
	}
}

func autoRefineMutex(topic string) *sync.Mutex {
	v, _ := autoRefineTopicLocks.LoadOrStore(topic, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func autoRefineStatePath(reg *SkillRegistry, topic string) string {
	dir := reg.DataDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "refine_state", topic+".json")
}

func readAutoRefineState(reg *SkillRegistry, topic string) (autoRefineState, error) {
	path := autoRefineStatePath(reg, topic)
	if path == "" {
		return autoRefineState{}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return autoRefineState{}, nil
		}
		return autoRefineState{}, err
	}
	var st autoRefineState
	if err := json.Unmarshal(raw, &st); err != nil {
		return autoRefineState{}, err
	}
	return st, nil
}

func writeAutoRefineState(reg *SkillRegistry, topic string, st autoRefineState) error {
	path := autoRefineStatePath(reg, topic)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

