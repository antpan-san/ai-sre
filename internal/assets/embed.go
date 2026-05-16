package assets

import "embed"

// FS holds knowledge markdown for lightweight RAG. Skill pack YAML is a private
// core asset (not in git); load from disk via loader.ResolveDefaultBuiltinSkillDirs.
//
//go:embed skills/README.md knowledge/*.md
var FS embed.FS
