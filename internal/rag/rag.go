package rag

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Chunk is a retrievable text segment with source label.
type Chunk struct {
	Source string
	Text   string
}

// Index holds embedded knowledge chunks.
type Index struct {
	Chunks []Chunk
}

// LoadFS reads *.md under dir, splits into paragraphs for retrieval.
func LoadFS(fsys fs.FS, dir string) (*Index, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read knowledge dir: %w", err)
	}
	var chunks []Chunk
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		parts := splitParagraphs(string(b))
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if len(t) < 40 {
				continue
			}
			chunks = append(chunks, Chunk{Source: path, Text: t})
		}
	}
	return &Index{Chunks: chunks}, nil
}

var paraBreak = regexp.MustCompile(`\n\s*\n`)

func splitParagraphs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return paraBreak.Split(s, -1)
}

// tokenize very light CN/EN split for scoring.
func tokenize(q string) []string {
	q = strings.ToLower(q)
	var runes []rune
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			runes = append(runes, r)
		} else {
			runes = append(runes, ' ')
		}
	}
	fields := strings.Fields(string(runes))
	seen := map[string]struct{}{}
	var out []string
	for _, f := range fields {
		if len(f) < 2 {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

type scored struct {
	i int
	s float64
}

// Search returns top-k chunks by simple token overlap score (lightweight RAG without embeddings).
func (idx *Index) Search(query string, k int) []Chunk {
	if idx == nil || len(idx.Chunks) == 0 {
		return nil
	}
	toks := tokenize(query)
	if len(toks) == 0 {
		return nil
	}
	var scores []scored
	for i := range idx.Chunks {
		chunkLower := strings.ToLower(idx.Chunks[i].Text)
		var score float64
		for _, t := range toks {
			if strings.Contains(chunkLower, t) {
				score += 1.0
			}
		}
		// slight boost for title-like first line match
		firstLine := strings.ToLower(strings.Split(idx.Chunks[i].Text, "\n")[0])
		for _, t := range toks {
			if strings.Contains(firstLine, t) {
				score += 0.5
			}
		}
		if score > 0 {
			scores = append(scores, scored{i: i, s: score})
		}
	}
	sort.Slice(scores, func(a, b int) bool {
		if scores[a].s == scores[b].s {
			return scores[a].i < scores[b].i
		}
		return scores[a].s > scores[b].s
	})
	if k <= 0 {
		k = 3
	}
	var out []Chunk
	for j := 0; j < len(scores) && j < k; j++ {
		out = append(out, idx.Chunks[scores[j].i])
	}
	return out
}

// FormatChunks for injection into prompts.
func FormatChunks(chunks []Chunk) string {
	if len(chunks) == 0 {
		return ""
	}
	var b strings.Builder
	for i, c := range chunks {
		b.WriteString(fmt.Sprintf("--- 片段 %d (%s) ---\n", i+1, c.Source))
		b.WriteString(c.Text)
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}
