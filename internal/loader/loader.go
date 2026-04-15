package loader

import (
	"fmt"

	"github.com/panshuai/ai-sre/internal/assets"
	"github.com/panshuai/ai-sre/internal/rag"
	"github.com/panshuai/ai-sre/internal/skill"
)

// Options configures optional on-disk skill packs and knowledge (merged with embedded defaults).
type Options struct {
	SkillsExtraDir    string
	KnowledgeExtraDir string
}

// LoadSkillsAndKnowledge loads embedded assets plus optional directories. Custom skills override same `name`.
func LoadSkillsAndKnowledge(opts Options) (*skill.Registry, *rag.Index, error) {
	sk, err := skill.LoadDir(assets.FS, "skills")
	if err != nil {
		return nil, nil, fmt.Errorf("embed skills: %w", err)
	}
	if opts.SkillsExtraDir != "" {
		extra, err := skill.LoadDirFromPath(opts.SkillsExtraDir)
		if err != nil {
			return nil, nil, err
		}
		sk = skill.MergeRegistries(sk, extra)
	}

	kb, err := rag.LoadFS(assets.FS, "knowledge")
	if err != nil {
		return nil, nil, fmt.Errorf("embed knowledge: %w", err)
	}
	if opts.KnowledgeExtraDir != "" {
		extra, err := rag.LoadDir(opts.KnowledgeExtraDir)
		if err != nil {
			return nil, nil, err
		}
		kb = rag.Merge(kb, extra)
	}
	return sk, kb, nil
}
