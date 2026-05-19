package cli

import (
	"os"
	"sync"
)

var executionCtx struct {
	sync.Mutex
	correlationID string
	recordID      string
	finishMeta    map[string]interface{}
}

func setActiveExecution(correlationID, recordID string) {
	executionCtx.Lock()
	defer executionCtx.Unlock()
	executionCtx.correlationID = correlationID
	executionCtx.recordID = recordID
	_ = os.Setenv("OPSFLEET_EXECUTION_CORRELATION_ID", correlationID)
	if recordID != "" {
		_ = os.Setenv("OPSFLEET_EXECUTION_ID", recordID)
	}
}

func ActiveExecutionCorrelationID() string {
	executionCtx.Lock()
	defer executionCtx.Unlock()
	if executionCtx.correlationID != "" {
		return executionCtx.correlationID
	}
	return stringsTrim(os.Getenv("OPSFLEET_EXECUTION_CORRELATION_ID"))
}

func ActiveExecutionRecordID() string {
	executionCtx.Lock()
	defer executionCtx.Unlock()
	if executionCtx.recordID != "" {
		return executionCtx.recordID
	}
	return stringsTrim(os.Getenv("OPSFLEET_EXECUTION_ID"))
}

// MergeExecutionFinishMeta merges keys into metadata sent on execution finish.
func MergeExecutionFinishMeta(m map[string]interface{}) {
	if len(m) == 0 {
		return
	}
	executionCtx.Lock()
	defer executionCtx.Unlock()
	if executionCtx.finishMeta == nil {
		executionCtx.finishMeta = make(map[string]interface{}, len(m))
	}
	for k, v := range m {
		executionCtx.finishMeta[k] = v
	}
}

func drainExecutionFinishMeta() map[string]interface{} {
	executionCtx.Lock()
	defer executionCtx.Unlock()
	if len(executionCtx.finishMeta) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(executionCtx.finishMeta))
	for k, v := range executionCtx.finishMeta {
		out[k] = v
	}
	executionCtx.finishMeta = nil
	return out
}

func stringsTrim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
