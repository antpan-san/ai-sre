package loader

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/panshuai/ai-sre/internal/config"
)

// ResolveDefaultBuiltinSkillDirs returns on-disk builtin skill directories (newest paths last;
// later merges override same name in LoadSkillsAndKnowledge).
func ResolveDefaultBuiltinSkillDirs() []string {
	var dirs []string
	seen := map[string]struct{}{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		if st, err := os.Stat(p); err != nil || !st.IsDir() {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		dirs = append(dirs, p)
	}
	if v := os.Getenv("AI_SRE_BUILTIN_SKILLS_DIR"); v != "" {
		add(v)
	}
	if cfgDir, err := config.ResolveDir(); err == nil {
		add(filepath.Join(cfgDir, "builtin-skills"))
	}
	for _, rel := range []string{
		"ft-backend/skills/builtin",
		"skills/builtin",
		"internal/assets/skills",
	} {
		add(rel)
	}
	return dirs
}
