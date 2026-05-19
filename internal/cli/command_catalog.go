package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CatalogFlag describes a flag available on a command path (local or inherited persistent).
type CatalogFlag struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand,omitempty"`
	Scope     string `json:"scope"` // local | persistent
	Type      string `json:"type,omitempty"`
	Required  bool   `json:"required,omitempty"`
}

// CatalogCommand is one node in the exported CLI command tree.
type CatalogCommand struct {
	Path       string        `json:"path"`
	Use        string        `json:"use,omitempty"`
	MinArgs    int           `json:"min_args,omitempty"`
	MaxArgs    int           `json:"max_args,omitempty"`
	Flags      []CatalogFlag `json:"flags,omitempty"`
	Hidden     bool          `json:"hidden,omitempty"`
	Deprecated bool          `json:"deprecated,omitempty"`
}

// CommandCatalog is the CLI-local contract sent to fulfillment plan (digest only on wire).
type CommandCatalog struct {
	CLIVersion string             `json:"cli_version"`
	Commands   []CatalogCommand   `json:"commands"`
	Digest     string             `json:"digest"`
}

// BuildCommandCatalog walks the Cobra tree for the active program name.
func BuildCommandCatalog(root *cobra.Command) CommandCatalog {
	var cmds []CatalogCommand
	walkCatalogCommands(root, "", &cmds)
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Path < cmds[j].Path })
	cat := CommandCatalog{
		CLIVersion: strings.TrimSpace(Version),
		Commands:   cmds,
	}
	cat.Digest = catalogDigest(cat)
	return cat
}

func walkCatalogCommands(cmd *cobra.Command, parentPath string, out *[]CatalogCommand) {
	if cmd == nil {
		return
	}
	path := strings.TrimSpace(parentPath)
	if cmd.Parent() != nil {
		if path == "" {
			path = strings.TrimSpace(cmd.Name())
		} else if cmd.Name() != "" {
			path = path + " " + cmd.Name()
		}
	}
	if cmd.Parent() != nil && path != "" && !cmd.Hidden {
		entry := CatalogCommand{
			Path:       path,
			Use:        strings.TrimSpace(cmd.Use),
			Hidden:     cmd.Hidden,
			Deprecated: cmd.Deprecated != "",
		}
		if cmd.Args != nil {
			// Best-effort: cobra does not expose min/max directly.
			if strings.Contains(cmd.Use, "[") {
				entry.MinArgs = 0
			}
		}
		entry.Flags = collectCatalogFlags(cmd)
		*out = append(*out, entry)
	}
	for _, child := range cmd.Commands() {
		walkCatalogCommands(child, path, out)
	}
}

func collectCatalogFlags(cmd *cobra.Command) []CatalogFlag {
	seen := map[string]struct{}{}
	var flags []CatalogFlag
	add := func(fs *pflag.FlagSet, scope string) {
		if fs == nil {
			return
		}
		fs.VisitAll(func(f *pflag.Flag) {
			if f == nil || f.Hidden {
				return
			}
			key := scope + ":" + f.Name
			if _, ok := seen[key]; ok {
				return
			}
			seen[key] = struct{}{}
			cf := CatalogFlag{
				Name:     f.Name,
				Scope:    scope,
				Type:     f.Value.Type(),
				Required: false,
			}
			if f.Shorthand != "" && f.Shorthand != " " {
				cf.Shorthand = f.Shorthand
			}
			flags = append(flags, cf)
		})
	}
	for p := cmd; p != nil; p = p.Parent() {
		add(p.PersistentFlags(), "persistent")
	}
	add(cmd.Flags(), "local")
	sort.Slice(flags, func(i, j int) bool {
		if flags[i].Scope == flags[j].Scope {
			return flags[i].Name < flags[j].Name
		}
		return flags[i].Scope < flags[j].Scope
	})
	return flags
}

func catalogDigest(cat CommandCatalog) string {
	payload := struct {
		CLIVersion string           `json:"cli_version"`
		Commands   []CatalogCommand `json:"commands"`
	}{
		CLIVersion: cat.CLIVersion,
		Commands:   cat.Commands,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// CommandCatalogDigest returns the digest for the current program tree.
func CommandCatalogDigest(root *cobra.Command) string {
	return BuildCommandCatalog(root).Digest
}

// FindCatalogCommand returns the catalog entry for a command path like "check k8s".
func FindCatalogCommand(cat CommandCatalog, path string) *CatalogCommand {
	path = strings.TrimSpace(path)
	for i := range cat.Commands {
		if cat.Commands[i].Path == path {
			return &cat.Commands[i]
		}
	}
	return nil
}
