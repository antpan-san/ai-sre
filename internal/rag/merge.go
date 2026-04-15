package rag

// Merge concatenates chunk indexes (e.g. embedded + custom knowledge). Nil-safe.
func Merge(a, b *Index) *Index {
	if a == nil || len(a.Chunks) == 0 {
		return b
	}
	if b == nil || len(b.Chunks) == 0 {
		return a
	}
	ch := make([]Chunk, 0, len(a.Chunks)+len(b.Chunks))
	ch = append(ch, a.Chunks...)
	ch = append(ch, b.Chunks...)
	return &Index{Chunks: ch}
}
