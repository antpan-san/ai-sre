package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// argvHasUnresolvedSubcommand reports whether argv names a subcommand that does not exist
// in the current binary (e.g. old ai-sre without `probe linux`). Cobra Find() alone stops at
// the parent command and leaves the unknown token in remaining args — preflight must treat
// that as "unknown command" and attempt auto-upgrade before showing parent help.
func argvHasUnresolvedSubcommand(root *cobra.Command, args []string) bool {
	if root == nil || len(args) == 0 {
		return false
	}
	cmd, remaining, err := root.Find(args)
	if err != nil {
		return true
	}
	return cmdHasUnresolvedSubcommand(cmd, remaining)
}

func cmdHasUnresolvedSubcommand(cmd *cobra.Command, remaining []string) bool {
	if cmd == nil {
		return false
	}
	word, rest := firstPositionalAfterFlags(remaining)
	if word == "" {
		return false
	}
	if !cmd.HasAvailableSubCommands() {
		return false
	}
	child := findChildCommand(cmd, word)
	if child == nil {
		return true
	}
	_, rest2, err := child.Find(rest)
	if err != nil {
		return true
	}
	return cmdHasUnresolvedSubcommand(child, rest2)
}

func firstPositionalAfterFlags(args []string) (word string, rest []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			if strings.Contains(a, "=") {
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
			continue
		}
		return a, args[i+1:]
	}
	return "", nil
}
