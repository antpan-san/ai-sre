package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseKafkaConsumerGroupDescribe_HighLagHotPartition(t *testing.T) {
	out := `GROUP           TOPIC   PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG     CONSUMER-ID  HOST        CLIENT-ID
order-service   orders  0          100             200             100     member-1     /10.0.0.2  c1
order-service   orders  3          100             100100          100000  member-2     /10.0.0.3  c2
`
	g := parseKafkaConsumerGroupDescribe("order-service", out)
	if g.TotalLag != 100100 {
		t.Fatalf("TotalLag=%d want 100100", g.TotalLag)
	}
	if g.MaxLagTopic != "orders" || g.MaxLagPartition != 3 || g.MaxPartitionLag != 100000 {
		t.Fatalf("max lag mismatch: topic=%s partition=%d lag=%d", g.MaxLagTopic, g.MaxLagPartition, g.MaxPartitionLag)
	}
	if g.ActiveMembers != 2 {
		t.Fatalf("ActiveMembers=%d want 2", g.ActiveMembers)
	}
}

func TestParseKafkaTopicsDescribe_OfflineAndUnderReplicated(t *testing.T) {
	out := `Topic: orders TopicId: abc PartitionCount: 2 ReplicationFactor: 3 Configs:
Topic: orders Partition: 0 Leader: -1 Replicas: 1,2,3 Isr: 1,2
Topic: orders Partition: 1 Leader: 1 Replicas: 1,2,3 Isr: 1,2,3
`
	topics := parseKafkaTopicsDescribe(out)
	if len(topics) != 1 {
		t.Fatalf("topics=%d want 1", len(topics))
	}
	if topics[0].OfflinePartitions != 1 {
		t.Fatalf("OfflinePartitions=%d want 1", topics[0].OfflinePartitions)
	}
	if topics[0].UnderReplicatedPartitions != 1 {
		t.Fatalf("UnderReplicatedPartitions=%d want 1", topics[0].UnderReplicatedPartitions)
	}
}

func TestDiagnoseKafkaSnapshot_PrioritizesOfflinePartition(t *testing.T) {
	s := kafkaSnapshot{
		BootstrapServer: "10.0.0.1:9092",
		Topics: []kafkaTopicSummary{{
			Name:                      "orders",
			OfflinePartitions:         1,
			UnderReplicatedPartitions: 1,
		}},
		Groups: []kafkaGroupSummary{{
			Name:            "order-service",
			TotalLag:        100000,
			MaxPartitionLag: 90000,
			MaxLagTopic:     "orders",
			MaxLagPartition: 3,
			Partitions:      12,
			ActiveMembers:   2,
		}},
	}
	findings := diagnoseKafkaSnapshot(s)
	if len(findings) == 0 {
		t.Fatal("expected findings")
	}
	if findings[0].Severity != "P0" || !strings.Contains(findings[0].Title, "offline partition") {
		t.Fatalf("top finding=%+v, want P0 offline partition", findings[0])
	}
}

func TestRunKafkaDiagnose_FakeKafkaCLI(t *testing.T) {
	dir := t.TempDir()
	writeFakeKafkaScript(t, dir, "kafka-consumer-groups.sh", `#!/bin/sh
case "$*" in
  *"--list"*)
    echo "order-service"
    echo "payment-service"
    ;;
  *"--state --group order-service"*)
    echo "GROUP COORDINATOR (ID) ASSIGNMENT-STRATEGY STATE #MEMBERS"
    echo "order-service broker-1 range Stable 2"
    ;;
  *"--state --group payment-service"*)
    echo "GROUP COORDINATOR (ID) ASSIGNMENT-STRATEGY STATE #MEMBERS"
    echo "payment-service broker-1 range Stable 0"
    ;;
  *"--describe --group order-service"*)
    echo "GROUP TOPIC PARTITION CURRENT-OFFSET LOG-END-OFFSET LAG CONSUMER-ID HOST CLIENT-ID"
    echo "order-service orders 0 100 200 100 member-1 /10.0.0.2 c1"
    echo "order-service orders 3 100 100100 100000 member-2 /10.0.0.3 c2"
    ;;
  *"--describe --group payment-service"*)
    echo "Consumer group 'payment-service' has no active members."
    echo "GROUP TOPIC PARTITION CURRENT-OFFSET LOG-END-OFFSET LAG CONSUMER-ID HOST CLIENT-ID"
    echo "payment-service payments 0 0 5000 5000 - - -"
    ;;
esac
`)
	writeFakeKafkaScript(t, dir, "kafka-topics.sh", `#!/bin/sh
echo "Topic: orders TopicId: abc PartitionCount: 2 ReplicationFactor: 3 Configs:"
echo "Topic: orders Partition: 0 Leader: 1 Replicas: 1,2,3 Isr: 1,2,3"
echo "Topic: orders Partition: 1 Leader: 1 Replicas: 1,2,3 Isr: 1,2,3"
`)
	writeFakeKafkaScript(t, dir, "kafka-broker-api-versions.sh", `#!/bin/sh
echo "broker-1 -> ok"
`)

	report, err := runKafkaDiagnose(context.Background(), kafkaDiagnoseOptions{
		BootstrapServer: "10.0.0.1:9092",
		Limit:           20,
		Timeout:         2 * time.Second,
		CommandDir:      dir,
	})
	if err != nil {
		t.Fatalf("runKafkaDiagnose: %v", err)
	}
	if report.GroupsScanned != 2 {
		t.Fatalf("GroupsScanned=%d want 2", report.GroupsScanned)
	}
	if len(report.Findings) == 0 {
		t.Fatal("expected findings")
	}
	if report.Findings[0].Severity != "P1" || !strings.Contains(report.Findings[0].Title, "order-service") {
		t.Fatalf("top finding=%+v, want P1 order-service lag", report.Findings[0])
	}
	text := formatKafkaDiagnoseText(report)
	if !strings.Contains(text, "结论：") || !strings.Contains(text, "最快验证：") {
		t.Fatalf("text output missing concise sections:\n%s", text)
	}
}

func writeFakeKafkaScript(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0755); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
