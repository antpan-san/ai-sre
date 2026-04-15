package assets

import "embed"

// FS holds default skill packs and knowledge markdown for lightweight RAG.
//
//go:embed skills/*.yaml knowledge/*.md
var FS embed.FS
