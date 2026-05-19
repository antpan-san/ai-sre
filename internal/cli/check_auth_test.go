package cli

import "testing"

func TestInjectMySQLPassword(t *testing.T) {
	t.Parallel()
	dsn := "root@tcp(127.0.0.1:3306)/"
	got := injectMySQLPassword(dsn, "secret")
	want := "root:secret@tcp(127.0.0.1:3306)/"
	if got != want {
		t.Fatalf("injectMySQLPassword()=%q want %q", got, want)
	}
}

func TestInjectPostgreSQLPassword(t *testing.T) {
	t.Parallel()
	dsn := "postgres://user@127.0.0.1:5432/db?sslmode=disable"
	got := injectPostgreSQLPassword(dsn, "secret")
	if !postgresqlDSNHasPassword(got) {
		t.Fatalf("expected password in %q", got)
	}
}

func TestIsKafkaAuthLikely(t *testing.T) {
	t.Parallel()
	if !isKafkaAuthLikely("SASL authentication failed") {
		t.Fatal("expected kafka auth likely")
	}
	if isKafkaAuthLikely("connection refused") {
		t.Fatal("connection refused alone should not imply auth config required")
	}
}
