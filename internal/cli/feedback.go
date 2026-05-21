package cli

import "context"

// maybePromptFeedback 已移除交互式反馈；仍可通过 ai-sre expert skills feedback 主动提交。
func maybePromptFeedback(_ context.Context, _ string, _ *diagnoseResponse) {}
