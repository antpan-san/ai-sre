package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/panshuai/ai-sre/internal/engine"
	"github.com/panshuai/ai-sre/internal/skill"
)

// Payload is one CLI invocation result for JSON / automation.
type Payload struct {
	Command    string            `json:"command"`
	Topic      string            `json:"topic,omitempty"`
	Question   string            `json:"question,omitempty"`
	Scenario   string            `json:"scenario,omitempty"`
	Context    map[string]string `json:"context,omitempty"`
	RAG        bool              `json:"rag"`
	DurationMs int64             `json:"duration_ms"`
	Skill      *SkillJSON        `json:"skill,omitempty"`
	Answer     string            `json:"answer"`
}

// SkillJSON is a subset of skill pack for structured output.
type SkillJSON struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Topics      []string `json:"topics,omitempty"`
}

// BuildPayload fills a payload from engine result.
func BuildPayload(command, topic, question, scenario string, ctx map[string]string, ragOn bool, durMs int64, r *engine.RunResult) Payload {
	p := Payload{
		Command:    command,
		Topic:      topic,
		Question:   question,
		Scenario:   scenario,
		Context:    ctx,
		RAG:        ragOn,
		DurationMs: durMs,
	}
	if r != nil {
		p.Answer = r.Answer
		if r.SkillName != "" || r.SkillDisplay != "" {
			p.Skill = &SkillJSON{
				Name:        r.SkillName,
				DisplayName: r.SkillDisplay,
				Topics:      append([]string(nil), r.SkillTopics...),
			}
		}
	}
	return p
}

// WriteJSON prints one pretty-printed JSON object to w.
func WriteJSON(w io.Writer, p Payload) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(p)
}

// Print writes text or JSON depending on format ("text" | "json").
func Print(format string, p Payload) error {
	if strings.EqualFold(format, "json") {
		return WriteJSON(os.Stdout, p)
	}
	fmt.Println(p.Answer)
	return nil
}

// PrintSkillsTable writes skill registry as a text table.
func PrintSkillsTable(w io.Writer, reg *skill.Registry) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tDISPLAY\tTOPICS")
	for _, p := range reg.Packs {
		topics := strings.Join(p.Topics, ",")
		if topics == "" {
			topics = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", p.Name, p.DisplayName, topics)
	}
	tw.Flush()
}

// PrintSkillsJSON writes skill list as JSON array.
func PrintSkillsJSON(w io.Writer, reg *skill.Registry) error {
	type row struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"display_name"`
		Topics      []string `json:"topics,omitempty"`
		Keywords    []string `json:"match_keywords,omitempty"`
	}
	var rows []row
	for _, p := range reg.Packs {
		rows = append(rows, row{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Topics:      p.Topics,
			Keywords:    p.MatchKeywords,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}
