package db

import (
	"testing"

	"github.com/platform-smith-labs/japi-core/v4/internal/sqltrace"
)

func TestOpenConnector_TracingOptIn(t *testing.T) {
	const dsn = "host=localhost port=5432 user=u password=p dbname=d sslmode=disable"

	plain, err := openConnector(Config{}, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if sqltrace.IsWrapped(plain) {
		t.Error("connector is wrapped with Tracing off")
	}

	traced, err := openConnector(Config{Tracing: true}, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !sqltrace.IsWrapped(traced) {
		t.Error("connector is not wrapped with Tracing on")
	}
}
