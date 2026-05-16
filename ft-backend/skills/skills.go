// Package skills exposes the embedded built-in skill pack YAML files.
//
// The YAML files live under skills/builtin/ and are loaded by
// services.SkillRegistry. They describe how the server should structure prompts
// for each diagnostic / maintenance topic.
package skills

import "embed"

//go:embed builtin/README.md
var BuiltinFS embed.FS

// BuiltinDir is the directory inside BuiltinFS that holds the YAML files.
const BuiltinDir = "builtin"
