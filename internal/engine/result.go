package engine

// RunResult is LLM output plus skill metadata for structured JSON and tooling.
type RunResult struct {
	Answer       string
	SkillName    string
	SkillDisplay string
	SkillTopics  []string
}
