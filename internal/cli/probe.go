package cli

import (
	"github.com/spf13/cobra"
)

// probeCmd runs read-only local evidence collection without calling the LLM.
func probeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "probe",
		Short: "只读快采（无 AI）：kafka | redis | mysql | postgresql | nginx | elasticsearch | domain | linux",
		Long: `在各中间件上执行只读采集，输出结构化结论或 JSON，供人工判断或配合 check 使用。

与 check 的区别：probe 不调 LLM；check 走技能包做根因分析。`,
	}
	cmd.AddCommand(
		probeKafkaCmd(),
		probeRedisCmd(),
		probeMySQLCmd(),
		probePostgreSQLCmd(),
		probeNginxCmd(),
		probeElasticsearchCmd(),
		probeDomainCmd(),
		probeLinuxCmd(),
	)
	return cmd
}

func probeKafkaCmd() *cobra.Command {
	c := kafkaDiagnoseCmd()
	c.Use = "kafka <bootstrap-server>"
	c.Short = "Kafka 只读快采：consumer group / topic / lag"
	return c
}

func probeMySQLCmd() *cobra.Command {
	c := mysqlDiagnoseCmd()
	c.Use = "mysql <dsn>"
	c.Short = "MySQL 只读快采：连接、慢查询、线程"
	return c
}

func probePostgreSQLCmd() *cobra.Command {
	c := postgresqlDiagnoseCmd()
	c.Use = "postgresql <dsn>"
	c.Short = "PostgreSQL 只读快采：连接、事务、死锁"
	return c
}

func probeNginxCmd() *cobra.Command {
	c := nginxDiagnoseCmd()
	c.Use = "nginx"
	c.Short = "Nginx 访问日志统计：状态码、延迟、5xx"
	return c
}

func probeElasticsearchCmd() *cobra.Command {
	c := elasticsearchDiagnoseCmd()
	c.Use = "elasticsearch <url>"
	c.Short = "Elasticsearch 只读快采：集群健康与节点"
	return c
}
