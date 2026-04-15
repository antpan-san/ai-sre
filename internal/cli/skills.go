package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/panshuai/ai-sre/internal/loader"
	"github.com/panshuai/ai-sre/internal/output"
)

func skillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "技能包注册表：列出或查看内置/自定义技能",
	}
	cmd.AddCommand(skillsListCmd())
	return cmd
}

func skillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出已加载的技能包（内置 + --skills-dir）",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg, _, err := loader.LoadSkillsAndKnowledge(loader.Options{
				SkillsExtraDir:    skillsExtraDir,
				KnowledgeExtraDir: "",
			})
			if err != nil {
				return err
			}
			if strings.EqualFold(outputFormat, "json") {
				return output.PrintSkillsJSON(os.Stdout, reg)
			}
			fmt.Println("Skill packs:")
			output.PrintSkillsTable(os.Stdout, reg)
			return nil
		},
	}
}
