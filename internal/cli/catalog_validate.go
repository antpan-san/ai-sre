package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

// ValidateArgvInCatalog returns true when argv resolves to a known command and flags are in scope.
func ValidateArgvInCatalog(root *cobra.Command, argv []string) bool {
	if root == nil || len(argv) == 0 {
		return false
	}
	args := append([]string(nil), argv...)
	if args[0] == "ai-sre" || strings.HasSuffix(args[0], "/ai-sre") {
		args = args[1:]
	}
	if len(args) == 0 {
		return false
	}
	if _, _, err := root.Find(args); err != nil {
		return false
	}
	if res := validateFlagsInScope(root, args, BuildCommandCatalog(root), ""); res != nil {
		return false
	}
	return true
}

// FilterCatalogValidatedArgv drops server/AI suggested argv slices that fail local catalog checks.
func FilterCatalogValidatedArgv(root *cobra.Command, candidates [][]string) [][]string {
	if root == nil {
		return nil
	}
	var out [][]string
	for _, argv := range candidates {
		if ValidateArgvInCatalog(root, argv) {
			out = append(out, argv)
		}
	}
	return out
}
