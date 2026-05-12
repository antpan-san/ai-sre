package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestElasticsearchNormalizeBase(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"127.0.0.1:9200", "http://127.0.0.1:9200"},
		{"http://h:9200/", "http://h:9200"},
		{"https://es:9243/path/ignored", "https://es:9243"},
	} {
		u, err := elasticsearchNormalizeBase(tc.in)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if got := elasticsearchBaseString(u); got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestRunElasticsearchDiagnose_YellowSingleNode(t *testing.T) {
	health := `{"cluster_name":"c1","status":"yellow","timed_out":false,"number_of_nodes":1,"number_of_data_nodes":1,"active_primary_shards":3,"active_shards":3,"relocating_shards":0,"initializing_shards":0,"unassigned_shards":3,"delayed_unassigned_shards":0,"number_of_pending_tasks":0,"active_shards_percent_as_number":100}`
	nodes := `[{"name":"n1","heap.percent":"50","disk.used_percent":"10","node.roles":"cdfhilmrstw"}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_cluster/health":
			fmt.Fprint(w, health)
		case "/_cat/nodes":
			fmt.Fprint(w, nodes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	rep, err := runElasticsearchDiagnose(context.Background(), elasticsearchDiagnoseOptions{
		BaseURL: srv.URL,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Health == nil || rep.Health.Status != "yellow" {
		t.Fatalf("health: %+v", rep.Health)
	}
	if rep.RawSummary.Mode != "single-node" {
		t.Fatalf("mode=%q", rep.RawSummary.Mode)
	}
	foundYellow := false
	for _, f := range rep.Findings {
		if strings.Contains(f.Title, "黄态") {
			foundYellow = true
			break
		}
	}
	if !foundYellow {
		t.Fatalf("expected yellow single-node finding, got %#v", rep.Findings)
	}
}

func TestRunElasticsearchDiagnose_Red(t *testing.T) {
	health := `{"cluster_name":"c1","status":"red","timed_out":false,"number_of_nodes":3,"number_of_data_nodes":3,"active_primary_shards":1,"active_shards":1,"relocating_shards":0,"initializing_shards":1,"unassigned_shards":2,"delayed_unassigned_shards":0,"number_of_pending_tasks":0,"active_shards_percent_as_number":40}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_cluster/health" {
			fmt.Fprint(w, health)
			return
		}
		if r.URL.Path == "/_cat/nodes" {
			fmt.Fprint(w, `[]`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	rep, err := runElasticsearchDiagnose(context.Background(), elasticsearchDiagnoseOptions{
		BaseURL: srv.URL,
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Findings) == 0 || rep.Findings[0].Severity != "P0" {
		t.Fatalf("findings: %#v", rep.Findings)
	}
}
