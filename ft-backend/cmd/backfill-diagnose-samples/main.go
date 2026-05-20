package main

import (
	"encoding/json"
	"fmt"
	"os"

	"ft-backend/common/config"
	"ft-backend/common/logger"
	"ft-backend/database"
	"ft-backend/services"
)

func main() {
	const configPath = "conf/config.yaml"
	if err := config.EnsureConfigExists(configPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	config.ApplyEnvOverrides(cfg)
	config.GlobalCfg = cfg
	logger.InitLogger("error", nil)
	if err := database.Connect(&cfg.Database); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer database.Close()
	reg := services.DefaultSkillRegistry()
	out, err := services.BackfillDiagnoseSamplesFromJSONL(reg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
