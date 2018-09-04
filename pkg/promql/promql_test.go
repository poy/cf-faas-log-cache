package promql_test

import (
	"reflect"
	"testing"

	"github.com/apoydence/cf-faas-log-cache/pkg/promql"
)

func TestPromQLParse(t *testing.T) {
	t.Parallel()
	sIDs, err := promql.Parse(`metric{source_id="a"}+metric{source_id="b"}`)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual([]string{"a", "b"}, sIDs) {
		t.Fatalf("wrong: %v", sIDs)
	}
}

func TestPromQLInvalid(t *testing.T) {
	t.Parallel()
	_, err := promql.Parse(`invalid.query`)
	if err == nil {
		t.Fatal("expected an error")
	}
}
