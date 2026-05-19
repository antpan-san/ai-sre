package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
)

func TestServerAIFallbackEligible_network(t *testing.T) {
	cases := []error{
		context.DeadlineExceeded,
		&net.OpError{Op: "dial", Err: errors.New("connection refused")},
		fmt.Errorf("call server diagnose: dial tcp: i/o timeout"),
		fmt.Errorf("server ask status=503: unavailable"),
	}
	for _, err := range cases {
		if !serverAIFallbackEligible(err) {
			t.Fatalf("expected eligible: %v", err)
		}
	}
}

func TestServerAIFallbackEligible_notAuthOrBusiness(t *testing.T) {
	cases := []error{
		fmt.Errorf("cli sync status=401: CLI token 无效"),
		fmt.Errorf("server diagnose status=401: CLI token 无效"),
		fmt.Errorf("api code=402 msg=额度不足"),
		errors.New("能力不可执行"),
		context.Canceled,
	}
	for _, err := range cases {
		if serverAIFallbackEligible(err) {
			t.Fatalf("expected not eligible: %v", err)
		}
	}
}
