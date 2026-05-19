package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var errParamContract = errors.New("param_contract")

// ParamSuggestion is a local-only correction hint from the command catalog.
type ParamSuggestion struct {
	Kind    string `json:"kind"` // command | flag
	Value   string `json:"value"`
	Command string `json:"command,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// ParamContractResult is emitted for argv / flag contract violations.
type ParamContractResult struct {
	OK                   bool              `json:"ok"`
	Layer                string            `json:"layer"` // param
	Message              string            `json:"message"`
	Suggestions          []ParamSuggestion `json:"suggestions,omitempty"`
	AutoIterationCreated bool              `json:"auto_iteration_created"`
	CommandCatalogDigest string            `json:"command_catalog_digest,omitempty"`
}

func isParamContractExempt(args []string) bool {
	if len(args) == 0 || isHelpInvocation() {
		return true
	}
	first := firstPositionalArg(args)
	switch first {
	case "version", "help", "completion", "doctor":
		return true
	}
	return false
}

func firstPositionalArg(args []string) string {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		return a
	}
	return ""
}

// ValidateParamContract checks argv against the local Cobra catalog before execution.
func ValidateParamContract(root *cobra.Command, args []string) *ParamContractResult {
	if root == nil || isParamContractExempt(args) {
		return nil
	}
	cat := BuildCommandCatalog(root)
	digest := cat.Digest
	if _, _, err := root.Find(args); err != nil {
		return unknownCommandResult(cat, digest, args, err)
	}
	if res := validateFlagsInScope(root, args, cat, digest); res != nil {
		return res
	}
	return nil
}

func unknownCommandResult(cat CommandCatalog, digest string, args []string, findErr error) *ParamContractResult {
	first := firstPositionalArg(args)
	msg := findErr.Error()
	if first != "" {
		msg = fmt.Sprintf("未知命令 %q", first)
	}
	return &ParamContractResult{
		OK:                   false,
		Layer:                "param",
		Message:              msg,
		Suggestions:          suggestCommands(cat, first, 3),
		AutoIterationCreated: false,
		CommandCatalogDigest: digest,
	}
}

func validateFlagsInScope(root *cobra.Command, args []string, cat CommandCatalog, digest string) *ParamContractResult {
	target, flagArgs, err := resolveCommandForFlags(root, args)
	if err != nil || target == nil {
		return nil
	}
	path := commandPath(target)
	entry := FindCatalogCommand(cat, path)
	if entry == nil {
		return nil
	}
	allowed := map[string]struct{}{}
	for _, f := range entry.Flags {
		allowed[f.Name] = struct{}{}
		if f.Shorthand != "" {
			allowed[f.Shorthand] = struct{}{}
		}
	}
	for i := 0; i < len(flagArgs); i++ {
		a := flagArgs[i]
		if !strings.HasPrefix(a, "-") {
			continue
		}
		name, _ := parseFlagToken(a)
		key := strings.TrimLeft(name, "-")
		if idx := strings.IndexByte(key, '='); idx >= 0 {
			key = key[:idx]
		}
		if key == "h" || key == "help" {
			continue
		}
		if _, ok := allowed[key]; ok {
			continue
		}
		return &ParamContractResult{
			OK:                   false,
			Layer:                "param",
			Message:              fmt.Sprintf("未知或作用域错误的 flag %q（命令 %q）", name, path),
			Suggestions:          suggestFlags(entry, key, 3),
			AutoIterationCreated: false,
			CommandCatalogDigest: digest,
		}
	}
	return nil
}

func resolveCommandForFlags(root *cobra.Command, args []string) (*cobra.Command, []string, error) {
	cmd := root
	var rest []string
	i := 0
	for i < len(args) {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			break
		}
		child := findChildCommand(cmd, a)
		if child == nil {
			rest = append(rest, args[i:]...)
			break
		}
		cmd = child
		i++
	}
	if i < len(args) {
		rest = append(rest, args[i:]...)
	}
	return cmd, rest, nil
}

func findChildCommand(parent *cobra.Command, name string) *cobra.Command {
	if parent == nil {
		return nil
	}
	for _, c := range parent.Commands() {
		if c.Name() == name || c.HasAlias(name) {
			return c
		}
	}
	return nil
}

func commandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	var parts []string
	for c := cmd; c != nil && c.Parent() != nil; c = c.Parent() {
		parts = append([]string{c.Name()}, parts...)
	}
	return strings.Join(parts, " ")
}

func parseFlagToken(a string) (name string, hasValue bool) {
	a = strings.TrimSpace(a)
	if a == "" {
		return "", false
	}
	if strings.Contains(a, "=") {
		return a, true
	}
	return a, false
}

func suggestCommands(cat CommandCatalog, typo string, limit int) []ParamSuggestion {
	type scored struct {
		path  string
		score int
	}
	typo = strings.ToLower(strings.TrimSpace(typo))
	var list []scored
	for _, c := range cat.Commands {
		if c.Hidden {
			continue
		}
		base := c.Path
		if i := strings.IndexByte(base, ' '); i > 0 {
			base = base[:i]
		}
		score := levenshtein(typo, strings.ToLower(base))
		list = append(list, scored{path: c.Path, score: score})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].score < list[j].score })
	var out []ParamSuggestion
	for _, s := range list {
		out = append(out, ParamSuggestion{Kind: "command", Value: s.path, Reason: "相似命令"})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func suggestFlags(entry *CatalogCommand, typo string, limit int) []ParamSuggestion {
	typo = strings.TrimLeft(strings.ToLower(strings.TrimSpace(typo)), "-")
	type scored struct {
		name  string
		score int
	}
	var list []scored
	for _, f := range entry.Flags {
		list = append(list, scored{name: f.Name, score: levenshtein(typo, f.Name)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].score < list[j].score })
	var out []ParamSuggestion
	for _, s := range list {
		out = append(out, ParamSuggestion{
			Kind:    "flag",
			Value:   "--" + s.name,
			Command: entry.Path,
			Reason:  "当前命令可用 flag",
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// ClassifyCobraError maps cobra execution errors to param-layer results when applicable.
func ClassifyCobraError(root *cobra.Command, args []string, err error) *ParamContractResult {
	if err == nil || errors.Is(err, errParamContract) {
		return nil
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	if !(strings.Contains(lower, "unknown command") ||
		strings.Contains(lower, "unknown flag") ||
		strings.Contains(lower, "unknown shorthand flag") ||
		strings.Contains(lower, "flag needs an argument") ||
		strings.Contains(lower, "invalid argument")) {
		return nil
	}
	cat := BuildCommandCatalog(root)
	res := &ParamContractResult{
		OK:                   false,
		Layer:                "param",
		Message:              msg,
		AutoIterationCreated: false,
		CommandCatalogDigest: cat.Digest,
	}
	if strings.Contains(lower, "unknown command") {
		res.Suggestions = suggestCommands(cat, firstPositionalArg(args), 3)
	} else if strings.Contains(lower, "unknown flag") || strings.Contains(lower, "unknown shorthand") {
		if target, _, e := resolveCommandForFlags(root, args); e == nil && target != nil {
			if entry := FindCatalogCommand(cat, commandPath(target)); entry != nil {
				res.Suggestions = suggestFlags(entry, msg, 3)
			}
		}
	}
	return res
}

func emitParamContractError(res *ParamContractResult) {
	if res == nil {
		return
	}
	if strings.EqualFold(strings.TrimSpace(outputFormat), "json") || !isStderrTTY() {
		b, _ := json.MarshalIndent(res, "", "  ")
		_, _ = fmt.Fprintln(os.Stderr, string(b))
		return
	}
	fmt.Fprintln(os.Stderr, res.Message)
	for _, s := range res.Suggestions {
		switch s.Kind {
		case "flag":
			fmt.Fprintf(os.Stderr, "  建议: %s (%s)\n", s.Value, s.Command)
		default:
			fmt.Fprintf(os.Stderr, "  建议: %s\n", s.Value)
		}
	}
	fmt.Fprintln(os.Stderr, "（参数层问题，未创建自动迭代）")
	if offer := topCommandSuggestion(res); offer != "" {
		maybeRunTTYCommandSuggestion(offer)
	}
}

func topCommandSuggestion(res *ParamContractResult) string {
	if res == nil || len(res.Suggestions) == 0 {
		return ""
	}
	best := res.Suggestions[0]
	if best.Kind != "command" || strings.TrimSpace(best.Value) == "" {
		return ""
	}
	return strings.TrimSpace(best.Value)
}

// maybeRunTTYCommandSuggestion offers to run a single high-confidence command fix (param layer only).
func maybeRunTTYCommandSuggestion(suggestedPath string) {
	if !isStdinTTY() || !isStderrTTY() {
		return
	}
	root := newRoot(progName)
	argv := strings.Fields(suggestedPath)
	if !ValidateArgvInCatalog(root, argv) {
		return
	}
	fmt.Fprintf(os.Stderr, "是否执行建议命令 %q？输入 y 回车执行: ", suggestedPath)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.ToLower(strings.TrimSpace(line)) != "y" && strings.ToLower(strings.TrimSpace(line)) != "yes" {
		return
	}
	fmt.Fprintf(os.Stderr, "正在执行: %s %s\n", progName, strings.Join(argv, " "))
	reporter := newExecutionReporter(progName, argv)
	reporter.start()
	root.SetArgs(argv)
	if err := root.Execute(); err != nil {
		reporter.finish(err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	reporter.finish(nil)
	os.Exit(0)
}
