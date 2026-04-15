package engine

import (
	"context"
	"io/fs"

	"github.com/panshuai/ai-sre/internal/llm"
	"github.com/panshuai/ai-sre/internal/prompt"
	"github.com/panshuai/ai-sre/internal/rag"
	"github.com/panshuai/ai-sre/internal/skill"
)

const systemSRE = `你是 AI SRE Copilot：专注线上稳定性、可观测性与安全变更。
回答必须可执行、可验证；不要编造具体版本号或内部不存在的命令输出。`

// Engine ties skills, prompts, optional RAG, and LLM.
type Engine struct {
	Skills *skill.Registry
	RAG    *rag.Index
	LLM    *llm.Client
}

// Analyze runs fault diagnosis for a topic with key-value context.
func (e *Engine) Analyze(ctx context.Context, topic string, context map[string]string, useRAG bool) (string, error) {
	sp := e.Skills.MatchAnalyze(topic, context)
	var ragText string
	if useRAG && e.RAG != nil {
		q := topic
		for k, v := range context {
			q += " " + k + " " + v
		}
		chunks := e.RAG.Search(q, 4)
		ragText = rag.FormatChunks(chunks)
	}
	user := prompt.BuildAnalyze(sp, topic, context, ragText)
	return e.LLM.Chat(ctx, systemSRE, user)
}

// Ask answers a free-form question; optional skill hint + RAG.
func (e *Engine) Ask(ctx context.Context, question string, useRAG bool) (string, error) {
	sp := e.Skills.MatchQuery(question)
	var ragText string
	if useRAG && e.RAG != nil {
		chunks := e.RAG.Search(question, 5)
		ragText = rag.FormatChunks(chunks)
	}
	user := prompt.BuildAsk(question, ragText, sp)
	return e.LLM.Chat(ctx, systemSRE, user)
}

// Runbook generates a runbook for the given scenario description.
func (e *Engine) Runbook(ctx context.Context, scenario string, context map[string]string, useRAG bool) (string, error) {
	sp := e.Skills.MatchQuery(scenario)
	var ragText string
	if useRAG && e.RAG != nil {
		q := scenario
		for k, v := range context {
			q += " " + k + " " + v
		}
		chunks := e.RAG.Search(q, 5)
		ragText = rag.FormatChunks(chunks)
	}
	user := prompt.BuildRunbook(scenario, sp, context, ragText)
	return e.LLM.Chat(ctx, systemSRE, user)
}

// LoadSkillsFromFS loads skill registry from fs.FS path "skills".
func LoadSkillsFromFS(fsys fs.FS) (*skill.Registry, error) {
	return skill.LoadDir(fsys, "skills")
}

// LoadKnowledgeFromFS loads RAG index from fs.FS path "knowledge".
func LoadKnowledgeFromFS(fsys fs.FS) (*rag.Index, error) {
	return rag.LoadFS(fsys, "knowledge")
}
