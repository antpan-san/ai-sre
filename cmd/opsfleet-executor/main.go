// Command opsfleet-executor is the OpsFleet "local executor" binary: same CLI semantics as ai-sre
// (skill packs, analyze / ask / runbook, RAG, LLM), intended for deployment on managed hosts.
package main

import "github.com/panshuai/ai-sre/internal/cli"

func main() {
	cli.ExecuteAs("opsfleet-executor")
}
