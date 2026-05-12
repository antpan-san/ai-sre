package cli

import (
	"encoding/json"
	"testing"
)

func TestDecodeDiagnoseResponseFromBody_Envelope(t *testing.T) {
	inner := diagnoseResponse{
		Source: "server-ai",
		Answer: "结论：先检查节点",
	}
	innerBytes, _ := json.Marshal(inner)
	wrap := map[string]interface{}{
		"code": 200,
		"msg":  "success",
		"data": json.RawMessage(innerBytes),
	}
	raw, _ := json.Marshal(wrap)
	out, err := decodeDiagnoseResponseFromBody(raw)
	if err != nil {
		t.Fatal(err)
	}
	if out.Answer != inner.Answer || out.Source != inner.Source {
		t.Fatalf("%+v", out)
	}
}
